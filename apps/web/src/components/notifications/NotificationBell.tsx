"use client";

import { useEffect, useRef, useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { useUnreadNotificationCount } from "@/lib/notifications/use-notifications";
import { useNotificationStream } from "@/lib/notifications/use-notification-stream";
import { NotificationsDropdown } from "@/components/notifications/NotificationsDropdown";

/**
 * Notification bell shown in the authenticated header. It polls the unread count
 * and opens a dropdown with the latest notifications. It renders nothing for
 * unauthenticated users, so public share viewers never see it or fetch
 * notification data.
 */
export function NotificationBell() {
  const { isAuthenticated, isLoading } = useAuth();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const enabled = !isLoading && isAuthenticated;
  const stream = useNotificationStream(enabled);
  const unread = useUnreadNotificationCount(enabled, stream.isConnected);

  // Close the dropdown when clicking outside or pressing Escape.
  useEffect(() => {
    if (!open) {
      return;
    }
    function handlePointerDown(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  if (!enabled) {
    return null;
  }

  const count = unread.data ?? 0;
  const badge = count > 99 ? "99+" : String(count);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        className="relative inline-flex h-11 w-11 items-center justify-center rounded-md text-slate-700 transition hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2"
        aria-label={count > 0 ? `Notifications (${count} unread)` : "Notifications"}
        aria-haspopup="dialog"
        aria-expanded={open}
      >
        <BellIcon />
        {count > 0 ? (
          <span className="absolute -right-1 -top-1 inline-flex min-w-[1.1rem] items-center justify-center rounded-full bg-red-600 px-1 text-[0.65rem] font-semibold leading-4 text-white">
            {badge}
          </span>
        ) : null}
      </button>

      {open ? <NotificationsDropdown open={open} onClose={() => setOpen(false)} /> : null}
    </div>
  );
}

function BellIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-5 w-5"
      aria-hidden="true"
    >
      <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
      <path d="M13.73 21a2 2 0 0 1-3.46 0" />
    </svg>
  );
}
