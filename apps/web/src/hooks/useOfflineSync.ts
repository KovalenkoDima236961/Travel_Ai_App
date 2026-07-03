"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  OFFLINE_QUEUE_CHANGED_EVENT,
  getPendingMutations,
  syncPendingMutations
} from "@/lib/offline/sync-queue";
import type { PendingItineraryMutation, SyncResult } from "@/lib/offline/types";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";

type UseOfflineSyncOptions = {
  userId?: string | null;
  enabled?: boolean;
  onSyncResults?: (results: SyncResult[]) => void;
};

export function useOfflineSync({
  userId,
  enabled = true,
  onSyncResults
}: UseOfflineSyncOptions) {
  const { online } = useNetworkStatus();
  const [mutations, setMutations] = useState<PendingItineraryMutation[]>([]);
  const [syncing, setSyncing] = useState(false);
  const syncingRef = useRef(false);
  const onSyncResultsRef = useRef(onSyncResults);
  onSyncResultsRef.current = onSyncResults;

  const refresh = useCallback(async () => {
    if (!enabled || !userId) {
      setMutations([]);
      return;
    }

    try {
      setMutations(await getPendingMutations(userId));
    } catch {
      setMutations([]);
    }
  }, [enabled, userId]);

  const syncNow = useCallback(async () => {
    if (!enabled || !userId || !online || syncingRef.current) {
      return;
    }

    syncingRef.current = true;
    setSyncing(true);
    try {
      const results = await syncPendingMutations(userId);
      onSyncResultsRef.current?.(results);
      await refresh();
    } finally {
      syncingRef.current = false;
      setSyncing(false);
    }
  }, [enabled, online, refresh, userId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    function handleQueueChanged() {
      void refresh();
    }

    window.addEventListener(OFFLINE_QUEUE_CHANGED_EVENT, handleQueueChanged);
    return () => {
      window.removeEventListener(OFFLINE_QUEUE_CHANGED_EVENT, handleQueueChanged);
    };
  }, [refresh]);

  useEffect(() => {
    if (enabled && userId && online) {
      void syncNow();
    }
  }, [enabled, online, syncNow, userId]);

  const conflicts = useMemo(
    () => mutations.filter((mutation) => mutation.status === "conflict"),
    [mutations]
  );
  const failed = useMemo(
    () => mutations.filter((mutation) => mutation.status === "failed"),
    [mutations]
  );

  return {
    pendingCount: mutations.length,
    syncing,
    conflicts,
    failed,
    mutations,
    refresh,
    syncNow
  };
}
