import { openDB, type DBSchema, type IDBPDatabase } from "idb";
import type {
  CachedTripRecord,
  OfflineMutationStatus,
  PendingItineraryMutation,
  SyncMetadataRecord
} from "@/lib/offline/types";

const DB_NAME = "travel-ai-offline-v1";
const DB_VERSION = 1;

interface TravelAIOfflineDB extends DBSchema {
  cachedTrips: {
    key: string;
    value: CachedTripRecord;
  };
  pendingMutations: {
    key: string;
    value: PendingItineraryMutation;
    indexes: {
      by_tripId: string;
      by_status: OfflineMutationStatus;
      by_createdAt: string;
    };
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

      if (!db.objectStoreNames.contains("pendingMutations")) {
        const store = db.createObjectStore("pendingMutations", {
          keyPath: "mutationId"
        });
        store.createIndex("by_tripId", "tripId");
        store.createIndex("by_status", "status");
        store.createIndex("by_createdAt", "createdAt");
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
