import { apiFetch } from "@/shared/api/client";
import { getNotificationApiBaseUrl } from "@/shared/config";
import type {
  NotificationPreference,
  NotificationPreferencesResponse,
  NotificationSettings,
  NotificationTripMute,
  NotificationTripMutesResponse,
  NotificationCategory
} from "@/entities/notification-preferences/model";

export const notificationPreferenceKeys = {
  all: ["notification-preferences"] as const,
  tripMutes: (tripId: string) => ["notification-preferences", "trip-mutes", tripId] as const
};
const options = () => ({ baseUrl: getNotificationApiBaseUrl(), serviceName: "Notification Service" });

export async function getNotificationPreferences(): Promise<NotificationPreferencesResponse> {
  return apiFetch<NotificationPreferencesResponse>("/notifications/preferences", {}, options());
}
export async function updateNotificationPreferences(input: {
  items: NotificationPreference[];
  settings: NotificationSettings;
}): Promise<NotificationPreferencesResponse> {
  return apiFetch<NotificationPreferencesResponse>("/notifications/preferences", { method: "PUT", body: JSON.stringify(input) }, options());
}
export async function getTripNotificationMutes(tripId: string): Promise<NotificationTripMute[]> {
  const response = await apiFetch<NotificationTripMutesResponse>(`/notifications/trip-mutes?tripId=${encodeURIComponent(tripId)}`, {}, options());
  return response.items ?? [];
}
export async function upsertTripNotificationMute(input: { tripId: string; category?: NotificationCategory | null; mutedUntil?: string | null }): Promise<NotificationTripMute> {
  return apiFetch<NotificationTripMute>("/notifications/trip-mutes", { method: "PUT", body: JSON.stringify(input) }, options());
}
export async function deleteTripNotificationMute(muteId: string): Promise<{ success: boolean }> {
  return apiFetch<{ success: boolean }>(`/notifications/trip-mutes/${muteId}`, { method: "DELETE" }, options());
}
