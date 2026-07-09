"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  notificationKeys
} from "@/lib/api/notifications";
import { getNotificationHref } from "@/lib/notifications/notification-navigation";
import { formatRelativeTime } from "@/lib/notifications/relative-time";
import type { AppNotification } from "@/entities/notification/model";
import { PAGE_LIMIT } from "../model/notificationsPageModel";
import { instrumentSans, newsreader } from "./fonts";
import { notificationVisual } from "./notification-visuals";
import { NotificationsHeader } from "./NotificationsHeader";
import { localizedNotificationTitle } from "@/lib/i18n/notifications";

const OUTLINE_PILL =
  "inline-flex h-[38px] items-center rounded-full border border-sand-400 bg-white px-4 text-[13.5px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:border-sand-400 disabled:hover:text-cocoa-700";

export function NotificationsPageContent() {
  const translateNotification = useTranslations("notifications");
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
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <NotificationsHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the <main> landmark, and nesting a second one is invalid. */}
      <div className="mx-auto max-w-[720px] px-6 pb-[72px] pt-12 sm:px-10">
        <div className="flex items-end justify-between gap-4">
          <h1 className="font-newsreader text-[40px] font-medium leading-none tracking-[-0.02em] text-cocoa-900">
            Notifications
          </h1>
          <button
            type="button"
            onClick={handleMarkAllRead}
            disabled={!hasUnread}
            className={OUTLINE_PILL}
          >
            Mark all as read
          </button>
        </div>

        {status === "loading" ? (
          <div className="mt-7 rounded-[20px] border border-sand-300 bg-white/60 p-7 text-[14.5px] text-cocoa-500">
            Loading…
          </div>
        ) : status === "error" ? (
          <div className="mt-7 rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7">
            <p className="text-[14.5px] text-clay-deep">Could not load notifications.</p>
            <button
              type="button"
              onClick={() => void loadFirstPage()}
              className={cn(OUTLINE_PILL, "mt-4")}
            >
              Retry
            </button>
          </div>
        ) : items.length === 0 ? (
          <div className="mt-7 rounded-[20px] border border-dashed border-sand-400 bg-white/60 px-8 py-14 text-center">
            <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
              You’re all caught up
            </h2>
            <p className="mx-auto mt-2 max-w-md text-[14.5px] text-cocoa-400">
              New activity across your trips and workspaces will show up here.
            </p>
          </div>
        ) : (
          <>
            <ul className="mt-7 overflow-hidden rounded-[20px] border border-sand-300 bg-white">
              {items.map((notification, index) => {
                const isUnread = !notification.readAt;
                const { Icon, tileClassName } = notificationVisual(notification.type);
                return (
                  <li key={notification.id}>
                    <button
                      type="button"
                      onClick={() => handleSelect(notification)}
                      className={cn(
                        "flex w-full items-start gap-3.5 px-[22px] py-[18px] text-left transition",
                        index > 0 && "border-t border-sand-200",
                        isUnread ? "bg-sand-100 hover:bg-sand-150" : "bg-white hover:bg-sand-50"
                      )}
                    >
                      <span
                        className={cn(
                          "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
                          tileClassName
                        )}
                      >
                        <Icon className="h-[19px] w-[19px]" />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="flex items-center gap-2">
                          {isUnread ? (
                            <span
                              className="h-2 w-2 shrink-0 rounded-full bg-clay"
                              aria-label="Unread"
                            />
                          ) : null}
                          <span className="text-[14.5px] font-semibold text-cocoa-900">
                            {localizedNotificationTitle(notification, translateNotification)}
                          </span>
                        </span>
                        <span className="mt-[3px] block text-[13.5px] leading-[1.5] text-cocoa-500">
                          {notification.message}
                        </span>
                        <span className="mt-[5px] block text-[12px] text-[#A08D78]">
                          {formatRelativeTime(notification.createdAt)}
                        </span>
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>

            {nextCursor ? (
              <div className="mt-6 text-center">
                <button
                  type="button"
                  onClick={handleLoadMore}
                  disabled={loadingMore}
                  className={OUTLINE_PILL}
                >
                  {loadingMore ? "Loading…" : "Load more"}
                </button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}
