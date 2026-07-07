"use client";

import { useEffect, useRef, useState } from "react";

import { useAuth } from "@/components/auth/AuthProvider";
import { getAccessToken } from "@/shared/api/auth";
import { getTripApiBaseUrl } from "@/shared/config";
import { parseSSEChunk } from "@/lib/notifications/sse-parser";
import type { TripPresenceSnapshot } from "@/entities/presence/model";

const INITIAL_RECONNECT_DELAY_MS = 1_000;
const MAX_RECONNECT_DELAY_MS = 30_000;
const SLOW_RECONNECT_DELAY_MS = 60_000;

type UseTripPresenceStreamOptions = {
  tripId: string;
  enabled: boolean;
  accessToken?: string | null;
  onSnapshot?: (snapshot: TripPresenceSnapshot) => void;
};

type TripPresenceStreamState = {
  isConnected: boolean;
  snapshot: TripPresenceSnapshot | null;
};

export function useTripPresenceStream({
  tripId,
  enabled,
  accessToken,
  onSnapshot
}: UseTripPresenceStreamOptions): TripPresenceStreamState {
  const { isAuthenticated, isLoading, user } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const [snapshot, setSnapshot] = useState<TripPresenceSnapshot | null>(null);
  const onSnapshotRef = useRef(onSnapshot);
  onSnapshotRef.current = onSnapshot;

  const shouldConnect = enabled && Boolean(tripId) && !isLoading && isAuthenticated;

  useEffect(() => {
    if (!shouldConnect) {
      setIsConnected(false);
      setSnapshot(null);
      return;
    }

    let stopped = false;
    let reconnectDelay = INITIAL_RECONNECT_DELAY_MS;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let abortController: AbortController | null = null;

    function scheduleReconnect(delay = reconnectDelay) {
      if (stopped) {
        return;
      }
      setIsConnected(false);
      reconnectTimer = setTimeout(() => {
        void connect();
      }, delay);
      reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY_MS);
    }

    function handleEvent(eventName: string, data: unknown, malformed: boolean) {
      if (malformed || eventName === "presence.heartbeat") {
        return;
      }
      if (eventName !== "presence.snapshot" || !isTripPresenceSnapshot(data)) {
        return;
      }
      setSnapshot(data);
      onSnapshotRef.current?.(data);
    }

    async function connect() {
      if (stopped) {
        return;
      }

      const token = accessToken ?? getAccessToken();
      if (!token) {
        setIsConnected(false);
        return;
      }

      abortController = new AbortController();
      let buffer = "";
      const decoder = new TextDecoder();

      try {
        const response = await fetch(
          `${getTripApiBaseUrl()}/trips/${tripId}/presence/stream`,
          {
            method: "GET",
            headers: {
              Accept: "text/event-stream",
              Authorization: `Bearer ${token}`
            },
            cache: "no-store",
            signal: abortController.signal
          }
        );

        if (response.status === 401) {
          setIsConnected(false);
          window.dispatchEvent(new Event("auth:session-expired"));
          return;
        }
        if (response.status === 403 || response.status === 404) {
          setIsConnected(false);
          return;
        }
        if (response.status === 503) {
          scheduleReconnect(SLOW_RECONNECT_DELAY_MS);
          return;
        }
        if (!response.ok || !response.body) {
          scheduleReconnect();
          return;
        }

        reconnectDelay = INITIAL_RECONNECT_DELAY_MS;
        setIsConnected(true);

        const reader = response.body.getReader();
        try {
          while (!stopped) {
            const { done, value } = await reader.read();
            if (done) {
              break;
            }
            const parsed = parseSSEChunk(buffer, decoder.decode(value, { stream: true }));
            buffer = parsed.remainder;
            for (const event of parsed.events) {
              handleEvent(event.event, event.data, event.malformed);
            }
          }
        } finally {
          reader.releaseLock();
        }

        if (!stopped) {
          scheduleReconnect();
        }
      } catch (error) {
        if (stopped || isAbortError(error)) {
          return;
        }
        scheduleReconnect();
      }
    }

    void connect();

    return () => {
      stopped = true;
      setIsConnected(false);
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      abortController?.abort();
    };
  }, [accessToken, shouldConnect, tripId, user?.id]);

  return { isConnected, snapshot };
}

function isTripPresenceSnapshot(data: unknown): data is TripPresenceSnapshot {
  if (!data || typeof data !== "object") {
    return false;
  }
  const candidate = data as Partial<TripPresenceSnapshot>;
  return (
    typeof candidate.tripId === "string" &&
    Array.isArray(candidate.users) &&
    candidate.users.every(isTripPresenceUser)
  );
}

function isTripPresenceUser(data: unknown) {
  if (!data || typeof data !== "object") {
    return false;
  }
  const user = data as Partial<TripPresenceSnapshot["users"][number]>;
  return (
    typeof user.userId === "string" &&
    (user.displayName == null || typeof user.displayName === "string") &&
    (user.role === "owner" || user.role === "editor" || user.role === "viewer") &&
    (user.state === "viewing" || user.state === "editing") &&
    typeof user.connectedAt === "string" &&
    typeof user.lastSeenAt === "string"
  );
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
