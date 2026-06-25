// Small dependency-free relative-time formatter for notification timestamps
// ("just now", "3m ago", "2h ago", "5d ago", or a date for older entries).

const MINUTE = 60;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;

export function formatRelativeTime(iso: string, now: Date = new Date()): string {
  const then = new Date(iso);
  const timestamp = then.getTime();
  if (Number.isNaN(timestamp)) {
    return "";
  }

  const diffSeconds = Math.round((now.getTime() - timestamp) / 1000);
  if (diffSeconds < 0) {
    return "just now";
  }
  if (diffSeconds < MINUTE) {
    return "just now";
  }
  if (diffSeconds < HOUR) {
    return `${Math.floor(diffSeconds / MINUTE)}m ago`;
  }
  if (diffSeconds < DAY) {
    return `${Math.floor(diffSeconds / HOUR)}h ago`;
  }
  if (diffSeconds < WEEK) {
    return `${Math.floor(diffSeconds / DAY)}d ago`;
  }

  return then.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric"
  });
}
