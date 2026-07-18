import { getOfflineDb } from "@/lib/offline/db";
import { getOfflineStorageEstimate, getOfflineSettings, updateOfflineSettings } from "@/lib/offline/trip-cache";

export type OfflineDataSummary = {
  cachedTrips: number;
  cachedDetails: number;
  cachedChecklists: number;
  cachedReminders: number;
  cachedExpenses: number;
  cachedTravelDays: number;
  pendingMutations: number;
  receiptDrafts: number;
  usage?: number;
  quota?: number;
  lastCleanupAt: string | null;
};

export type OfflineCleanupScope = "cachedTrips" | "pendingMutations" | "receiptDrafts" | "all";

export async function getOfflineDataSummary(userId: string): Promise<OfflineDataSummary> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId) {
    return emptySummary();
  }
  const db = await getOfflineDb();
  const [trips, details, checklists, reminders, expenses, travelDays, mutations, drafts, estimate, settings] = await Promise.all([
    db.getAll("cachedTrips"), db.getAll("cachedTripDetails"), db.getAll("cachedChecklists"),
    db.getAll("cachedReminders"), db.getAll("cachedExpenses"), db.getAll("cachedTravelDays"),
    db.getAll("pendingMutations"), db.getAll("offlineReceiptDrafts"), getOfflineStorageEstimate(),
    getOfflineSettings(normalizedUserId)
  ]);
  const forUser = <T extends { userId: string }>(items: T[]) => items.filter((item) => item.userId === normalizedUserId).length;
  return {
    cachedTrips: forUser(trips), cachedDetails: forUser(details), cachedChecklists: forUser(checklists),
    cachedReminders: forUser(reminders), cachedExpenses: forUser(expenses), cachedTravelDays: forUser(travelDays),
    pendingMutations: forUser(mutations), receiptDrafts: forUser(drafts), usage: estimate.usage,
    quota: estimate.quota, lastCleanupAt: settings.lastCleanupAt ?? null
  };
}

export async function clearOfflineDataScope(userId: string, scope: OfflineCleanupScope): Promise<void> {
  const normalizedUserId = userId.trim();
  if (!normalizedUserId) return;
  const db = await getOfflineDb();
  const shouldClearCache = scope === "cachedTrips" || scope === "all";
  const shouldClearMutations = scope === "pendingMutations" || scope === "all";
  const shouldClearDrafts = scope === "receiptDrafts" || scope === "all";
  if (shouldClearCache) {
    await Promise.all([
      deleteCacheStore("cachedTrips"), deleteCacheStore("cachedTripDetails"),
      deleteCacheStore("cachedChecklists"), deleteCacheStore("cachedReminders"),
      deleteCacheStore("cachedExpenses"), deleteCacheStore("cachedExpenseSummaries"),
      deleteCacheStore("cachedSettlements"), deleteCacheStore("cachedTravelDays")
    ]);
  }
  if (shouldClearMutations) {
    const records = await db.getAll("pendingMutations");
    await Promise.all(records.filter((item) => item.userId === normalizedUserId).map((item) => db.delete("pendingMutations", item.mutationId)));
  }
  if (shouldClearDrafts) {
    const records = await db.getAll("offlineReceiptDrafts");
    await Promise.all(records.filter((item) => item.userId === normalizedUserId).map((item) => db.delete("offlineReceiptDrafts", item.id)));
  }
  if (scope === "all" && typeof caches !== "undefined") {
    const names = await caches.keys();
    await Promise.all(names.filter((name) => name.startsWith("travel-ai-app-shell-")).map((name) => caches.delete(name)));
  }
  await updateOfflineSettings(normalizedUserId, { lastCleanupAt: new Date().toISOString() });

  async function deleteCacheStore(store: "cachedTrips" | "cachedTripDetails" | "cachedChecklists" | "cachedReminders" | "cachedExpenses" | "cachedExpenseSummaries" | "cachedSettlements" | "cachedTravelDays") {
    const records = await db.getAll(store);
    await Promise.all(records.filter((item) => item.userId === normalizedUserId).map((item) => db.delete(store, item.cacheKey)));
  }
}

function emptySummary(): OfflineDataSummary {
  return { cachedTrips: 0, cachedDetails: 0, cachedChecklists: 0, cachedReminders: 0, cachedExpenses: 0, cachedTravelDays: 0, pendingMutations: 0, receiptDrafts: 0, lastCleanupAt: null };
}
