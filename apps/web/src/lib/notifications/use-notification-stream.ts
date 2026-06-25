"use client";

import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/components/auth/AuthProvider";
import { notificationKeys } from "@/lib/api/notifications";
import { getAccessToken } from "@/lib/auth/token-storage";
import { getNotificationApiBaseUrl } from "@/lib/config";
import { parseSSEChunk } from "@/lib/notifications/sse-parser";
import type { NotificationCreatedStreamPayload } from "@/types/notifications";

const INITIAL_RECONNECT_DELAY_MS = 1_000;
const MAX_RECONNECT_DELAY_MS = 30_000;

type NotificationStreamState = {
  isConnected: boolean;
};

export function useNotificationStream(enabled: boolean): NotificationStreamState {
  const { isAuthenticated, isLoading, user } = useAuth();
  const queryClient = useQueryClient();
  const [isConnected, setIsConnected] = useState(false);
  const shouldConnect = enabled && !isLoading && isAuthenticated;

  useEffect(() => {
    if (!shouldConnect) {
      setIsConnected(false);
      return;
    }

    let stopped = false;
    let reconnectDelay = INITIAL_RECONNECT_DELAY_MS;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let abortController: AbortController | null = null;

    function scheduleReconnect() {
      if (stopped) {
        return;
      }
      setIsConnected(false);
      reconnectTimer = setTimeout(() => {
        void connect();
      }, reconnectDelay);
      reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY_MS);
    }

    function handleEvent(eventName: string, data: unknown, malformed: boolean) {
      if (eventName === "heartbeat" || malformed) {
        return;
      }
      if (eventName !== "notification.created" || !isNotificationCreatedPayload(data)) {
        return;
      }
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }

    async function connect() {
      if (stopped) {
        return;
      }

      const accessToken = getAccessToken();
      if (!accessToken) {
        setIsConnected(false);
        return;
      }

      abortController = new AbortController();
      let buffer = "";
      const decoder = new TextDecoder();

      try {
        const response = await fetch(`${getNotificationApiBaseUrl()}/notifications/stream`, {
          method: "GET",
          headers: {
            Accept: "text/event-stream",
            Authorization: `Bearer ${accessToken}`
          },
          cache: "no-store",
          signal: abortController.signal
        });

        if (response.status === 401) {
          setIsConnected(false);
          window.dispatchEvent(new Event("auth:session-expired"));
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
  }, [queryClient, shouldConnect, user?.id]);

  return { isConnected };
}

function isNotificationCreatedPayload(data: unknown): data is NotificationCreatedStreamPayload {
  if (!data || typeof data !== "object") {
    return false;
  }
  const notification = (data as Partial<NotificationCreatedStreamPayload>).notification;
  return Boolean(
    notification &&
      typeof notification === "object" &&
      typeof notification.id === "string" &&
      typeof notification.userId === "string" &&
      typeof notification.type === "string" &&
      typeof notification.title === "string" &&
      typeof notification.message === "string"
  );
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
