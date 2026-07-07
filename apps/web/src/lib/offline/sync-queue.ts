import { isItineraryConflictError } from "@/shared/api/client";
import { getTrip, updateTripItinerary } from "@/lib/api/trips";
import { getOfflineDb } from "@/lib/offline/db";
import { isOfflineLikeError } from "@/lib/offline/network";
import { cacheTripSnapshot, cloneOfflineValue } from "@/lib/offline/trip-cache";
import type {
  EnqueueItineraryUpdateInput,
  OfflineMutationStatus,
  PendingItineraryMutation,
  SyncResult
} from "@/lib/offline/types";

export const OFFLINE_QUEUE_CHANGED_EVENT = "travel-ai:offline-queue-changed";

const ACTIVE_STATUSES = new Set<OfflineMutationStatus>([
  "pending",
  "syncing",
  "conflict",
  "failed"
]);

const FINAL_STATUSES = new Set<OfflineMutationStatus>(["synced", "discarded"]);
const syncLocks = new Set<string>();

export async function enqueueItineraryUpdate(
  input: EnqueueItineraryUpdateInput
): Promise<PendingItineraryMutation> {
  const now = new Date().toISOString();
  const existing = await getPendingMutationForTrip(input.tripId, input.userId);

  if (existing && !FINAL_STATUSES.has(existing.status)) {
    const updated: PendingItineraryMutation = {
      ...existing,
      draftItinerary: cloneOfflineValue(input.draftItinerary),
      status: "pending",
      updatedAt: now,
      lastAttemptAt: null,
      errorCode: null,
      errorMessage: null
    };
    const db = await getOfflineDb();
    await db.put("pendingMutations", updated);
    notifyOfflineQueueChanged();
    return cloneOfflineValue(updated);
  }

  const mutation: PendingItineraryMutation = {
    mutationId: createMutationId(),
    type: "update_itinerary",
    tripId: input.tripId,
    userId: input.userId,
    baseRevision: input.baseRevision,
    baseItinerary: cloneOfflineValue(input.baseItinerary),
    draftItinerary: cloneOfflineValue(input.draftItinerary),
    status: "pending",
    createdAt: now,
    updatedAt: now,
    lastAttemptAt: null,
    errorCode: null,
    errorMessage: null
  };

  const db = await getOfflineDb();
  await db.put("pendingMutations", mutation);
  notifyOfflineQueueChanged();
  return cloneOfflineValue(mutation);
}

export async function getPendingMutations(userId: string): Promise<PendingItineraryMutation[]> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId) {
    return [];
  }

  const db = await getOfflineDb();
  const mutations = await db.getAll("pendingMutations");
  return mutations
    .filter(
      (mutation) =>
        mutation.userId === normalizedUserId &&
        mutation.type === "update_itinerary" &&
        ACTIVE_STATUSES.has(mutation.status)
    )
    .sort((left, right) => left.createdAt.localeCompare(right.createdAt))
    .map(cloneOfflineValue);
}

export async function getPendingMutationForTrip(
  tripId: string,
  userId: string
): Promise<PendingItineraryMutation | null> {
  const mutations = await getPendingMutations(userId);
  return (
    mutations
      .filter((mutation) => mutation.tripId === tripId)
      .sort((left, right) => right.updatedAt.localeCompare(left.updatedAt))[0] ?? null
  );
}

export async function markMutationSyncing(
  mutationId: string
): Promise<PendingItineraryMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "syncing",
    lastAttemptAt: new Date().toISOString(),
    errorCode: null,
    errorMessage: null
  });
}

export async function markMutationConflict(
  mutationId: string,
  error: { code?: string | null; message?: string | null }
): Promise<PendingItineraryMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "conflict",
    errorCode: error.code ?? "itinerary_conflict",
    errorMessage: error.message ?? "This trip changed while you were offline."
  });
}

export async function markMutationFailed(
  mutationId: string,
  error: { code?: string | null; message?: string | null }
): Promise<PendingItineraryMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "failed",
    errorCode: error.code ?? "sync_failed",
    errorMessage: error.message ?? "Could not sync offline changes."
  });
}

export async function markMutationSynced(
  mutationId: string
): Promise<PendingItineraryMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "synced",
    errorCode: null,
    errorMessage: null
  });
}

export async function discardMutation(
  mutationId: string
): Promise<PendingItineraryMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "discarded",
    errorCode: null,
    errorMessage: null
  });
}

export async function syncPendingMutations(userId: string): Promise<SyncResult[]> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId || syncLocks.has(normalizedUserId)) {
    return [];
  }

  syncLocks.add(normalizedUserId);
  const results: SyncResult[] = [];

  try {
    const mutations = (await getPendingMutations(normalizedUserId)).filter(
      (mutation) => mutation.status === "pending"
    );

    for (const mutation of mutations) {
      const syncingMutation = (await markMutationSyncing(mutation.mutationId)) ?? mutation;

      try {
        const updatedTrip = await updateTripItinerary(
          syncingMutation.tripId,
          syncingMutation.draftItinerary,
          syncingMutation.baseRevision
        );
        const syncedMutation =
          (await markMutationSynced(syncingMutation.mutationId)) ?? syncingMutation;
        await cacheTripSnapshot({
          userId: normalizedUserId,
          trip: updatedTrip,
          budgetSummary: null,
          accommodation: updatedTrip.accommodation ?? null
        });
        results.push({
          status: "synced",
          mutation: syncedMutation,
          trip: updatedTrip
        });
      } catch (error) {
        if (isItineraryConflictError(error)) {
          const conflictMutation =
            (await markMutationConflict(syncingMutation.mutationId, {
              code: error.code,
              message: error.message
            })) ?? syncingMutation;
          let latestTrip = null;
          try {
            latestTrip = await getTrip(syncingMutation.tripId);
          } catch {
            latestTrip = null;
          }
          results.push({
            status: "conflict",
            mutation: conflictMutation,
            currentItineraryRevision: error.currentItineraryRevision,
            latestTrip,
            errorMessage: error.message
          });
          continue;
        }

        if (isOfflineLikeError(error)) {
          const retryableMutation =
            (await updateMutationRecord(syncingMutation.mutationId, {
              status: "pending",
              errorCode: "network_error",
              errorMessage: "Network connection was lost. Sync will retry when online."
            })) ?? syncingMutation;
          results.push({
            status: "failed",
            mutation: retryableMutation,
            retryable: true,
            errorCode: retryableMutation.errorCode,
            errorMessage: retryableMutation.errorMessage
          });
          break;
        }

        const failedMutation =
          (await markMutationFailed(syncingMutation.mutationId, {
            message: error instanceof Error ? error.message : "Could not sync offline changes."
          })) ?? syncingMutation;
        results.push({
          status: "failed",
          mutation: failedMutation,
          retryable: false,
          errorCode: failedMutation.errorCode,
          errorMessage: failedMutation.errorMessage
        });
      }
    }

    return results;
  } finally {
    syncLocks.delete(normalizedUserId);
    notifyOfflineQueueChanged();
  }
}

function createMutationId() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }

  return `offline-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

async function updateMutationRecord(
  mutationId: string,
  patch: Partial<PendingItineraryMutation>
): Promise<PendingItineraryMutation | null> {
  const db = await getOfflineDb();
  const current = await db.get("pendingMutations", mutationId);
  if (!current) {
    return null;
  }

  const updated: PendingItineraryMutation = {
    ...current,
    ...patch,
    updatedAt: new Date().toISOString()
  };
  await db.put("pendingMutations", updated);
  notifyOfflineQueueChanged();
  return cloneOfflineValue(updated);
}

function notifyOfflineQueueChanged() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(OFFLINE_QUEUE_CHANGED_EVENT));
  }
}
