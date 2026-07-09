"use client";

import { useRouter } from "next/navigation";
import { useNotificationsList, useMarkAllNotificationsRead, useMarkNotificationRead } from "@/lib/notifications/use-notifications";
import { getNotificationHref } from "@/lib/notifications/notification-navigation";
import { formatRelativeTime } from "@/lib/notifications/relative-time";
import { cn } from "@/lib/utils";
import type { AppNotification } from "@/entities/notification/model";
import { useTranslations } from "next-intl";
import { localizedNotificationTitle } from "@/lib/i18n/notifications";

const DROPDOWN_LIMIT = 10;

type NotificationsDropdownProps = {
  open: boolean;
  onClose: () => void;
};

export function NotificationsDropdown({ open, onClose }: NotificationsDropdownProps) {
  const translateNotification = useTranslations("notifications");
  const router = useRouter();
  const list = useNotificationsList({ limit: DROPDOWN_LIMIT }, open);
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  const notifications = list.data?.items ?? [];
  const hasUnread = notifications.some((n) => !n.readAt);

  function handleSelect(notification: AppNotification) {
    if (!notification.readAt) {
      markRead.mutate(notification.id);
    }
    onClose();
    router.push(getNotificationHref(notification));
  }

  return (
    <div
      className="absolute right-0 z-50 mt-2 w-80 max-w-[calc(100vw-2rem)] overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg"
      role="dialog"
      aria-label="Notifications"
    >
      <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3">
        <span className="text-sm font-semibold text-slate-900">Notifications</span>
        <button
          type="button"
          className="text-xs font-medium text-primary-600 hover:text-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
          onClick={() => markAllRead.mutate()}
          disabled={!hasUnread || markAllRead.isPending}
        >
          Mark all as read
        </button>
      </div>

      <div className="max-h-96 overflow-y-auto">
        {list.isLoading ? (
          <p className="px-4 py-6 text-center text-sm text-slate-500">Loading…</p>
        ) : list.isError ? (
          <p className="px-4 py-6 text-center text-sm text-red-600">
            Could not load notifications.
          </p>
        ) : notifications.length === 0 ? (
          <p className="px-4 py-6 text-center text-sm text-slate-500">No notifications yet.</p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {notifications.map((notification) => (
              <li key={notification.id}>
                <button
                  type="button"
                  onClick={() => handleSelect(notification)}
                  className={cn(
                    "flex w-full flex-col gap-1 px-4 py-3 text-left transition hover:bg-slate-50",
                    !notification.readAt && "bg-primary-50/60"
                  )}
                >
                  <span className="flex items-center gap-2">
                    {!notification.readAt ? (
                      <span
                        className="h-2 w-2 shrink-0 rounded-full bg-primary-600"
                        aria-label="Unread"
                      />
                    ) : null}
                    <span className="text-sm font-medium text-slate-900">
                      {localizedNotificationTitle(notification, translateNotification)}
                    </span>
                  </span>
                  <span className="text-sm text-slate-600">{notification.message}</span>
                  <span className="text-xs text-slate-400">
                    {formatRelativeTime(notification.createdAt)}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="border-t border-slate-100 px-4 py-2 text-center">
        <a
          href="/notifications"
          className="text-xs font-medium text-primary-600 hover:text-primary-700"
          onClick={onClose}
        >
          View all
        </a>
      </div>
    </div>
  );
}
