"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import {
  acquireTripEditLock,
  getTripEditLock,
  releaseTripEditLock
} from "@/lib/api/edit-locks";
import type { AcquireEditLockResponse, EditLockView } from "@/types/edit-locks";

const RENEW_INTERVAL_MS = 45_000;

type UseTripEditLockInput = {
  tripId: string;
  enabled: boolean;
  canEdit: boolean;
  onLockConflict?: (lock: EditLockView) => void;
};

type UseTripEditLockResult = {
  lock: EditLockView | null;
  loading: boolean;
  error: string | null;
  acquire: () => Promise<AcquireEditLockResponse>;
  release: () => Promise<void>;
  startRenewal: () => void;
  stopRenewal: () => void;
  refetch: () => Promise<void>;
};

export function useTripEditLock({
  tripId,
  enabled,
  canEdit,
  onLockConflict
}: UseTripEditLockInput): UseTripEditLockResult {
  const [lock, setLock] = useState<EditLockView | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const ownsLockRef = useRef(false);
  const renewingRef = useRef(false);
  const onLockConflictRef = useRef(onLockConflict);

  useEffect(() => {
    onLockConflictRef.current = onLockConflict;
  }, [onLockConflict]);

  const stopRenewal = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  const renew = useCallback(async () => {
    if (!enabled || !canEdit || !tripId || renewingRef.current) {
      return;
    }
    renewingRef.current = true;
    try {
      const result = await acquireTripEditLock(tripId);
      setLock(result.lock ?? null);
      if (!result.acquired) {
        ownsLockRef.current = false;
        stopRenewal();
        if (result.lock) {
          onLockConflictRef.current?.(result.lock);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not renew edit lock.");
    } finally {
      renewingRef.current = false;
    }
  }, [canEdit, enabled, stopRenewal, tripId]);

  const startRenewal = useCallback(() => {
    if (!enabled || !canEdit || !tripId || intervalRef.current) {
      return;
    }
    intervalRef.current = setInterval(() => {
      void renew();
    }, RENEW_INTERVAL_MS);
  }, [canEdit, enabled, renew, tripId]);

  const refetch = useCallback(async () => {
    if (!enabled || !tripId) {
      setLock(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const nextLock = await getTripEditLock(tripId);
      setLock(nextLock);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load edit lock.");
    } finally {
      setLoading(false);
    }
  }, [enabled, tripId]);

  const acquire = useCallback(async () => {
    if (!enabled || !canEdit || !tripId) {
      return {
        acquired: false,
        reason: "edit_not_allowed",
        lock
      };
    }
    setLoading(true);
    setError(null);
    try {
      const result = await acquireTripEditLock(tripId);
      setLock(result.lock ?? null);
      ownsLockRef.current = Boolean(result.acquired && !result.disabled);
      if (result.acquired && !result.disabled) {
        startRenewal();
      } else if (result.lock) {
        onLockConflictRef.current?.(result.lock);
      }
      return result;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Could not acquire edit lock.";
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [canEdit, enabled, lock, startRenewal, tripId]);

  const release = useCallback(async () => {
    stopRenewal();
    if (!enabled || !tripId || !ownsLockRef.current) {
      ownsLockRef.current = false;
      return;
    }
    try {
      await releaseTripEditLock(tripId);
      ownsLockRef.current = false;
      await refetch();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not release edit lock.");
      ownsLockRef.current = false;
    }
  }, [enabled, refetch, stopRenewal, tripId]);

  useEffect(() => {
    void refetch();
  }, [refetch]);

  useEffect(() => {
    return () => {
      stopRenewal();
      if (enabled && tripId && ownsLockRef.current) {
        ownsLockRef.current = false;
        void releaseTripEditLock(tripId);
      }
    };
  }, [enabled, stopRenewal, tripId]);

  return {
    lock,
    loading,
    error,
    acquire,
    release,
    startRenewal,
    stopRenewal,
    refetch
  };
}
