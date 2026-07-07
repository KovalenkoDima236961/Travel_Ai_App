import { afterEach, describe, expect, it, vi } from "vitest";
import { isItineraryConflictError } from "@/shared/api/client";
import {
  generateItinerary,
  regenerateItineraryDay,
  regenerateItineraryItem,
  restoreItineraryVersion,
  updateTripItinerary
} from "@/lib/api/trips";
import type { Itinerary, Trip } from "@/entities/trip/model";

const itinerary: Itinerary = {
  days: [
    {
      day: 1,
      title: "Arrival",
      items: [{ time: "09:00", type: "activity", name: "Walk" }]
    }
  ]
};

const trip: Trip = {
  id: "trip-1",
  destination: "Rome",
  days: 1,
  budgetCurrency: "EUR",
  travelers: 1,
  interests: [],
  pace: "balanced",
  status: "COMPLETED",
  itinerary,
  itineraryRevision: 8,
  createdAt: "2026-06-25T00:00:00Z",
  updatedAt: "2026-06-25T00:00:00Z"
};

function jsonResponse(body: unknown, init: { ok: boolean; status: number }): Response {
  return {
    ok: init.ok,
    status: init.status,
    text: async () => JSON.stringify(body),
    json: async () => body
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("trip itinerary mutation API", () => {
  it("sends expectedItineraryRevision for all itinerary-changing requests", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(trip, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await updateTripItinerary("trip-1", itinerary, 7);
    await generateItinerary("trip-1", 0);
    await regenerateItineraryDay("trip-1", 2, "less walking", 8);
    await regenerateItineraryItem("trip-1", 2, 1, undefined, 9);
    await restoreItineraryVersion("trip-1", "version-1", 10);

    const bodies = fetchMock.mock.calls.map(([, init]) => JSON.parse(init?.body as string));
    expect(bodies[0]).toEqual({ itinerary, expectedItineraryRevision: 7 });
    expect(bodies[1]).toEqual({ expectedItineraryRevision: 0 });
    expect(bodies[2]).toEqual({
      instruction: "less walking",
      expectedItineraryRevision: 8
    });
    expect(bodies[3]).toEqual({ expectedItineraryRevision: 9 });
    expect(bodies[4]).toEqual({ expectedItineraryRevision: 10 });
  });

  it("parses 409 itinerary_conflict responses as typed conflicts", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse(
        {
          error: "itinerary_conflict",
          message: "This itinerary was changed by someone else.",
          currentItineraryRevision: 8
        },
        { ok: false, status: 409 }
      )
    );
    vi.stubGlobal("fetch", fetchMock);

    try {
      await updateTripItinerary("trip-1", itinerary, 7);
      throw new Error("expected updateTripItinerary to reject");
    } catch (error) {
      expect(isItineraryConflictError(error)).toBe(true);
      if (!isItineraryConflictError(error)) {
        throw error;
      }
      expect(error.message).toBe("This itinerary was changed by someone else.");
      expect(error.currentItineraryRevision).toBe(8);
    }
  });
});
