import { apiFetch } from "@/lib/api/client";
import { getNotificationApiBaseUrl } from "@/lib/config";
import type {
  AppNotification,
  MarkNotificationResponse,
  NotificationsResponse,
  UnreadNotificationsResponse
} from "@/types/notifications";

// React Query keys for notifications. Notifications are private, authenticated
// data and are never fetched from the public share page.
export const notificationKeys = {
  all: ["notifications"] as const,
  list: (params?: ListNotificationsParams) =>
    ["notifications", "list", params ?? {}] as const,
  unreadCount: ["notifications", "unread-count"] as const
};

type ListNotificationsParams = {
  limit?: number;
  cursor?: string;
};

function notificationOptions() {
  return {
    baseUrl: getNotificationApiBaseUrl(),
    serviceName: "Notification Service"
  };
}

/**
 * Fetches one newest-first page of the current user's notifications. The
 * Authorization header is attached by apiFetch. Must only be called from
 * authenticated views — never from the public share page.
 */
export async function listNotifications(
  params: ListNotificationsParams = {}
): Promise<NotificationsResponse> {
  const query = new URLSearchParams();
  if (params.limit != null) {
    query.set("limit", String(params.limit));
  }
  if (params.cursor) {
    query.set("cursor", params.cursor);
  }
  const suffix = query.toString() ? `?${query.toString()}` : "";

  const response = await apiFetch<NotificationsResponse>(
    `/notifications${suffix}`,
    {},
    notificationOptions()
  );
  return {
    items: response?.items ?? [],
    nextCursor: response?.nextCursor ?? null
  };
}

/** Returns the current user's unread notification count. */
export async function getUnreadNotificationCount(): Promise<number> {
  const response = await apiFetch<UnreadNotificationsResponse>(
    "/notifications/unread-count",
    {},
    notificationOptions()
  );
  return response?.count ?? 0;
}

/** Marks a single notification read. Idempotent. */
export function markNotificationRead(
  notificationId: string
): Promise<MarkNotificationResponse> {
  return apiFetch<MarkNotificationResponse>(
    `/notifications/${notificationId}/read`,
    { method: "PATCH" },
    notificationOptions()
  );
}

/** Marks all of the current user's unread notifications read. */
export function markAllNotificationsRead(): Promise<MarkNotificationResponse> {
  return apiFetch<MarkNotificationResponse>(
    "/notifications/read-all",
    { method: "PATCH" },
    notificationOptions()
  );
}

export type { AppNotification, ListNotificationsParams };
