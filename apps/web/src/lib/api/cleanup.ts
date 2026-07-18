import { apiFetch } from "@/shared/api/client";
import { getNotificationApiBaseUrl, getUserApiBaseUrl } from "@/shared/config";
import type { NotificationCleanupInput } from "@/types/data-export";

export function cleanupNotifications(input: NotificationCleanupInput) {
  return apiFetch<{ deletedOrArchivedCount: number }>("/notifications/cleanup", {
    method: "POST",
    body: JSON.stringify(input)
  }, { baseUrl: getNotificationApiBaseUrl(), serviceName: "Notification Service" });
}

export function requestAccountCleanup(input: { reason: string; exportRequestedFirst: boolean }) {
  return apiFetch<{ status: string; message: string }>("/users/me/account-cleanup/request-deletion", {
    method: "POST",
    body: JSON.stringify(input)
  }, { baseUrl: getUserApiBaseUrl(), serviceName: "User Service" });
}
