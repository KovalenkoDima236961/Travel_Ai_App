"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@/shared/ui/button";
import { activityKeys, listTripActivity } from "@/lib/api/activity";
import { formatActivityEvent } from "@/entities/activity/model";
import { groupActivityByDate } from "@/entities/activity/model";
import { formatDate } from "@/lib/utils";
import type { TripActivityEvent } from "@/entities/activity/model";

const PAGE_SIZE = 30;

type ActivityFeedProps = {
  tripId: string;
  currentUserId?: string | null;
  canViewActivity: boolean;
};

/**
 * Recent activity panel for a private trip. Owners and accepted collaborators
 * see a chronological, newest-first feed grouped by day, with cursor-based
 * "Load more" paging. It is never rendered on the public share page and renders
 * nothing when the viewer lacks access. Private trip pages invalidate this
 * query from the activity SSE stream when new events arrive.
 */
export function ActivityFeed({ tripId, currentUserId, canViewActivity }: ActivityFeedProps) {
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
  const groups = groupActivityByDate(events);

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5">
      <h2 className="text-lg font-semibold text-slate-950">Recent activity</h2>
      <p className="mt-1 text-sm text-slate-500">
        Important changes to this trip, newest first.
      </p>

      <div className="mt-4 space-y-4">
        {query.isPending ? (
          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
            Loading activity...
          </div>
        ) : null}

        {query.isError ? (
          <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {query.error instanceof Error ? query.error.message : "Could not load activity."}
          </div>
        ) : null}

        {!query.isPending && !query.isError && events.length === 0 ? (
          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
            No activity yet.
          </div>
        ) : null}

        {groups.map((group) => (
          <div key={group.key}>
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-400">
              {group.label}
            </h3>
            <ul className="mt-2 space-y-2">
              {group.events.map((event) => (
                <ActivityRow
                  currentUserId={currentUserId}
                  event={event}
                  key={event.id}
                />
              ))}
            </ul>
          </div>
        ))}

        {query.hasNextPage ? (
          <Button
            disabled={query.isFetchingNextPage}
            onClick={() => query.fetchNextPage()}
            type="button"
            variant="secondary"
          >
            {query.isFetchingNextPage ? "Loading..." : "Load more"}
          </Button>
        ) : null}
      </div>
    </section>
  );
}

type ActivityRowProps = {
  event: TripActivityEvent;
  currentUserId?: string | null;
};

function ActivityRow({ event, currentUserId }: ActivityRowProps) {
  const formatted = formatActivityEvent(event, currentUserId);

  return (
    <li
      className="scroll-mt-28 flex items-start gap-3 rounded-lg border border-slate-100 bg-slate-50 p-3 outline-none transition-shadow"
      id={`activity-event-${event.id}`}
    >
      <span
        aria-hidden="true"
        className="mt-1.5 h-2 w-2 flex-shrink-0 rounded-full bg-slate-400"
      />
      <div className="min-w-0 flex-1">
        <p className="text-sm text-slate-800">{formatted.title}</p>
        {formatted.description ? (
          <p className="mt-0.5 text-sm text-slate-500">{formatted.description}</p>
        ) : null}
        <p className="mt-0.5 text-xs text-slate-400">
          {formatDate(event.createdAt, { dateStyle: "medium", timeStyle: "short" })}
        </p>
      </div>
    </li>
  );
}
