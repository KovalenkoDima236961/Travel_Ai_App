"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { Button } from "@/components/ui/Button";
import { PageContainer } from "@/components/layout/PageContainer";
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  notificationKeys
} from "@/lib/api/notifications";
import { getNotificationHref } from "@/lib/notifications/notification-navigation";
import { formatRelativeTime } from "@/lib/notifications/relative-time";
import { cn } from "@/lib/utils";
import type { AppNotification } from "@/types/notifications";

const PAGE_LIMIT = 30;

function NotificationsPageContent() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [items, setItems] = useState<AppNotification[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadingMore, setLoadingMore] = useState(false);

  const loadFirstPage = useCallback(async () => {
    setStatus("loading");
    try {
      const response = await listNotifications({ limit: PAGE_LIMIT });
      setItems(response.items);
      setNextCursor(response.nextCursor ?? null);
      setStatus("ready");
    } catch {
      setStatus("error");
    }
  }, []);

  useEffect(() => {
    void loadFirstPage();
  }, [loadFirstPage]);

  async function handleLoadMore() {
    if (!nextCursor) {
      return;
    }
    setLoadingMore(true);
    try {
      const response = await listNotifications({ limit: PAGE_LIMIT, cursor: nextCursor });
      setItems((current) => [...current, ...response.items]);
      setNextCursor(response.nextCursor ?? null);
    } catch {
      // Keep what we have; the user can retry.
    } finally {
      setLoadingMore(false);
    }
  }

  async function handleSelect(notification: AppNotification) {
    if (!notification.readAt) {
      try {
        await markNotificationRead(notification.id);
        setItems((current) =>
          current.map((item) =>
            item.id === notification.id ? { ...item, readAt: new Date().toISOString() } : item
          )
        );
        void queryClient.invalidateQueries({ queryKey: notificationKeys.unreadCount });
      } catch {
        // Navigation should still proceed even if marking read fails.
      }
    }
    router.push(getNotificationHref(notification));
  }

  async function handleMarkAllRead() {
    try {
      await markAllNotificationsRead();
      const now = new Date().toISOString();
      setItems((current) => current.map((item) => ({ ...item, readAt: item.readAt ?? now })));
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    } catch {
      // No-op: leave state unchanged on failure.
    }
  }

  const hasUnread = items.some((item) => !item.readAt);

  return (
    <PageContainer>
      <div className="mx-auto max-w-2xl">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-xl font-semibold text-slate-900">Notifications</h1>
          <Button
            size="sm"
            variant="secondary"
            onClick={handleMarkAllRead}
            disabled={!hasUnread}
          >
            Mark all as read
          </Button>
        </div>

        {status === "loading" ? (
          <p className="py-10 text-center text-sm text-slate-500">Loading…</p>
        ) : status === "error" ? (
          <div className="py-10 text-center">
            <p className="text-sm text-red-600">Could not load notifications.</p>
            <Button size="sm" variant="secondary" className="mt-3" onClick={() => void loadFirstPage()}>
              Retry
            </Button>
          </div>
        ) : items.length === 0 ? (
          <p className="py-10 text-center text-sm text-slate-500">No notifications yet.</p>
        ) : (
          <>
            <ul className="divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white">
              {items.map((notification) => (
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
                      <span className="text-sm font-medium text-slate-900">{notification.title}</span>
                    </span>
                    <span className="text-sm text-slate-600">{notification.message}</span>
                    <span className="text-xs text-slate-400">
                      {formatRelativeTime(notification.createdAt)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>

            {nextCursor ? (
              <div className="mt-4 text-center">
                <Button size="sm" variant="secondary" onClick={handleLoadMore} disabled={loadingMore}>
                  {loadingMore ? "Loading…" : "Load more"}
                </Button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </PageContainer>
  );
}

export default function NotificationsPage() {
  return (
    <ProtectedRoute>
      <NotificationsPageContent />
    </ProtectedRoute>
  );
}
