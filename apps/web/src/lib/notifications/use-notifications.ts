"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getUnreadNotificationCount,
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  notificationKeys,
  type ListNotificationsParams
} from "@/lib/api/notifications";

const UNREAD_POLL_INTERVAL_MS = 45_000;

/**
 * Polls the current user's unread notification count. Polling is enabled only
 * for authenticated users so logged-out / public share viewers never call the
 * Notification Service.
 */
export function useUnreadNotificationCount(enabled: boolean) {
  return useQuery({
    queryKey: notificationKeys.unreadCount,
    queryFn: getUnreadNotificationCount,
    enabled,
    refetchInterval: enabled ? UNREAD_POLL_INTERVAL_MS : false,
    refetchOnWindowFocus: true
  });
}

/** Fetches a page of the current user's notifications. */
export function useNotificationsList(params: ListNotificationsParams, enabled: boolean) {
  return useQuery({
    queryKey: notificationKeys.list(params),
    queryFn: () => listNotifications(params),
    enabled
  });
}

/** Marks one notification read and refreshes the list + unread count. */
export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (notificationId: string) => markNotificationRead(notificationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  });
}

/** Marks all notifications read and refreshes the list + unread count. */
export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => markAllNotificationsRead(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  });
}
