"use client";

import Link from "next/link";

export function NotificationBulkActions({ hasUnread, onMarkAllRead }: { hasUnread: boolean; onMarkAllRead: ()=>void }) {
  return <div className="flex flex-wrap justify-end gap-2">
    <Link href="/settings#notifications" className="inline-flex h-[38px] items-center rounded-full border border-sand-400 bg-white px-4 text-[13.5px] font-medium text-cocoa-700 transition hover:border-sand-600">Notification settings</Link>
    <button type="button" onClick={onMarkAllRead} disabled={!hasUnread} className="inline-flex h-[38px] items-center rounded-full border border-sand-400 bg-white px-4 text-[13.5px] font-medium text-cocoa-700 transition hover:border-sand-600 disabled:opacity-50">Mark all as read</button>
  </div>;
}
