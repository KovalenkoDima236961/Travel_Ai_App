/**
 * Renders a trip's date span like the mock ("Sep 14 – 18, 2026 · 4 days"),
 * collapsing the month/year when the range stays within one. Falls back to just
 * the duration when there is no start date.
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
