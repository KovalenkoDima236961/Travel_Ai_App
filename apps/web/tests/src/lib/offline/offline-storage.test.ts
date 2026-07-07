import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/shared/api/client";
import {
  cacheTripSnapshot,
  clearOfflineData,
  clearOfflineDataForUser,
  deleteCachedTrip,
  getCachedTrip,
  listCachedTrips
} from "@/lib/offline/trip-cache";
import {
  discardMutation,
  enqueueItineraryUpdate,
  getPendingMutations,
  markMutationFailed,
  syncPendingMutations
} from "@/lib/offline/sync-queue";
import type { BudgetSummary } from "@/entities/budget/model";
import type { Itinerary, Trip } from "@/entities/trip/model";

const dbState = vi.hoisted(() => ({
  stores: {
    cachedTrips: new Map<string, unknown>(),
    pendingMutations: new Map<string, unknown>(),
    syncMetadata: new Map<string, unknown>()
  }
}));

const tripApi = vi.hoisted(() => ({
  getTrip: vi.fn(),
  updateTripItinerary: vi.fn()
}));

vi.mock("@/lib/offline/db", () => ({
  getOfflineDb: async () => ({
    put: async (storeName: keyof typeof dbState.stores, value: Record<string, unknown>) => {
      dbState.stores[storeName].set(keyForStore(storeName, value), clone(value));
    },
    get: async (storeName: keyof typeof dbState.stores, key: string) =>
      clone(dbState.stores[storeName].get(key)),
    getAll: async (storeName: keyof typeof dbState.stores) =>
      Array.from(dbState.stores[storeName].values()).map(clone),
    delete: async (storeName: keyof typeof dbState.stores, key: string) => {
      dbState.stores[storeName].delete(key);
    },
    clear: async (storeName: keyof typeof dbState.stores) => {
      dbState.stores[storeName].clear();
    }
  })
}));

vi.mock("@/lib/api/trips", () => tripApi);

beforeEach(() => {
  Object.values(dbState.stores).forEach((store) => store.clear());
  tripApi.getTrip.mockReset();
  tripApi.updateTripItinerary.mockReset();
});

describe("offline trip cache", () => {
  it("stores cached trip snapshots by user and clears them on logout", async () => {
    await cacheTripSnapshot({
      userId: "user-1",
      trip: sampleTrip(),
      budgetSummary: sampleBudgetSummary()
    });

    const cached = await getCachedTrip("trip-1", "user-1");
    expect(cached?.trip.destination).toBe("Rome");
    expect(cached?.budgetSummary?.estimatedTotal).toBe(42);
    expect(await getCachedTrip("trip-1", "user-2")).toBeNull();

    await clearOfflineData("user-1");

    expect(await getCachedTrip("trip-1", "user-1")).toBeNull();
  });

  it("lists and deletes cached trips only for the current user", async () => {
    await cacheTripSnapshot({
      userId: "user-1",
      trip: { ...sampleTrip(), id: "trip-1", destination: "Rome" }
    });
    await cacheTripSnapshot({
      userId: "user-1",
      trip: { ...sampleTrip(), id: "trip-2", destination: "Paris" }
    });
    await cacheTripSnapshot({
      userId: "user-2",
      trip: { ...sampleTrip(), id: "trip-3", userId: "user-2", destination: "Lisbon" }
    });

    expect(await listCachedTrips("user-1")).toHaveLength(2);
    expect(
      (await listCachedTrips("user-1"))
        .map((record) => record.trip.destination)
        .sort()
    ).toEqual(["Paris", "Rome"]);

    await deleteCachedTrip("trip-3", "user-1");
    expect(await getCachedTrip("trip-3", "user-2")).not.toBeNull();

    await deleteCachedTrip("trip-1", "user-1");
    expect((await listCachedTrips("user-1")).map((record) => record.tripId)).toEqual(["trip-2"]);

    await clearOfflineDataForUser("user-1");
    expect(await listCachedTrips("user-1")).toHaveLength(0);
    expect(await getCachedTrip("trip-3", "user-2")).not.toBeNull();
  });
});

describe("offline itinerary queue", () => {
  it("coalesces multiple itinerary edits for the same trip", async () => {
    const first = await enqueueItineraryUpdate({
      tripId: "trip-1",
      userId: "user-1",
      baseRevision: 7,
      baseItinerary: itinerary("Base"),
      draftItinerary: itinerary("Draft 1")
    });

    const second = await enqueueItineraryUpdate({
      tripId: "trip-1",
      userId: "user-1",
      baseRevision: 9,
      baseItinerary: itinerary("New base should not replace original"),
      draftItinerary: itinerary("Draft 2")
    });

    expect(second.mutationId).toBe(first.mutationId);
    expect(second.baseRevision).toBe(7);
    expect(second.baseItinerary.days[0].items[0].name).toBe("Base");
    expect(second.draftItinerary.days[0].items[0].name).toBe("Draft 2");
    expect(await getPendingMutations("user-1")).toHaveLength(1);
  });

  it("marks failed and discarded mutations outside the pending set", async () => {
    const mutation = await enqueueItineraryUpdate({
      tripId: "trip-1",
      userId: "user-1",
      baseRevision: 7,
      baseItinerary: itinerary("Base"),
      draftItinerary: itinerary("Draft")
    });

    const failed = await markMutationFailed(mutation.mutationId, {
      code: "forbidden",
      message: "No permission"
    });

    expect(failed?.status).toBe("failed");
    expect((await getPendingMutations("user-1"))[0].errorCode).toBe("forbidden");

    await discardMutation(mutation.mutationId);

    expect(await getPendingMutations("user-1")).toHaveLength(0);
  });

  it("syncs pending itinerary edits and updates the cached trip", async () => {
    await enqueueItineraryUpdate({
      tripId: "trip-1",
      userId: "user-1",
      baseRevision: 7,
      baseItinerary: itinerary("Base"),
      draftItinerary: itinerary("Draft")
    });
    tripApi.updateTripItinerary.mockResolvedValue({
      ...sampleTrip(),
      itinerary: itinerary("Draft"),
      itineraryRevision: 8
    });

    const results = await syncPendingMutations("user-1");

    expect(tripApi.updateTripItinerary).toHaveBeenCalledWith(
      "trip-1",
      itinerary("Draft"),
      7
    );
    expect(results[0].status).toBe("synced");
    expect(await getPendingMutations("user-1")).toHaveLength(0);
    expect((await getCachedTrip("trip-1", "user-1"))?.itineraryRevision).toBe(8);
  });

  it("keeps local drafts when sync hits an itinerary conflict", async () => {
    await enqueueItineraryUpdate({
      tripId: "trip-1",
      userId: "user-1",
      baseRevision: 7,
      baseItinerary: itinerary("Base"),
      draftItinerary: itinerary("Local")
    });
    tripApi.updateTripItinerary.mockRejectedValue(
      new ApiError("Conflict", 409, undefined, "itinerary_conflict", 8)
    );
    tripApi.getTrip.mockResolvedValue({
      ...sampleTrip(),
      itinerary: itinerary("Latest"),
      itineraryRevision: 8
    });

    const results = await syncPendingMutations("user-1");

    expect(results[0].status).toBe("conflict");
    if (results[0].status !== "conflict") {
      throw new Error("expected conflict");
    }
    expect(results[0].latestTrip?.itinerary?.days[0].items[0].name).toBe("Latest");
    expect((await getPendingMutations("user-1"))[0].status).toBe("conflict");
    expect((await getPendingMutations("user-1"))[0].draftItinerary.days[0].items[0].name).toBe(
      "Local"
    );
  });
});

function keyForStore(storeName: keyof typeof dbState.stores, value: Record<string, unknown>) {
  if (storeName === "cachedTrips") {
    return String(value.tripId);
  }
  if (storeName === "pendingMutations") {
    return String(value.mutationId);
  }
  return String(value.key);
}

function clone<T>(value: T): T {
  if (value == null) {
    return value;
  }
  return JSON.parse(JSON.stringify(value)) as T;
}

function itinerary(itemName: string): Itinerary {
  return {
    days: [
      {
        day: 1,
        title: "Day 1",
        items: [{ time: "09:00", type: "activity", name: itemName }]
      }
    ]
  };
}

function sampleTrip(): Trip {
  return {
    id: "trip-1",
    userId: "user-1",
    destination: "Rome",
    days: 1,
    budgetCurrency: "EUR",
    travelers: 1,
    interests: [],
    pace: "balanced",
    status: "COMPLETED",
    itinerary: itinerary("Base"),
    itineraryRevision: 7,
    createdAt: "2026-06-25T00:00:00Z",
    updatedAt: "2026-06-25T00:00:00Z"
  };
}

function sampleBudgetSummary(): BudgetSummary {
  return {
    currency: "EUR",
    tripBudget: 100,
    estimatedTotal: 42,
    remaining: 58,
    missingEstimateCount: 0,
    estimatedItemCount: 1,
    byDay: [],
    byCategory: []
  };
}
