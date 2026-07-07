import { apiFetch, apiFetchPublic } from "@/shared/api/client";
import { getNotificationApiBaseUrl } from "@/shared/config";

export const pushNotificationKeys = {
  publicKey: ["push-notifications", "public-key"] as const,
  status: ["push-notifications", "status"] as const
};

export type PushPublicKeyResponse = {
  enabled: boolean;
  publicKey: string | null;
};

export type PushSubscribeResponse = {
  subscribed: boolean;
  enabled: boolean;
};

export type PushStatusResponse = {
  enabled: boolean;
  activeSubscriptions: number;
};

export type PushSubscriptionMetadata = {
  userAgent?: string;
  browser?: string;
  deviceLabel?: string;
};

function notificationOptions() {
  return {
    baseUrl: getNotificationApiBaseUrl(),
    serviceName: "Notification Service"
  };
}

export function getPushPublicKey(): Promise<PushPublicKeyResponse> {
  return apiFetchPublic<PushPublicKeyResponse>(
    "/notifications/push/public-key",
    {},
    notificationOptions()
  );
}

export function subscribePush(
  subscription: PushSubscription,
  metadata: PushSubscriptionMetadata = {}
): Promise<PushSubscribeResponse> {
  return apiFetch<PushSubscribeResponse>(
    "/notifications/push/subscribe",
    {
      method: "POST",
      body: JSON.stringify({
        subscription,
        ...metadata
      })
    },
    notificationOptions()
  );
}

export function unsubscribePush(endpoint: string): Promise<{ unsubscribed: boolean }> {
  return apiFetch<{ unsubscribed: boolean }>(
    "/notifications/push/unsubscribe",
    {
      method: "DELETE",
      body: JSON.stringify({ endpoint })
    },
    notificationOptions()
  );
}

export function getPushStatus(): Promise<PushStatusResponse> {
  return apiFetch<PushStatusResponse>(
    "/notifications/push/status",
    {},
    notificationOptions()
  );
}
