import { describe, expect, it } from "vitest";
import {
  formatDayOfWeek,
  formatOpeningHoursForDay,
  getDayOfWeekMondayBased,
  getOpeningStatus,
  getTripItemDate,
  isTimeWithinInterval,
  parseTimeToMinutes
} from "@/lib/itinerary/opening-hours-utils";
import type { OpeningHoursInterval } from "@/types/place";

const mondayToFriday: OpeningHoursInterval[] = [
  { dayOfWeek: 1, open: "09:00", close: "12:00" },
  { dayOfWeek: 1, open: "14:00", close: "18:00" },
  { dayOfWeek: 2, open: "09:00", close: "18:00" },
  { dayOfWeek: 3, open: "09:00", close: "18:00" },
  { dayOfWeek: 4, open: "09:00", close: "18:00" },
  { dayOfWeek: 5, open: "09:00", close: "18:00" }
];

describe("opening hours utilities", () => {
  it("calculates a trip item date from a local YYYY-MM-DD start date and day number", () => {
    const date = getTripItemDate("2026-08-10", 3);

    expect(date?.getFullYear()).toBe(2026);
    expect(date?.getMonth()).toBe(7);
    expect(date?.getDate()).toBe(12);
  });

  it("converts JavaScript dates to Monday-based day numbers", () => {
    expect(getDayOfWeekMondayBased(new Date(2026, 7, 10))).toBe(1);
    expect(getDayOfWeekMondayBased(new Date(2026, 7, 9))).toBe(7);
    expect(formatDayOfWeek(7)).toBe("Sunday");
  });

  it("parses strict HH:mm values and the first HH:mm occurrence in approximate text", () => {
    expect(parseTimeToMinutes("09:00")).toBe(540);
    expect(parseTimeToMinutes("Meet around 14:30 near the entrance")).toBe(870);
    expect(parseTimeToMinutes("9:00")).toBeNull();
    expect(parseTimeToMinutes("24:00")).toBeNull();
  });

  it("detects whether an item time is inside an opening interval", () => {
    expect(
      isTimeWithinInterval("09:30", { dayOfWeek: 1, open: "09:00", close: "10:00" })
    ).toBe(true);
    expect(
      isTimeWithinInterval("10:00", { dayOfWeek: 1, open: "09:00", close: "10:00" })
    ).toBe(false);
  });

  it("returns open when the item time is inside one of multiple intervals", () => {
    const status = getOpeningStatus({
      startDate: "2026-08-10",
      dayNumber: 1,
      itemTime: "14:30",
      openingHours: mondayToFriday
    });

    expect(status.status).toBe("open");
    expect(status.label).toBe("Likely open at this time");
  });

  it("returns closed when the item time is outside all intervals for the day", () => {
    const status = getOpeningStatus({
      startDate: "2026-08-10",
      dayNumber: 1,
      itemTime: "12:30",
      openingHours: mondayToFriday
    });

    expect(status.status).toBe("closed");
    expect(status.label).toBe("May be closed at this time");
  });

  it("returns closed when there is no interval for the trip day", () => {
    const status = getOpeningStatus({
      startDate: "2026-08-10",
      dayNumber: 7,
      itemTime: "10:00",
      openingHours: mondayToFriday
    });

    expect(status.status).toBe("closed");
    expect(status.label).toBe("May be closed on this day");
  });

  it("returns unknown without opening hours or valid item time", () => {
    expect(
      getOpeningStatus({
        startDate: "2026-08-10",
        dayNumber: 1,
        itemTime: "09:00",
        openingHours: null
      }).status
    ).toBe("unknown");
    expect(
      getOpeningStatus({
        startDate: "2026-08-10",
        dayNumber: 1,
        itemTime: "morning",
        openingHours: mondayToFriday
      }).label
    ).toBe("Opening status unknown for this time");
  });

  it("formats intervals for a single day", () => {
    expect(formatOpeningHoursForDay(mondayToFriday, 1)).toBe("09:00\u201312:00, 14:00\u201318:00");
    expect(formatOpeningHoursForDay(mondayToFriday, 7)).toBe("Closed or unknown");
  });
});
