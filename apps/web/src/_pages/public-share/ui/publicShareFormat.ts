import { getCostAmount } from "@/entities/budget/model";
import type { Itinerary } from "@/entities/trip/model";

/**
 * Renders a trip's date span like the mock ("Sep 14 – 18, 2026 · 4 days"),
 * collapsing the month/year when the range stays within one. Falls back to just
 * the duration when there is no start date. Mirrors trip-detail's helper so the
 * two redesigned screens format spans identically.
 */
export function formatTripDateRange(
  startDate: string | null | undefined,
  days: number
): string {
  const durationLabel = `${days} ${days === 1 ? "day" : "days"}`;

  if (!startDate) {
    return durationLabel;
  }

  const start = new Date(startDate);
  if (Number.isNaN(start.getTime())) {
    return durationLabel;
  }

  const end = new Date(start);
  end.setDate(start.getDate() + Math.max(0, days - 1));

  const month = (date: Date) =>
    new Intl.DateTimeFormat("en", { month: "short" }).format(date);
  const dayNum = (date: Date) => date.getDate();
  const year = end.getFullYear();

  const sameMonth =
    start.getMonth() === end.getMonth() && start.getFullYear() === end.getFullYear();

  const range =
    days <= 1
      ? `${month(start)} ${dayNum(start)}, ${year}`
      : sameMonth
        ? `${month(start)} ${dayNum(start)} – ${dayNum(end)}, ${year}`
        : `${month(start)} ${dayNum(start)} – ${month(end)} ${dayNum(end)}, ${year}`;

  return `${range} · ${durationLabel}`;
}

/**
 * Per-day heading date like the mock ("Sun, Sep 14"). Returns "Day N" when there
 * is no usable start date.
 */
export function formatDayDate(
  startDate: string | null | undefined,
  dayNumber: number
): string {
  if (!startDate) {
    return `Day ${dayNumber}`;
  }
  const start = new Date(startDate);
  if (Number.isNaN(start.getTime())) {
    return `Day ${dayNumber}`;
  }
  const date = new Date(start);
  date.setDate(start.getDate() + Math.max(0, dayNumber - 1));
  return new Intl.DateTimeFormat("en", {
    weekday: "short",
    month: "short",
    day: "numeric"
  }).format(date);
}

/**
 * Sums the estimated cost across every itinerary item to back the summary card's
 * "Estimated" figure. PublicTrip omits the private trip budget, so this itinerary
 * roll-up is the only cost signal available. Returns null when nothing is priced,
 * so the caller can omit the row rather than showing a fabricated total.
 */
export function estimateItineraryTotal(itinerary: Itinerary | null | undefined): number | null {
  if (!itinerary?.days?.length) {
    return null;
  }
  let total = 0;
  let hasAny = false;
  for (const day of itinerary.days) {
    for (const item of day.items ?? []) {
      const amount = getCostAmount(item.estimatedCost);
      if (amount != null) {
        total += amount;
        hasAny = true;
      }
    }
  }
  return hasAny ? total : null;
}
