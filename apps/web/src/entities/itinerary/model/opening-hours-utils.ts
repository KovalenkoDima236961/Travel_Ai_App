import type { OpeningHoursInterval } from "@/entities/place/model";

export type OpeningStatus =
  | {
      status: "unknown";
      label: string;
    }
  | {
      status: "open";
      label: string;
      matchingInterval: OpeningHoursInterval;
    }
  | {
      status: "closed";
      label: string;
      intervalsForDay: OpeningHoursInterval[];
    };

const dayNames = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
const timeSeparator = "\u2013";

export function getTripItemDate(startDate: string, dayNumber: number): Date | null {
  if (!Number.isInteger(dayNumber) || dayNumber < 1) {
    return null;
  }

  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(startDate.trim());
  if (!match) {
    return null;
  }

  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const date = new Date(year, month - 1, day);
  if (
    date.getFullYear() !== year ||
    date.getMonth() !== month - 1 ||
    date.getDate() !== day
  ) {
    return null;
  }

  date.setDate(date.getDate() + dayNumber - 1);
  return date;
}

export function getDayOfWeekMondayBased(date: Date): number {
  const jsDay = date.getDay();
  return jsDay === 0 ? 7 : jsDay;
}

export function parseTimeToMinutes(value: string): number | null {
  const match = /(^|[^0-9])([01][0-9]|2[0-3]):([0-5][0-9])(?![0-9])/.exec(value);
  if (!match) {
    return null;
  }
  return Number(match[2]) * 60 + Number(match[3]);
}

export function isTimeWithinInterval(time: string, interval: OpeningHoursInterval): boolean {
  const timeMinutes = parseTimeToMinutes(time);
  const openMinutes = parseTimeToMinutes(interval.open);
  const closeMinutes = parseTimeToMinutes(interval.close);
  if (timeMinutes == null || openMinutes == null || closeMinutes == null) {
    return false;
  }
  if (openMinutes >= closeMinutes) {
    return false;
  }
  return timeMinutes >= openMinutes && timeMinutes < closeMinutes;
}

export function getOpeningStatus(params: {
  startDate?: string | null;
  dayNumber: number;
  itemTime?: string | null;
  openingHours?: OpeningHoursInterval[] | null;
}): OpeningStatus {
  if (!params.startDate) {
    return { status: "unknown", label: "Opening hours need a trip start date" };
  }

  if (!params.openingHours || params.openingHours.length === 0) {
    return { status: "unknown", label: "Opening hours unknown" };
  }

  const itemDate = getTripItemDate(params.startDate, params.dayNumber);
  if (!itemDate) {
    return { status: "unknown", label: "Opening hours need a trip start date" };
  }

  if (!params.itemTime || parseTimeToMinutes(params.itemTime) == null) {
    return { status: "unknown", label: "Opening status unknown for this time" };
  }

  const dayOfWeek = getDayOfWeekMondayBased(itemDate);
  const intervalsForDay = params.openingHours.filter(
    (interval) => interval.dayOfWeek === dayOfWeek
  );
  if (intervalsForDay.length === 0) {
    return { status: "closed", label: "May be closed on this day", intervalsForDay };
  }

  const matchingInterval = intervalsForDay.find((interval) =>
    isTimeWithinInterval(params.itemTime ?? "", interval)
  );
  if (matchingInterval) {
    return { status: "open", label: "Likely open at this time", matchingInterval };
  }

  return { status: "closed", label: "May be closed at this time", intervalsForDay };
}

export function formatOpeningHoursForDay(
  openingHours: OpeningHoursInterval[] | null | undefined,
  dayOfWeek: number
): string {
  const intervals = (openingHours ?? []).filter((interval) => interval.dayOfWeek === dayOfWeek);
  if (intervals.length === 0) {
    return "Closed or unknown";
  }
  return intervals.map((interval) => `${interval.open}${timeSeparator}${interval.close}`).join(", ");
}

export function formatDayOfWeek(dayOfWeek: number): string {
  return dayNames[dayOfWeek - 1] ?? "Unknown day";
}
