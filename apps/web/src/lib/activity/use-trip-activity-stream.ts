"use client";

import { useEffect, useRef, useState } from "react";

import { useAuth } from "@/components/auth/AuthProvider";
import { getAccessToken } from "@/shared/api/auth";
import { getTripApiBaseUrl } from "@/shared/config";
import { parseSSEChunk } from "@/lib/notifications/sse-parser";
import type { ActivityStreamMessage, TripActivityEvent } from "@/entities/activity/model";

const INITIAL_RECONNECT_DELAY_MS = 1_000;
const MAX_RECONNECT_DELAY_MS = 30_000;
const SLOW_RECONNECT_DELAY_MS = 60_000;

type UseTripActivityStreamOptions = {
  tripId: string;
  enabled: boolean;
  accessToken?: string | null;
  onActivityCreated?: (event: TripActivityEvent) => void;
};

type TripActivityStreamState = {
  isConnected: boolean;
};

export function useTripActivityStream({
  tripId,
  enabled,
  accessToken,
  onActivityCreated
}: UseTripActivityStreamOptions): TripActivityStreamState {
  const { isAuthenticated, isLoading, user } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const onActivityCreatedRef = useRef(onActivityCreated);
  onActivityCreatedRef.current = onActivityCreated;

  const shouldConnect = enabled && Boolean(tripId) && !isLoading && isAuthenticated;

  useEffect(() => {
    if (!shouldConnect) {
      setIsConnected(false);
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
      if (malformed || eventName === "activity.heartbeat") {
        return;
      }
      if (eventName !== "activity.created" || !isActivityStreamMessage(data)) {
        return;
      }
      onActivityCreatedRef.current?.(data.event);
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
          `${getTripApiBaseUrl()}/trips/${tripId}/activity/stream`,
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

  return { isConnected };
}

function isActivityStreamMessage(data: unknown): data is ActivityStreamMessage {
  if (!data || typeof data !== "object") {
    return false;
  }
  const event = (data as Partial<ActivityStreamMessage>).event;
  return isTripActivityEvent(event);
}

function isTripActivityEvent(data: unknown): data is TripActivityEvent {
  if (!data || typeof data !== "object") {
    return false;
  }
  const event = data as Partial<TripActivityEvent>;
  return (
    typeof event.id === "string" &&
    typeof event.tripId === "string" &&
    (event.actorUserId == null || typeof event.actorUserId === "string") &&
    typeof event.eventType === "string" &&
    (event.entityType == null || typeof event.entityType === "string") &&
    (event.entityId == null || typeof event.entityId === "string") &&
    Boolean(event.metadata) &&
    typeof event.metadata === "object" &&
    typeof event.createdAt === "string"
  );
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
