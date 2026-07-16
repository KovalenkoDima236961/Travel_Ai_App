import { ReadinessCard } from "./ReadinessCard";
import { formatActivityEvent, formatShortDate } from "@/lib/trip-command-center/format";
import type { TripActivityEvent } from "@/entities/activity/model";
import type { ReadinessCard as ReadinessCardModel } from "@/types/trip-command-center";

export function RecentActivityCard({
  card,
  activity
}: {
  card: ReadinessCardModel;
  activity: TripActivityEvent[];
}) {
  return (
    <article className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex items-center justify-between gap-3">
        <h3 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
          Recent activity
        </h3>
        {card.primaryAction ? (
          <a href={card.primaryAction.href} className="text-[13px] font-semibold text-clay">
            {card.primaryAction.label}
          </a>
        ) : null}
      </div>
      {activity.length === 0 ? (
        <p className="mt-4 text-[14px] text-cocoa-500">No recent activity yet.</p>
      ) : (
        <ol className="mt-4 space-y-3">
          {activity.slice(0, 5).map((event) => (
            <li key={event.id} className="rounded-[12px] bg-sand-50 p-3">
              <p className="text-[13px] font-semibold text-cocoa-800">
                {formatActivityEvent(event)}
              </p>
              <p className="mt-1 text-[12px] text-cocoa-400">
                {formatShortDate(event.createdAt)}
              </p>
            </li>
          ))}
        </ol>
      )}
    </article>
  );
}

export function ActivityReadinessCard({ card }: { card: ReadinessCardModel }) {
  return <ReadinessCard card={card} />;
}
