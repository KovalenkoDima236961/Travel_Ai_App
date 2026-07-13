import { openDB, type DBSchema, type IDBPDatabase } from "idb";
import type {
  CachedChecklistRecord,
  CachedExpenseSummaryRecord,
  CachedExpensesRecord,
  CachedRemindersRecord,
  CachedSettlementsRecord,
  CachedTripDetailsRecord,
  CachedTripRecord,
  OfflineReceiptDraftRecord,
  OfflineMutationStatus,
  OfflineSettingsRecord,
  PendingOfflineMutation,
  SyncLogRecord,
  SyncMetadataRecord
} from "@/lib/offline/types";

const DB_NAME = "travel-ai-offline-v1";
const DB_VERSION = 2;

interface TravelAIOfflineDB extends DBSchema {
  cachedTrips: {
    key: string;
    value: CachedTripRecord;
  };
  cachedTripDetails: {
    key: string;
    value: CachedTripDetailsRecord;
    indexes: {
      by_userId: string;
      by_tripId: string;
    };
  };
  cachedChecklists: {
    key: string;
    value: CachedChecklistRecord;
    indexes: {
      by_userId: string;
      by_tripId: string;
    };
  };
  cachedReminders: {
    key: string;
    value: CachedRemindersRecord;
    indexes: {
      by_userId: string;
      by_tripId: string;
    };
  };
  cachedExpenses: {
    key: string;
    value: CachedExpensesRecord;
    indexes: {
      by_userId: string;
      by_tripId: string;
    };
  };
  cachedExpenseSummaries: {
    key: string;
    value: CachedExpenseSummaryRecord;
    indexes: {
      by_userId: string;
      by_tripId: string;
    };
  };
  cachedSettlements: {
    key: string;
    value: CachedSettlementsRecord;
    indexes: {
      by_userId: string;
      by_tripId: string;
    };
  };
  pendingMutations: {
    key: string;
    value: PendingOfflineMutation;
    indexes: {
      by_tripId: string;
      by_status: OfflineMutationStatus;
      by_createdAt: string;
    };
  };
  offlineReceiptDrafts: {
    key: string;
    value: OfflineReceiptDraftRecord;
    indexes: {
      by_tripId: string;
      by_userId: string;
      by_status: OfflineReceiptDraftRecord["status"];
    };
  };
  syncLogs: {
    key: string;
    value: SyncLogRecord;
    indexes: {
      by_userId: string;
      by_tripId: string;
      by_createdAt: string;
    };
  };
  offlineSettings: {
    key: string;
    value: OfflineSettingsRecord;
  };
  syncMetadata: {
    key: string;
    value: SyncMetadataRecord;
  };
}

export type OfflineDatabase = IDBPDatabase<TravelAIOfflineDB>;

let dbPromise: Promise<OfflineDatabase> | null = null;

export function getOfflineDb(): Promise<OfflineDatabase> {
  if (typeof indexedDB === "undefined") {
    throw new Error("IndexedDB is not available in this browser.");
  }

  dbPromise ??= openDB<TravelAIOfflineDB>(DB_NAME, DB_VERSION, {
    upgrade(db) {
      if (!db.objectStoreNames.contains("cachedTrips")) {
        db.createObjectStore("cachedTrips", { keyPath: "tripId" });
      }

      if (!db.objectStoreNames.contains("cachedTripDetails")) {
        const store = db.createObjectStore("cachedTripDetails", { keyPath: "cacheKey" });
        store.createIndex("by_userId", "userId");
        store.createIndex("by_tripId", "tripId");
      }

      if (!db.objectStoreNames.contains("cachedChecklists")) {
        const store = db.createObjectStore("cachedChecklists", { keyPath: "cacheKey" });
        store.createIndex("by_userId", "userId");
        store.createIndex("by_tripId", "tripId");
      }

      if (!db.objectStoreNames.contains("cachedReminders")) {
        const store = db.createObjectStore("cachedReminders", { keyPath: "cacheKey" });
        store.createIndex("by_userId", "userId");
        store.createIndex("by_tripId", "tripId");
      }

      if (!db.objectStoreNames.contains("cachedExpenses")) {
        const store = db.createObjectStore("cachedExpenses", { keyPath: "cacheKey" });
        store.createIndex("by_userId", "userId");
        store.createIndex("by_tripId", "tripId");
      }

      if (!db.objectStoreNames.contains("cachedExpenseSummaries")) {
        const store = db.createObjectStore("cachedExpenseSummaries", { keyPath: "cacheKey" });
        store.createIndex("by_userId", "userId");
        store.createIndex("by_tripId", "tripId");
      }

      if (!db.objectStoreNames.contains("cachedSettlements")) {
        const store = db.createObjectStore("cachedSettlements", { keyPath: "cacheKey" });
        store.createIndex("by_userId", "userId");
        store.createIndex("by_tripId", "tripId");
      }

      if (!db.objectStoreNames.contains("pendingMutations")) {
        const store = db.createObjectStore("pendingMutations", {
          keyPath: "mutationId"
        });
        store.createIndex("by_tripId", "tripId");
        store.createIndex("by_status", "status");
        store.createIndex("by_createdAt", "createdAt");
      }

      if (!db.objectStoreNames.contains("offlineReceiptDrafts")) {
        const store = db.createObjectStore("offlineReceiptDrafts", { keyPath: "id" });
        store.createIndex("by_tripId", "tripId");
        store.createIndex("by_userId", "userId");
        store.createIndex("by_status", "status");
      }

      if (!db.objectStoreNames.contains("syncLogs")) {
        const store = db.createObjectStore("syncLogs", { keyPath: "id" });
        store.createIndex("by_userId", "userId");
        store.createIndex("by_tripId", "tripId");
        store.createIndex("by_createdAt", "createdAt");
      }

      if (!db.objectStoreNames.contains("offlineSettings")) {
        db.createObjectStore("offlineSettings", { keyPath: "userId" });
      }

      if (!db.objectStoreNames.contains("syncMetadata")) {
        db.createObjectStore("syncMetadata", { keyPath: "key" });
      }
    }
  });

  return dbPromise;
}

export function resetOfflineDbForTests() {
  dbPromise = null;
}
