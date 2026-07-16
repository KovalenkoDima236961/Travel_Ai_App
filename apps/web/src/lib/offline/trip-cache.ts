import { getOfflineDb } from "@/lib/offline/db";
import type {
  CachedChecklistRecord,
  CachedExpenseSummaryRecord,
  CachedExpensesRecord,
  CachedRemindersRecord,
  CachedSettlementsRecord,
  CachedTripDetailsRecord,
  CachedTripRecord,
  OfflineReceiptDraftRecord,
  OfflineSettingsRecord,
  SyncLogRecord
} from "@/lib/offline/types";
import type { TripAccommodation } from "@/entities/accommodation/model";
import type { BudgetSummary } from "@/entities/budget/model";
import type { ChecklistViewResponse } from "@/entities/checklist/model";
import type {
  ExpenseSummary,
  SettlementsResponse,
  TripExpense,
  TripExpensesResponse
} from "@/entities/expense/model";
import type {
  ReminderSummary,
  ReminderViewResponse,
  TripReminder
} from "@/entities/trip-reminder/model";
import type { Itinerary, Trip } from "@/entities/trip/model";

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
    cacheKey: tripCacheKey(normalizedUserId, sanitizedTrip.id),
    tripId: sanitizedTrip.id,
    userId: normalizedUserId,
    trip: sanitizedTrip,
    tripSummary: cloneOfflineValue(sanitizedTrip),
    budgetSummary: cloneOfflineValue(budgetSummary ?? null),
    accommodation: sanitizedAccommodation,
    itineraryRevision: sanitizedTrip.itineraryRevision,
    routeRevision: routeRevisionFromTrip(sanitizedTrip),
    cachedAt: new Date().toISOString(),
    lastOpenedAt: new Date().toISOString(),
    offlineEnabled: true
  };

  const db = await getOfflineDb();
  await Promise.all([
    db.put("cachedTrips", record),
    db.put("cachedTripDetails", {
      cacheKey: tripCacheKey(normalizedUserId, sanitizedTrip.id),
      tripId: sanitizedTrip.id,
      userId: normalizedUserId,
      trip: sanitizedTrip,
      itinerary: sanitizedTrip.itinerary ?? null,
      route: sanitizedTrip.route ?? null,
      accommodation: sanitizedAccommodation,
      itineraryRevision: sanitizedTrip.itineraryRevision,
      cachedAt: record.cachedAt,
      source: "trip_detail"
    } satisfies CachedTripDetailsRecord)
  ]);
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
  const record = await db.get("cachedTrips", tripCacheKey(normalizedUserId, tripId));
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
  const record = await db.get("cachedTrips", tripCacheKey(normalizedUserId, tripId));
  if (!record || record.userId !== normalizedUserId) {
    return;
  }

  await Promise.all([
    db.delete("cachedTrips", tripCacheKey(normalizedUserId, tripId)),
    db.delete("cachedTripDetails", tripCacheKey(normalizedUserId, tripId)),
    db.delete("cachedChecklists", tripCacheKey(normalizedUserId, tripId)),
    db.delete("cachedReminders", tripCacheKey(normalizedUserId, tripId)),
    db.delete("cachedExpenses", tripCacheKey(normalizedUserId, tripId)),
    db.delete("cachedExpenseSummaries", tripCacheKey(normalizedUserId, tripId)),
    db.delete("cachedSettlements", tripCacheKey(normalizedUserId, tripId))
  ]);
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

export async function cacheChecklistSnapshot(input: {
  tripId: string;
  userId: string;
  checklist: ChecklistViewResponse;
}): Promise<void> {
  const normalizedUserId = input.userId.trim();
  if (!input.tripId || !normalizedUserId) {
    return;
  }
  const db = await getOfflineDb();
  const existing = await getCachedChecklist(input.tripId, normalizedUserId);
  await db.put("cachedChecklists", {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    checklist: cloneOfflineValue(input.checklist),
    cachedAt: new Date().toISOString(),
    localVersion: (existing?.localVersion ?? 0) + 1
  });
}

export async function getCachedChecklist(
  tripId: string,
  userId: string
): Promise<CachedChecklistRecord | null> {
  const normalizedUserId = userId.trim();
  if (!tripId || !normalizedUserId) {
    return null;
  }
  const db = await getOfflineDb();
  const record = await db.get("cachedChecklists", tripCacheKey(normalizedUserId, tripId));
  return record?.userId === normalizedUserId ? cloneOfflineValue(record) : null;
}

export async function cacheRemindersSnapshot(input: {
  tripId: string;
  userId: string;
  response: ReminderViewResponse;
}): Promise<void> {
  const normalizedUserId = input.userId.trim();
  if (!input.tripId || !normalizedUserId) {
    return;
  }
  const db = await getOfflineDb();
  const existing = await getCachedReminders(input.tripId, normalizedUserId);
  await db.put("cachedReminders", {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    reminders: cloneOfflineValue(input.response.reminders),
    summary: cloneOfflineValue(input.response.summary),
    cachedAt: new Date().toISOString(),
    localVersion: (existing?.localVersion ?? 0) + 1
  });
}

export async function getCachedReminders(
  tripId: string,
  userId: string
): Promise<CachedRemindersRecord | null> {
  const normalizedUserId = userId.trim();
  if (!tripId || !normalizedUserId) {
    return null;
  }
  const db = await getOfflineDb();
  const record = await db.get("cachedReminders", tripCacheKey(normalizedUserId, tripId));
  return record?.userId === normalizedUserId ? cloneOfflineValue(record) : null;
}

export async function cacheExpensesSnapshot(input: {
  tripId: string;
  userId: string;
  response: TripExpensesResponse;
}): Promise<void> {
  const normalizedUserId = input.userId.trim();
  if (!input.tripId || !normalizedUserId) {
    return;
  }
  const db = await getOfflineDb();
  const existing = await getCachedExpenses(input.tripId, normalizedUserId);
  await db.put("cachedExpenses", {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    expenses: cloneOfflineValue(input.response.items),
    cachedAt: new Date().toISOString(),
    localVersion: (existing?.localVersion ?? 0) + 1
  });
}

export async function getCachedExpenses(
  tripId: string,
  userId: string
): Promise<CachedExpensesRecord | null> {
  const normalizedUserId = userId.trim();
  if (!tripId || !normalizedUserId) {
    return null;
  }
  const db = await getOfflineDb();
  const record = await db.get("cachedExpenses", tripCacheKey(normalizedUserId, tripId));
  return record?.userId === normalizedUserId ? cloneOfflineValue(record) : null;
}

export async function cacheExpenseSummarySnapshot(input: {
  tripId: string;
  userId: string;
  summary: ExpenseSummary;
}): Promise<void> {
  const normalizedUserId = input.userId.trim();
  if (!input.tripId || !normalizedUserId) {
    return;
  }
  const db = await getOfflineDb();
  await db.put("cachedExpenseSummaries", {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    summary: cloneOfflineValue(input.summary),
    cachedAt: new Date().toISOString()
  });
}

export async function getCachedExpenseSummary(
  tripId: string,
  userId: string
): Promise<CachedExpenseSummaryRecord | null> {
  const normalizedUserId = userId.trim();
  if (!tripId || !normalizedUserId) {
    return null;
  }
  const db = await getOfflineDb();
  const record = await db.get("cachedExpenseSummaries", tripCacheKey(normalizedUserId, tripId));
  return record?.userId === normalizedUserId ? cloneOfflineValue(record) : null;
}

export async function cacheSettlementsSnapshot(input: {
  tripId: string;
  userId: string;
  settlements: SettlementsResponse;
}): Promise<void> {
  const normalizedUserId = input.userId.trim();
  if (!input.tripId || !normalizedUserId) {
    return;
  }
  const db = await getOfflineDb();
  await db.put("cachedSettlements", {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    settlements: cloneOfflineValue(input.settlements),
    cachedAt: new Date().toISOString()
  });
}

export async function getCachedSettlements(
  tripId: string,
  userId: string
): Promise<CachedSettlementsRecord | null> {
  const normalizedUserId = userId.trim();
  if (!tripId || !normalizedUserId) {
    return null;
  }
  const db = await getOfflineDb();
  const record = await db.get("cachedSettlements", tripCacheKey(normalizedUserId, tripId));
  return record?.userId === normalizedUserId ? cloneOfflineValue(record) : null;
}

export async function putCachedExpenses(input: {
  tripId: string;
  userId: string;
  expenses: TripExpense[];
}): Promise<CachedExpensesRecord> {
  const normalizedUserId = input.userId.trim();
  const existing = await getCachedExpenses(input.tripId, normalizedUserId);
  const record: CachedExpensesRecord = {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    expenses: cloneOfflineValue(input.expenses),
    cachedAt: new Date().toISOString(),
    localVersion: (existing?.localVersion ?? 0) + 1
  };
  const db = await getOfflineDb();
  await db.put("cachedExpenses", record);
  return cloneOfflineValue(record);
}

export async function putCachedReminders(input: {
  tripId: string;
  userId: string;
  reminders: TripReminder[];
  summary: ReminderSummary;
}): Promise<CachedRemindersRecord> {
  const normalizedUserId = input.userId.trim();
  const existing = await getCachedReminders(input.tripId, normalizedUserId);
  const record: CachedRemindersRecord = {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    reminders: cloneOfflineValue(input.reminders),
    summary: cloneOfflineValue(input.summary),
    cachedAt: new Date().toISOString(),
    localVersion: (existing?.localVersion ?? 0) + 1
  };
  const db = await getOfflineDb();
  await db.put("cachedReminders", record);
  return cloneOfflineValue(record);
}

export async function putCachedChecklist(input: {
  tripId: string;
  userId: string;
  checklist: ChecklistViewResponse;
}): Promise<CachedChecklistRecord> {
  const normalizedUserId = input.userId.trim();
  const existing = await getCachedChecklist(input.tripId, normalizedUserId);
  const record: CachedChecklistRecord = {
    cacheKey: tripCacheKey(normalizedUserId, input.tripId),
    tripId: input.tripId,
    userId: normalizedUserId,
    checklist: cloneOfflineValue(input.checklist),
    cachedAt: new Date().toISOString(),
    localVersion: (existing?.localVersion ?? 0) + 1
  };
  const db = await getOfflineDb();
  await db.put("cachedChecklists", record);
  return cloneOfflineValue(record);
}

export async function saveOfflineReceiptDraft(input: {
  tripId: string;
  userId: string;
  file: File;
  consentGranted: boolean;
  linkedExpenseLocalId?: string | null;
  linkedExpenseId?: string | null;
}): Promise<OfflineReceiptDraftRecord> {
  if (!input.consentGranted) {
    throw new Error("Explicit consent is required before storing a receipt on this device.");
  }
  const now = new Date().toISOString();
  const record: OfflineReceiptDraftRecord = {
    id: createOfflineId("receipt"),
    tripId: input.tripId,
    userId: input.userId.trim(),
    fileBlob: input.file,
    filename: input.file.name,
    contentType: input.file.type || "application/octet-stream",
    sizeBytes: input.file.size,
    linkedExpenseLocalId: input.linkedExpenseLocalId ?? null,
    linkedExpenseId: input.linkedExpenseId ?? null,
    createdOfflineAt: now,
    status: "pending_upload",
    error: null
  };
  const db = await getOfflineDb();
  await db.put("offlineReceiptDrafts", record);
  return record;
}

export async function getOfflineReceiptDraft(
  draftId: string
): Promise<OfflineReceiptDraftRecord | null> {
  if (!draftId) {
    return null;
  }
  const db = await getOfflineDb();
  const record = await db.get("offlineReceiptDrafts", draftId);
  return record ? cloneOfflineValue(record) : null;
}

export async function listOfflineReceiptDrafts(
  userId: string,
  tripId?: string
): Promise<OfflineReceiptDraftRecord[]> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId) {
    return [];
  }
  const db = await getOfflineDb();
  const drafts = await db.getAll("offlineReceiptDrafts");
  return drafts
    .filter(
      (draft) =>
        draft.userId === normalizedUserId &&
        (!tripId || draft.tripId === tripId) &&
        draft.status !== "cancelled"
    )
    .sort((left, right) => left.createdOfflineAt.localeCompare(right.createdOfflineAt))
    .map(cloneOfflineValue);
}

export async function updateOfflineReceiptDraft(
  draftId: string,
  patch: Partial<OfflineReceiptDraftRecord>
): Promise<OfflineReceiptDraftRecord | null> {
  const db = await getOfflineDb();
  const current = await db.get("offlineReceiptDrafts", draftId);
  if (!current) {
    return null;
  }
  const updated = { ...current, ...patch };
  await db.put("offlineReceiptDrafts", updated);
  return cloneOfflineValue(updated);
}

export async function deleteOfflineReceiptDraft(draftId: string, userId: string): Promise<void> {
  const normalizedUserId = userId.trim();
  if (!draftId || !normalizedUserId) {
    return;
  }
  const db = await getOfflineDb();
  const draft = await db.get("offlineReceiptDrafts", draftId);
  if (draft?.userId === normalizedUserId) {
    await db.delete("offlineReceiptDrafts", draftId);
  }
}

export async function purgeStaleOfflineData(
  userId: string,
  maxAgeDays = Number(process.env.NEXT_PUBLIC_OFFLINE_CACHE_MAX_AGE_DAYS ?? 30)
): Promise<number> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId || !Number.isFinite(maxAgeDays) || maxAgeDays < 1) {
    return 0;
  }
  const db = await getOfflineDb();
  const cutoff = Date.now() - maxAgeDays * 24 * 60 * 60 * 1000;
  const records = await db.getAll("cachedTrips");
  const stale = records.filter(
    (record) => record.userId === normalizedUserId && Date.parse(record.cachedAt) < cutoff
  );
  await Promise.all(stale.map((record) => deleteCachedTrip(record.tripId, normalizedUserId)));
  return stale.length;
}

export async function getOfflineSettings(userId: string): Promise<OfflineSettingsRecord> {
  const normalizedUserId = userId.trim();
  const db = await getOfflineDb();
  const existing = normalizedUserId ? await db.get("offlineSettings", normalizedUserId) : null;
  return (
    existing ?? {
      userId: normalizedUserId,
      autoCacheOpenedTrips: true,
      cacheReceiptsOffline: false,
      maxCachedTrips: 10,
      lastCleanupAt: null
    }
  );
}

export async function updateOfflineSettings(
  userId: string,
  patch: Partial<Omit<OfflineSettingsRecord, "userId">>
): Promise<OfflineSettingsRecord> {
  const current = await getOfflineSettings(userId);
  const updated = { ...current, ...patch };
  const db = await getOfflineDb();
  await db.put("offlineSettings", updated);
  return cloneOfflineValue(updated);
}

export async function appendSyncLog(input: Omit<SyncLogRecord, "id" | "createdAt">) {
  const db = await getOfflineDb();
  await db.put("syncLogs", {
    ...input,
    id: createOfflineId("log"),
    createdAt: new Date().toISOString()
  });
}

export async function clearOfflineData(userId?: string | null): Promise<void> {
  const db = await getOfflineDb();
  const normalizedUserId = userId?.trim();

  if (!normalizedUserId) {
    await Promise.all([
      db.clear("cachedTrips"),
      db.clear("cachedTripDetails"),
      db.clear("cachedChecklists"),
      db.clear("cachedReminders"),
      db.clear("cachedExpenses"),
      db.clear("cachedExpenseSummaries"),
      db.clear("cachedSettlements"),
      db.clear("pendingMutations"),
      db.clear("offlineReceiptDrafts"),
      db.clear("syncLogs"),
      db.clear("offlineSettings"),
      db.clear("syncMetadata")
    ]);
    return;
  }

  const [
    cachedTrips,
    cachedDetails,
    cachedChecklists,
    cachedReminders,
    cachedExpenses,
    cachedExpenseSummaries,
    cachedSettlements,
    mutations,
    receiptDrafts,
    syncLogs,
    metadata
  ] = await Promise.all([
    db.getAll("cachedTrips"),
    db.getAll("cachedTripDetails"),
    db.getAll("cachedChecklists"),
    db.getAll("cachedReminders"),
    db.getAll("cachedExpenses"),
    db.getAll("cachedExpenseSummaries"),
    db.getAll("cachedSettlements"),
    db.getAll("pendingMutations"),
    db.getAll("offlineReceiptDrafts"),
    db.getAll("syncLogs"),
    db.getAll("syncMetadata")
  ]);

  await Promise.all([
    ...cachedTrips
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedTrips", record.cacheKey)),
    ...cachedDetails
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedTripDetails", record.cacheKey)),
    ...cachedChecklists
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedChecklists", record.cacheKey)),
    ...cachedReminders
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedReminders", record.cacheKey)),
    ...cachedExpenses
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedExpenses", record.cacheKey)),
    ...cachedExpenseSummaries
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedExpenseSummaries", record.cacheKey)),
    ...cachedSettlements
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("cachedSettlements", record.cacheKey)),
    ...mutations
      .filter((mutation) => mutation.userId === normalizedUserId)
      .map((mutation) => db.delete("pendingMutations", mutation.mutationId)),
    ...receiptDrafts
      .filter((draft) => draft.userId === normalizedUserId)
      .map((draft) => db.delete("offlineReceiptDrafts", draft.id)),
    ...syncLogs
      .filter((record) => record.userId === normalizedUserId)
      .map((record) => db.delete("syncLogs", record.id)),
    db.delete("offlineSettings", normalizedUserId),
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

export function tripCacheKey(userId: string, tripId: string) {
  return `private:${userId.trim()}:${tripId}`;
}

export function createOfflineId(prefix: string) {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function routeRevisionFromTrip(trip: Trip) {
  const metadata = trip.route?.preferences as Record<string, unknown> | undefined;
  return typeof metadata?.revision === "number" ? metadata.revision : null;
}
