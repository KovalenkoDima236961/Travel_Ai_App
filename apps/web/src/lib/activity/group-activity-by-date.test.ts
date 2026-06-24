import { describe, expect, it } from "vitest";

import { groupActivityByDate } from "@/lib/activity/group-activity-by-date";
import type { TripActivityEvent } from "@/types/activity";

function makeEvent(id: string, createdAt: string): TripActivityEvent {
  return {
    id,
    tripId: "trip-1",
    actorUserId: "user-1",
    eventType: "comment_created",
    entityType: "comment",
    entityId: id,
    metadata: {},
    createdAt
  };
}

describe("groupActivityByDate", () => {
  // Local-time construction so day grouping is deterministic regardless of the
  // test runner's timezone.
  const now = new Date(2026, 5, 24, 12, 0, 0);
  const localISO = (y: number, mo: number, d: number, h: number) =>
    new Date(y, mo, d, h, 0, 0).toISOString();

  it("labels today, yesterday, and older dates", () => {
    const events = [
      makeEvent("a", localISO(2026, 5, 24, 9)),
      makeEvent("b", localISO(2026, 5, 23, 20)),
      makeEvent("c", localISO(2026, 5, 20, 8))
    ];
    const groups = groupActivityByDate(events, now);

    expect(groups.map((group) => group.label)).toEqual(["Today", "Yesterday", "Jun 20, 2026"]);
    expect(groups[0].events.map((event) => event.id)).toEqual(["a"]);
  });

  it("keeps multiple events on the same day in their original order", () => {
    const events = [
      makeEvent("a", localISO(2026, 5, 24, 11)),
      makeEvent("b", localISO(2026, 5, 24, 10))
    ];
    const groups = groupActivityByDate(events, now);

    expect(groups).toHaveLength(1);
    expect(groups[0].label).toBe("Today");
    expect(groups[0].events.map((event) => event.id)).toEqual(["a", "b"]);
  });

  it("collects events with an invalid timestamp under an Earlier bucket", () => {
    const events = [makeEvent("a", "2026-06-24T11:00:00.000Z"), makeEvent("bad", "not-a-date")];
    const groups = groupActivityByDate(events, now);

    const earlier = groups.find((group) => group.label === "Earlier");
    expect(earlier).toBeDefined();
    expect(earlier?.events.map((event) => event.id)).toEqual(["bad"]);
  });

  it("returns no groups for an empty list", () => {
    expect(groupActivityByDate([], now)).toEqual([]);
  });
});
