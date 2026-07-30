import { describe, expect, it } from "vitest";
import type { Itinerary } from "@/entities/trip/model";
import {
  applyItemSchedule,
  detectScheduleConflicts,
  formatMinutesAsTime,
  moveScheduledItem,
  parseTimeToMinutes,
  unscheduledItemsForItinerary
} from "@/features/timeline-planning";

describe("timeline planning schedule helpers", () => {
  it("parses and formats HH:mm times", () => {
    expect(parseTimeToMinutes("09:30")).toBe(570);
    expect(parseTimeToMinutes("24:00")).toBeNull();
    expect(formatMinutesAsTime(570)).toBe("09:30");
  });

  it("detects overlapping scheduled activities", () => {
    const conflicts = detectScheduleConflicts({
      days: [{
        day: 1,
        title: "Day 1",
        items: [
          { time: "09:00", type: "activity", name: "Museum", durationMinutes: 90 },
          { time: "10:00", type: "activity", name: "Market", durationMinutes: 60 }
        ]
      }]
    });

    expect(conflicts.some((conflict) => conflict.id.startsWith("overlap"))).toBe(true);
  });

  it("keeps unscheduled activities inside the itinerary", () => {
    const itinerary: Itinerary = {
      days: [{
        day: 1,
        title: "Day 1",
        items: [
          { time: "", type: "activity", name: "Optional gallery", schedulingStatus: "Unscheduled" },
          { time: "09:00", type: "food", name: "Breakfast" }
        ]
      }]
    };

    expect(unscheduledItemsForItinerary(itinerary)).toHaveLength(1);
  });

  it("moves an activity between days by updating the same itinerary model", () => {
    const itinerary: Itinerary = {
      days: [
        {
          day: 1,
          title: "Day 1",
          items: [
            { time: "09:00", type: "food", name: "Breakfast" },
            { time: "10:00", type: "activity", name: "Museum", durationMinutes: 60 }
          ]
        },
        { day: 2, title: "Day 2", items: [{ time: "11:00", type: "activity", name: "Market" }] }
      ]
    };

    const moved = moveScheduledItem(itinerary, 0, 1, 1, "14:00");

    expect(moved.days[0].items.map((item) => item.name)).toEqual(["Breakfast"]);
    expect(moved.days[1].items[1]).toMatchObject({
      name: "Museum",
      time: "14:00",
      startTime: "14:00",
      schedulingStatus: "Scheduled"
    });
  });

  it("preserves duration when only start time changes", () => {
    const updated = applyItemSchedule(
      { time: "09:00", startTime: "09:00", type: "activity", name: "Museum", durationMinutes: 90 },
      { startTime: "10:00" }
    );

    expect(updated.durationMinutes).toBe(90);
    expect(updated.endTime).toBe("11:30");
  });
});
