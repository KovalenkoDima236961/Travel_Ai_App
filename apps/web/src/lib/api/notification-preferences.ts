import { apiFetch } from "@/lib/api/client";
import { getNotificationApiBaseUrl } from "@/lib/config";
import type {
  NotificationPreference,
  NotificationPreferencesResponse
} from "@/types/notification-preferences";

export const notificationPreferenceKeys = {
  all: ["notification-preferences"] as const
};

function notificationOptions() {
  return {
    baseUrl: getNotificationApiBaseUrl(),
    serviceName: "Notification Service"
  };
}

export async function getNotificationPreferences(): Promise<NotificationPreference[]> {
  const response = await apiFetch<NotificationPreferencesResponse>(
    "/notifications/preferences",
    {},
    notificationOptions()
  );
  return response?.items ?? [];
}

export async function updateNotificationPreferences(
  items: NotificationPreference[]
): Promise<NotificationPreference[]> {
  const response = await apiFetch<NotificationPreferencesResponse>(
    "/notifications/preferences",
    {
      method: "PUT",
      body: JSON.stringify({ items })
    },
    notificationOptions()
  );
  return response?.items ?? [];
}
