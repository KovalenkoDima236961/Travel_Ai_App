import { getOfflineDb } from "@/lib/offline/db";
import type { CachedTripRecord } from "@/lib/offline/types";
import type { TripAccommodation } from "@/types/accommodation";
import type { BudgetSummary } from "@/types/budget";
import type { Itinerary, Trip } from "@/types/trip";

type CacheTripSnapshotInput = {
  userId: string;
  trip: Trip;
  budgetSummary?: BudgetSummary | null;
  accommodation?: TripAccommodation | null;
};

export async function cacheTripSnapshot({
  userId,
  trip,
  budgetSummary,
  accommodation
}: CacheTripSnapshotInput): Promise<void> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId || !trip.id) {
    return;
  }

  const sanitizedTrip = cloneOfflineValue(trip);
  const sanitizedAccommodation = cloneOfflineValue(
    accommodation ?? sanitizedTrip.accommodation ?? null
  );

  sanitizedTrip.accommodation = sanitizedAccommodation;

  const record: CachedTripRecord = {
    tripId: sanitizedTrip.id,
    userId: normalizedUserId,
    trip: sanitizedTrip,
    budgetSummary: cloneOfflineValue(budgetSummary ?? null),
    accommodation: sanitizedAccommodation,
    itineraryRevision: sanitizedTrip.itineraryRevision,
    cachedAt: new Date().toISOString()
  };

  const db = await getOfflineDb();
  await db.put("cachedTrips", record);
}

export async function getCachedTrip(
  tripId: string,
  userId: string
): Promise<CachedTripRecord | null> {
  const normalizedUserId = userId.trim();
  if (!tripId || !normalizedUserId) {
    return null;
  }

  const db = await getOfflineDb();
  const record = await db.get("cachedTrips", tripId);
  if (!record || record.userId !== normalizedUserId) {
    return null;
  }

  return cloneOfflineValue(record);
}

export async function listCachedTrips(userId: string): Promise<CachedTripRecord[]> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId) {
    return [];
  }

  const db = await getOfflineDb();
  const records = await db.getAll("cachedTrips");
  return records
    .filter((record) => record.userId === normalizedUserId)
    .sort((left, right) => right.cachedAt.localeCompare(left.cachedAt))
    .map(cloneOfflineValue);
}

export async function deleteCachedTrip(tripId: string, userId: string): Promise<void> {
  const normalizedUserId = userId.trim();
  if (!tripId || !normalizedUserId) {
    return;
  }

  const db = await getOfflineDb();
  const record = await db.get("cachedTrips", tripId);
  if (!record || record.userId !== normalizedUserId) {
    return;
  }

  await db.delete("cachedTrips", tripId);
}

export async function getOfflineStorageEstimate(): Promise<{
  usage?: number;
  quota?: number;
}> {
  if (typeof navigator === "undefined" || !navigator.storage?.estimate) {
    return {};
  }

  try {
    const estimate = await navigator.storage.estimate();
    return {
      usage: estimate.usage,
      quota: estimate.quota
    };
  } catch {
    return {};
  }
}

export async function clearOfflineDataForUser(userId: string): Promise<void> {
  await clearOfflineData(userId);
}

export async function updateCachedTripItinerary(input: {
  tripId: string;
  userId: string;
  itinerary: Itinerary;
}): Promise<CachedTripRecord | null> {
  const record = await getCachedTrip(input.tripId, input.userId);
  if (!record) {
    return null;
  }

  const nextRecord: CachedTripRecord = {
    ...record,
    trip: {
      ...record.trip,
      itinerary: cloneOfflineValue(input.itinerary),
      updatedAt: new Date().toISOString()
    },
    cachedAt: new Date().toISOString()
  };

  const db = await getOfflineDb();
  await db.put("cachedTrips", nextRecord);
  return cloneOfflineValue(nextRecord);
}

export async function clearOfflineData(userId?: string | null): Promise<void> {
  const db = await getOfflineDb();
  const normalizedUserId = userId?.trim();

  if (!normalizedUserId) {
    await Promise.all([
      db.clear("cachedTrips"),
      db.clear("pendingMutations"),
      db.clear("syncMetadata")
    ]);
    return;
  }

  const [cachedTrips, mutations, metadata] = await Promise.all([
    db.getAll("cachedTrips"),
    db.getAll("pendingMutations"),
    db.getAll("syncMetadata")
  ]);

  await Promise.all([
    ...cachedTrips
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedTrips", record.tripId)),
    ...mutations
      .filter((mutation) => mutation.userId === normalizedUserId)
      .map((mutation) => db.delete("pendingMutations", mutation.mutationId)),
    ...metadata
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("syncMetadata", record.key))
  ]);
}

export function cloneOfflineValue<T>(value: T): T {
  if (value == null) {
    return value;
  }

  if (typeof structuredClone === "function") {
    return structuredClone(value);
  }

  return JSON.parse(JSON.stringify(value)) as T;
}
