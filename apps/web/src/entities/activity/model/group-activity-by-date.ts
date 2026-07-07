import type { TripActivityEvent } from "@/entities/activity/model";

export type ActivityDateGroup = {
  /** Stable key for the calendar day (YYYY-MM-DD in local time). */
  key: string;
  /** Display label: "Today", "Yesterday", or a formatted date. */
  label: string;
  events: TripActivityEvent[];
};

/**
 * Groups activity events by calendar day (local time), preserving the incoming
 * newest-first order both across and within groups. Events with an unparseable
 * timestamp are collected under an "Earlier" bucket rather than dropped.
 */
export function groupActivityByDate(
  events: TripActivityEvent[],
  now: Date = new Date()
): ActivityDateGroup[] {
  const groups: ActivityDateGroup[] = [];
  const indexByKey = new Map<string, number>();

  for (const event of events) {
    const date = new Date(event.createdAt);
    const valid = !Number.isNaN(date.getTime());
    const key = valid ? dayKey(date) : "unknown";

    let groupIndex = indexByKey.get(key);
    if (groupIndex == null) {
      groupIndex = groups.length;
      indexByKey.set(key, groupIndex);
      groups.push({
        key,
        label: valid ? dayLabel(date, now) : "Earlier",
        events: []
      });
    }
    groups[groupIndex].events.push(event);
  }

  return groups;
}

function dayKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function dayLabel(date: Date, now: Date): string {
  if (dayKey(date) === dayKey(now)) {
    return "Today";
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (dayKey(date) === dayKey(yesterday)) {
    return "Yesterday";
  }
  return new Intl.DateTimeFormat("en", {
    year: "numeric",
    month: "short",
    day: "numeric"
  }).format(date);
}
