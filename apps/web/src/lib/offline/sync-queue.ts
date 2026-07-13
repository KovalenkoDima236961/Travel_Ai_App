import { isItineraryConflictError } from "@/shared/api/client";
import {
  checkChecklistItem,
  createChecklistItem,
  uncheckChecklistItem
} from "@/lib/api/checklists";
import { createTripExpense } from "@/lib/api/expenses";
import { uploadReceipt } from "@/lib/api/receipts";
import {
  completeTripReminder,
  createTripReminder,
  disableTripReminder,
  reopenTripReminder
} from "@/lib/api/trip-reminders";
import { getTrip, updateTripItinerary } from "@/lib/api/trips";
import { getOfflineDb } from "@/lib/offline/db";
import { isOfflineLikeError } from "@/lib/offline/network";
import {
  appendSyncLog,
  cacheTripSnapshot,
  cloneOfflineValue,
  createOfflineId,
  getOfflineReceiptDraft,
  updateOfflineReceiptDraft
} from "@/lib/offline/trip-cache";
import {
  clearOfflineMetadata,
  replaceOfflineChecklistItem,
  replaceOfflineExpense,
  replaceOfflineReminder
} from "@/lib/offline/cache-writer";
import type {
  EnqueueItineraryUpdateInput,
  OfflineCompanionMutationType,
  OfflineMutationEntity,
  OfflineMutationStatus,
  PendingCompanionMutation,
  PendingItineraryMutation,
  PendingOfflineMutation,
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
const MAX_AUTOMATIC_ATTEMPTS = 3;

type EnqueueCompanionMutationInput<TType extends OfflineCompanionMutationType> = {
  tripId: string;
  userId: string;
  type: TType;
  entity: OfflineMutationEntity;
  payload: PendingCompanionMutation<TType>["payload"];
  localEntityId?: string | null;
  dependsOn?: string[];
  clientMutationId?: string;
  requestHash?: string | null;
};

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
    id: createMutationId(),
    type: "update_itinerary",
    entity: "itinerary",
    tripId: input.tripId,
    userId: input.userId,
    baseRevision: input.baseRevision,
    baseItinerary: cloneOfflineValue(input.baseItinerary),
    draftItinerary: cloneOfflineValue(input.draftItinerary),
    status: "pending",
    createdAt: now,
    updatedAt: now,
    lastAttemptAt: null,
    attemptCount: 0,
    errorCode: null,
    errorMessage: null,
    error: null,
    clientMutationId: createMutationId(),
    createdOfflineAt: now,
    dependsOn: []
  };

  const db = await getOfflineDb();
  await db.put("pendingMutations", mutation);
  notifyOfflineQueueChanged();
  return cloneOfflineValue(mutation);
}

export async function enqueueCompanionMutation<TType extends OfflineCompanionMutationType>(
  input: EnqueueCompanionMutationInput<TType>
): Promise<PendingCompanionMutation<TType>> {
  const now = new Date().toISOString();
  const clientMutationId = input.clientMutationId ?? createMutationId();
  const mutationId = createMutationId();
  const mutation = {
    mutationId,
    id: mutationId,
    type: input.type,
    entity: input.entity,
    tripId: input.tripId,
    userId: input.userId.trim(),
    status: input.type.endsWith("_delete_local") ? "cancelled" : "pending",
    payload: cloneOfflineValue(input.payload),
    clientMutationId,
    createdOfflineAt: now,
    createdAt: now,
    updatedAt: now,
    lastAttemptAt: null,
    attemptCount: 0,
    error: null,
    errorCode: null,
    errorMessage: null,
    dependsOn: input.dependsOn ?? [],
    localEntityId: input.localEntityId ?? null,
    entityId: null,
    requestHash: input.requestHash ?? stableHash(input.payload)
  } as PendingCompanionMutation<TType>;

  const db = await getOfflineDb();
  await db.put("pendingMutations", mutation);
  notifyOfflineQueueChanged();
  return cloneOfflineValue(mutation);
}

export async function getPendingMutations(userId: string): Promise<PendingOfflineMutation[]> {
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
        ACTIVE_STATUSES.has(mutation.status)
    )
    .sort((left, right) => left.createdAt.localeCompare(right.createdAt))
    .map(cloneOfflineValue);
}

export async function getPendingItineraryMutations(
  userId: string
): Promise<PendingItineraryMutation[]> {
  return (await getPendingMutations(userId)).filter(
    (mutation): mutation is PendingItineraryMutation => mutation.type === "update_itinerary"
  );
}

export async function getPendingMutationForTrip(
  tripId: string,
  userId: string
): Promise<PendingItineraryMutation | null> {
  const mutations = await getPendingItineraryMutations(userId);
  return (
    mutations
      .filter((mutation) => mutation.tripId === tripId)
      .sort((left, right) => right.updatedAt.localeCompare(left.updatedAt))[0] ?? null
  );
}

export async function markMutationSyncing(
  mutationId: string
): Promise<PendingOfflineMutation | null> {
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
): Promise<PendingOfflineMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "conflict",
    errorCode: error.code ?? "itinerary_conflict",
    errorMessage: error.message ?? "This trip changed while you were offline."
  });
}

export async function markMutationFailed(
  mutationId: string,
  error: { code?: string | null; message?: string | null }
): Promise<PendingOfflineMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "failed",
    errorCode: error.code ?? "sync_failed",
    errorMessage: error.message ?? "Could not sync offline changes."
  });
}

export async function markMutationSynced(
  mutationId: string
): Promise<PendingOfflineMutation | null> {
  return updateMutationRecord(mutationId, {
    status: "synced",
    errorCode: null,
    errorMessage: null
  });
}

export async function discardMutation(
  mutationId: string
): Promise<PendingOfflineMutation | null> {
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
    const mutations = sortMutationsForSync(
      (await getPendingMutations(normalizedUserId)).filter(
        (mutation) =>
          mutation.status === "pending" &&
          (mutation.attemptCount ?? 0) < MAX_AUTOMATIC_ATTEMPTS
      )
    );

    for (const mutation of mutations) {
      if (!(await dependenciesSatisfied(mutation))) {
        continue;
      }

      const syncingMutation =
        ((await updateMutationRecord(mutation.mutationId, {
          status: "syncing",
          lastAttemptAt: new Date().toISOString(),
          attemptCount: (mutation.attemptCount ?? 0) + 1,
          errorCode: null,
          errorMessage: null,
          error: null
        })) as PendingOfflineMutation | null) ?? mutation;

      try {
        const result = await syncMutation(normalizedUserId, syncingMutation);
        results.push(result);
      } catch (error) {
        if (syncingMutation.type === "update_itinerary" && isItineraryConflictError(error)) {
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
              errorMessage: "Network connection was lost. Sync will retry when online.",
              error: "Network connection was lost. Sync will retry when online."
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

async function syncMutation(
  userId: string,
  mutation: PendingOfflineMutation
): Promise<SyncResult> {
  switch (mutation.type) {
    case "update_itinerary": {
      const updatedTrip = await updateTripItinerary(
        mutation.tripId,
        mutation.draftItinerary,
        mutation.baseRevision
      );
      const syncedMutation = (await markMutationSynced(mutation.mutationId)) ?? mutation;
      await cacheTripSnapshot({
        userId,
        trip: updatedTrip,
        budgetSummary: null,
        accommodation: updatedTrip.accommodation ?? null
      });
      await appendSyncLog({
        userId,
        tripId: mutation.tripId,
        eventType: "itinerary_synced",
        message: "Offline itinerary changes synced.",
        metadata: { mutationId: mutation.mutationId }
      });
      return { status: "synced", mutation: syncedMutation, trip: updatedTrip };
    }
    case "checklist_item_check":
    case "checklist_item_uncheck": {
      const itemId =
        (await findSyncedEntityId(userId, mutation.payload.itemId)) ?? mutation.payload.itemId;
      const item =
        mutation.type === "checklist_item_check"
          ? await checkChecklistItem(mutation.tripId, itemId)
          : await uncheckChecklistItem(mutation.tripId, itemId);
      const cleanItem = clearOfflineMetadata(item);
      await replaceOfflineChecklistItem({
        tripId: mutation.tripId,
        userId,
        item: cleanItem
      });
      const syncedMutation = await markMutationSynced(mutation.mutationId);
      return { status: "synced", mutation: syncedMutation ?? mutation, entity: cleanItem };
    }
    case "checklist_item_create": {
      const item = await createChecklistItem(mutation.tripId, {
        ...mutation.payload.input,
        metadata: withOfflineIdempotencyMetadata(
          mutation.payload.input.metadata,
          mutation.clientMutationId,
          mutation.createdOfflineAt,
          mutation.requestHash
        )
      });
      const cleanItem = clearOfflineMetadata(item);
      await replaceOfflineChecklistItem({
        tripId: mutation.tripId,
        userId,
        localEntityId: mutation.payload.localEntityId,
        item: cleanItem
      });
      const syncedMutation = await updateMutationRecord(mutation.mutationId, {
        status: "synced",
        entityId: item.id,
        errorCode: null,
        errorMessage: null,
        error: null
      });
      return { status: "synced", mutation: syncedMutation ?? mutation, entity: cleanItem };
    }
    case "reminder_complete":
    case "reminder_reopen": {
      const reminderId =
        (await findSyncedEntityId(userId, mutation.payload.reminderId)) ??
        mutation.payload.reminderId;
      const reminder =
        mutation.type === "reminder_complete"
          ? await completeTripReminder(mutation.tripId, reminderId)
          : await reopenTripReminder(mutation.tripId, reminderId);
      const cleanReminder = clearOfflineMetadata(reminder);
      await replaceOfflineReminder({ tripId: mutation.tripId, userId, reminder: cleanReminder });
      const syncedMutation = await markMutationSynced(mutation.mutationId);
      return { status: "synced", mutation: syncedMutation ?? mutation, entity: cleanReminder };
    }
    case "reminder_disable": {
      const reminderId =
        (await findSyncedEntityId(userId, mutation.payload.reminderId)) ??
        mutation.payload.reminderId;
      const reminder = await disableTripReminder(mutation.tripId, reminderId);
      const cleanReminder = clearOfflineMetadata(reminder);
      await replaceOfflineReminder({ tripId: mutation.tripId, userId, reminder: cleanReminder });
      const syncedMutation = await markMutationSynced(mutation.mutationId);
      return { status: "synced", mutation: syncedMutation ?? mutation, entity: cleanReminder };
    }
    case "reminder_create": {
      const reminder = await createTripReminder(mutation.tripId, {
        ...mutation.payload.input,
        metadata: withOfflineIdempotencyMetadata(
          mutation.payload.input.metadata,
          mutation.clientMutationId,
          mutation.createdOfflineAt,
          mutation.requestHash
        )
      });
      const cleanReminder = clearOfflineMetadata(reminder);
      await replaceOfflineReminder({
        tripId: mutation.tripId,
        userId,
        localEntityId: mutation.payload.localEntityId,
        reminder: cleanReminder
      });
      const syncedMutation = await updateMutationRecord(mutation.mutationId, {
        status: "synced",
        entityId: reminder.id,
        errorCode: null,
        errorMessage: null,
        error: null
      });
      return { status: "synced", mutation: syncedMutation ?? mutation, entity: cleanReminder };
    }
    case "expense_create": {
      const expense = await createTripExpense(mutation.tripId, {
        ...mutation.payload.input,
        metadata: withOfflineIdempotencyMetadata(
          mutation.payload.input.metadata,
          mutation.clientMutationId,
          mutation.createdOfflineAt,
          mutation.requestHash
        )
      });
      const cleanExpense = clearOfflineMetadata(expense);
      await replaceOfflineExpense({
        tripId: mutation.tripId,
        userId,
        localEntityId: mutation.payload.localEntityId,
        expense: cleanExpense
      });
      const syncedMutation = await updateMutationRecord(mutation.mutationId, {
        status: "synced",
        entityId: expense.id,
        errorCode: null,
        errorMessage: null,
        error: null
      });
      return { status: "synced", mutation: syncedMutation ?? mutation, entity: cleanExpense };
    }
    case "receipt_upload": {
      const draft = await getOfflineReceiptDraft(mutation.payload.receiptDraftId);
      if (!draft) {
        throw new Error("Receipt draft no longer exists.");
      }
      const linkedExpenseId =
        mutation.payload.linkedExpenseId ??
        draft.linkedExpenseId ??
        (draft.linkedExpenseLocalId
          ? await findSyncedEntityId(userId, draft.linkedExpenseLocalId)
          : null);
      if (draft.linkedExpenseLocalId && !linkedExpenseId) {
        throw new Error("Receipt is waiting for its expense draft to sync first.");
      }
      await updateOfflineReceiptDraft(draft.id, { status: "uploading", error: null });
      const file = new File([draft.fileBlob], draft.filename, { type: draft.contentType });
      const receipt = await uploadReceipt(mutation.tripId, {
        file,
        expenseId: linkedExpenseId,
        runOcr: true
      });
      await updateOfflineReceiptDraft(draft.id, {
        status: "uploaded",
        linkedExpenseId,
        error: null
      });
      const syncedMutation = await updateMutationRecord(mutation.mutationId, {
        status: "synced",
        entityId: receipt.id,
        errorCode: null,
        errorMessage: null,
        error: null
      });
      return { status: "synced", mutation: syncedMutation ?? mutation, entity: receipt };
    }
    case "checklist_item_update":
    case "checklist_item_delete_local":
    case "expense_update_local":
    case "expense_delete_local": {
      const syncedMutation = await markMutationSynced(mutation.mutationId);
      return { status: "synced", mutation: syncedMutation ?? mutation };
    }
    default: {
      const exhaustive: never = mutation;
      throw new Error(`Unsupported offline mutation: ${String(exhaustive)}`);
    }
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
  patch: Partial<PendingOfflineMutation>
): Promise<PendingOfflineMutation | null> {
  const db = await getOfflineDb();
  const current = await db.get("pendingMutations", mutationId);
  if (!current) {
    return null;
  }

  const updated: PendingOfflineMutation = {
    ...current,
    ...patch,
    updatedAt: new Date().toISOString()
  } as PendingOfflineMutation;
  await db.put("pendingMutations", updated);
  notifyOfflineQueueChanged();
  return cloneOfflineValue(updated);
}

function notifyOfflineQueueChanged() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(OFFLINE_QUEUE_CHANGED_EVENT));
  }
}

function sortMutationsForSync(mutations: PendingOfflineMutation[]) {
  const order: Record<string, number> = {
    checklist: 1,
    reminder: 2,
    expense: 3,
    receipt: 4,
    itinerary: 5
  };
  return [...mutations].sort((left, right) => {
    const leftOrder = order[left.entity ?? "itinerary"] ?? 99;
    const rightOrder = order[right.entity ?? "itinerary"] ?? 99;
    if (leftOrder !== rightOrder) {
      return leftOrder - rightOrder;
    }
    return left.createdAt.localeCompare(right.createdAt);
  });
}

async function dependenciesSatisfied(mutation: PendingOfflineMutation) {
  const dependsOn = mutation.dependsOn ?? [];
  if (dependsOn.length === 0) {
    return true;
  }
  const db = await getOfflineDb();
  for (const mutationId of dependsOn) {
    const dependency = await db.get("pendingMutations", mutationId);
    if (!dependency || dependency.status !== "synced") {
      return false;
    }
  }
  return true;
}

async function findSyncedEntityId(userId: string, localEntityId: string) {
  const db = await getOfflineDb();
  const mutations = await db.getAll("pendingMutations");
  const syncedMutation = mutations.find(
    (mutation): mutation is PendingCompanionMutation =>
        mutation.userId === userId &&
        mutation.type !== "update_itinerary" &&
        mutation.status === "synced" &&
        mutation.localEntityId === localEntityId &&
        typeof mutation.entityId === "string"
  );
  return syncedMutation?.entityId ?? null;
}

function withOfflineIdempotencyMetadata(
  metadata: Record<string, unknown> | null | undefined,
  clientMutationId: string,
  createdOfflineAt: string,
  requestHash?: string | null
) {
  return {
    ...(metadata ?? {}),
    offlineClientMutationId: clientMutationId,
    offlineCreatedAt: createdOfflineAt,
    offlineRequestHash: requestHash ?? stableHash(metadata ?? {})
  };
}

function stableHash(value: unknown) {
  return JSON.stringify(sortForHash(value));
}

function sortForHash(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sortForHash);
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, entry]) => [key, sortForHash(entry)])
    );
  }
  return value;
}
