"use client";

import { useCallback, useEffect, useState } from "react";
import {
  getPushPublicKey,
  getPushStatus,
  subscribePush,
  unsubscribePush
} from "@/lib/api/push-notifications";
import { registerServiceWorker } from "@/lib/push/register-service-worker";
import { urlBase64ToUint8Array } from "@/lib/push/url-base64-to-uint8-array";

const endpointStorageKey = "travel-ai:web-push-endpoint";

type PushPermission = NotificationPermission | "unsupported";

export type WebPushState = {
  supported: boolean;
  permission: PushPermission;
  enabled: boolean;
  loading: boolean;
  error: string | null;
  activeSubscriptions?: number;
  enablePush: () => Promise<void>;
  disablePush: () => Promise<void>;
  refreshStatus: () => Promise<void>;
};

export function useWebPushNotifications(): WebPushState {
  const [supported, setSupported] = useState(false);
  const [permission, setPermission] = useState<PushPermission>("unsupported");
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeSubscriptions, setActiveSubscriptions] = useState<number | undefined>();

  useEffect(() => {
    const nextSupported = isPushSupported();
    setSupported(nextSupported);
    setPermission(nextSupported ? Notification.permission : "unsupported");
  }, []);

  const refreshStatus = useCallback(async () => {
    if (!supported) {
      setPermission("unsupported");
      setEnabled(false);
      setActiveSubscriptions(0);
      return;
    }

    setPermission(Notification.permission);
    try {
      const status = await getPushStatus();
      setActiveSubscriptions(status.activeSubscriptions);
      const localSubscription = await getLocalSubscription();
      setEnabled(status.enabled && Notification.permission === "granted" && Boolean(localSubscription));
    } catch {
      const localSubscription = await getLocalSubscription();
      setEnabled(Notification.permission === "granted" && Boolean(localSubscription));
    }
  }, [supported]);

  useEffect(() => {
    void refreshStatus();
  }, [refreshStatus]);

  const enablePush = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      if (!supported) {
        throw new Error("Push notifications are not supported in this browser.");
      }

      const publicKey = await getPushPublicKey();
      if (!publicKey.enabled || !publicKey.publicKey) {
        throw new Error("Push notifications are not enabled for this environment.");
      }

      let nextPermission = Notification.permission;
      if (nextPermission === "default") {
        nextPermission = await Notification.requestPermission();
      }
      setPermission(nextPermission);

      if (nextPermission === "denied") {
        throw new Error("Notifications are blocked in your browser settings.");
      }
      if (nextPermission !== "granted") {
        throw new Error("Notification permission was not granted.");
      }

      const registration = await registerServiceWorker();
      const readyRegistration = await navigator.serviceWorker.ready;
      const activeRegistration = readyRegistration ?? registration;
      let subscription = await activeRegistration.pushManager.getSubscription();

      if (!subscription) {
        subscription = await activeRegistration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(publicKey.publicKey)
        });
      }

      await subscribePush(subscription, buildSubscriptionMetadata());
      rememberEndpoint(subscription.endpoint);
      await refreshStatus();
      setEnabled(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not enable push notifications.");
      await refreshStatus();
    } finally {
      setLoading(false);
    }
  }, [refreshStatus, supported]);

  const disablePush = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      if (!supported) {
        throw new Error("Push notifications are not supported in this browser.");
      }

      const subscription = await getLocalSubscription();
      const endpoint = subscription?.endpoint ?? rememberedEndpoint();
      if (endpoint) {
        await unsubscribePush(endpoint);
      }
      if (subscription) {
        await subscription.unsubscribe();
      }
      forgetEndpoint();
      await refreshStatus();
      setEnabled(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not disable push notifications.");
      await refreshStatus();
    } finally {
      setLoading(false);
    }
  }, [refreshStatus, supported]);

  return {
    supported,
    permission,
    enabled,
    loading,
    error,
    activeSubscriptions,
    enablePush,
    disablePush,
    refreshStatus
  };
}

function isPushSupported() {
  return (
    typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window
  );
}

async function getLocalSubscription() {
  if (!isPushSupported()) {
    return null;
  }
  try {
    const registration = await navigator.serviceWorker.getRegistration();
    return registration?.pushManager.getSubscription() ?? null;
  } catch {
    return null;
  }
}

function buildSubscriptionMetadata() {
  return {
    userAgent: navigator.userAgent,
    browser: browserName(),
    deviceLabel: deviceLabel()
  };
}

function browserName() {
  const userAgent = navigator.userAgent;
  if (userAgent.includes("Edg/")) {
    return "Edge";
  }
  if (userAgent.includes("Firefox/")) {
    return "Firefox";
  }
  if (userAgent.includes("Chrome/")) {
    return "Chrome";
  }
  if (userAgent.includes("Safari/")) {
    return "Safari";
  }
  return "Browser";
}

function deviceLabel() {
  const platform = navigator.platform || "";
  return platform ? `${browserName()} on ${platform}` : browserName();
}

function rememberEndpoint(endpoint: string) {
  try {
    window.localStorage.setItem(endpointStorageKey, endpoint);
  } catch {
    // Non-critical. Backend unsubscribe still works when the live subscription exists.
  }
}

function rememberedEndpoint() {
  try {
    return window.localStorage.getItem(endpointStorageKey);
  } catch {
    return null;
  }
}

function forgetEndpoint() {
  try {
    window.localStorage.removeItem(endpointStorageKey);
  } catch {
    // Ignore storage restrictions.
  }
}
