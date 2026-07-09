"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { activityKeys, listTripActivity } from "@/lib/api/activity";
import { formatActivityEvent } from "@/entities/activity/model";
import type { TripActivityEvent } from "@/entities/activity/model";

const PAGE_SIZE = 30;

type RightRailActivityProps = {
  tripId: string;
  currentUserId?: string | null;
  canViewActivity: boolean;
};

/**
 * Warm right-rail activity card, forked from the shared ActivityFeed. Same
 * cursor-paged query and event formatting; renders nothing when the viewer lacks
 * access. Replaces the shared feed on this screen (no duplicate list) so paging is
 * preserved via the in-place "View more" control.
 */
export function RightRailActivity({
  tripId,
  currentUserId,
  canViewActivity
}: RightRailActivityProps) {
  const enabled = canViewActivity && Boolean(tripId);
  const query = useInfiniteQuery({
    queryKey: activityKeys.all(tripId),
    queryFn: ({ pageParam }) =>
      listTripActivity(tripId, { limit: PAGE_SIZE, cursor: pageParam || undefined }),
    initialPageParam: "" as string,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    enabled
  });

  if (!canViewActivity) {
    return null;
  }

  const events = query.data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <div id="activity" className="scroll-mt-24 rounded-[20px] border border-sand-300 bg-white px-[22px] py-5">
      <h2 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
        Recent activity
      </h2>

      <div className="mt-3.5 flex flex-col gap-3">
        {query.isPending ? (
          <p className="text-[13px] text-cocoa-400">Loading activity…</p>
        ) : null}
        {query.isError ? (
          <p className="text-[13px] text-[#B3402E]">
            {query.error instanceof Error ? query.error.message : "Could not load activity."}
          </p>
        ) : null}
        {!query.isPending && !query.isError && events.length === 0 ? (
          <p className="text-[13px] text-cocoa-400">No activity yet.</p>
        ) : null}

        {events.map((event) => (
          <ActivityRow key={event.id} event={event} currentUserId={currentUserId} />
        ))}
      </div>

      {query.hasNextPage ? (
        <button
          type="button"
          disabled={query.isFetchingNextPage}
          onClick={() => query.fetchNextPage()}
          className="mt-3.5 inline-block text-[13px] font-semibold text-clay-deep transition hover:text-clay disabled:opacity-60"
        >
          {query.isFetchingNextPage ? "Loading…" : "View all activity"}
        </button>
      ) : null}
    </div>
  );
}

function ActivityRow({
  event,
  currentUserId
}: {
  event: TripActivityEvent;
  currentUserId?: string | null;
}) {
  const formatted = formatActivityEvent(event, currentUserId);
  return (
    <div className="flex items-start gap-2.5">
      <span
        aria-hidden="true"
        className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-clay"
      />
      <div className="min-w-0">
        <p className="text-[13px] leading-[1.5] text-cocoa-500">
          <span className="font-semibold text-cocoa-900">{formatted.title}</span>
          {formatted.description ? ` — ${formatted.description}` : ""}
        </p>
        <p className="mt-0.5 text-[11.5px] text-[#A08D78]">{relativeTime(event.createdAt)}</p>
      </div>
    </div>
  );
}

function relativeTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const diffMs = Date.now() - date.getTime();
  const minutes = Math.round(diffMs / 60000);
  if (minutes < 1) {
    return "just now";
  }
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 24) {
    return `${hours}h ago`;
  }
  const dayCount = Math.round(hours / 24);
  if (dayCount < 7) {
    return `${dayCount}d ago`;
  }
  return new Intl.DateTimeFormat("en", { month: "short", day: "numeric" }).format(date);
}
