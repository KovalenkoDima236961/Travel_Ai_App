import { describe, expect, it } from "vitest";
import { dateBucket, groupNotifications } from "../src/components/notifications/NotificationCenter";
import { digestPreviewLines } from "../src/components/notifications/NotificationDigestCard";
import type { AppNotification, NotificationDigest } from "../src/entities/notification/model";

function notification(overrides: Partial<AppNotification>): AppNotification {
  return {
    id: crypto.randomUUID(), userId: crypto.randomUUID(), type: "comment_created",
    title: "New comment", message: "A collaborator commented.", metadata: { tripName: "Austria" },
    createdAt: "2026-07-17T10:00:00Z", latestEventAt: "2026-07-17T10:00:00Z",
    priority: "normal", category: "comments", groupedCount: 1, ...overrides,
  };
}

describe("notification noise-control UI models", () => {
  it("groups notifications by date, trip, and category", () => {
    const now = new Date("2026-07-17T12:00:00Z");
    const groups = groupNotifications([
      notification({ id: "one", tripId: "11111111-1111-1111-1111-111111111111" }),
      notification({ id: "two", tripId: "11111111-1111-1111-1111-111111111111" }),
      notification({ id: "three", category: "expenses", createdAt: "2026-07-16T12:00:00Z" }),
    ], now);
    expect(groups.map((group) => group.label)).toEqual(["Today", "Yesterday"]);
    expect(groups[0]?.groups[0]?.items).toHaveLength(2);
    expect(groups[1]?.groups[0]?.title).toContain("expenses");
  });

  it("uses stable date buckets", () => {
    const now = new Date("2026-07-17T12:00:00Z");
    expect(dateBucket("2026-07-17T01:00:00Z", now)).toBe("Today");
    expect(dateBucket("2026-07-16T01:00:00Z", now)).toBe("Yesterday");
    expect(dateBucket("2026-07-13T01:00:00Z", now)).toBe("This week");
    expect(dateBucket("2026-06-01T01:00:00Z", now)).toBe("Older");
  });

  it("builds an expandable digest preview with grouped counts", () => {
    const digest: NotificationDigest = {
      id: "digest", channel: "email", mode: "daily_digest", status: "pending",
      scheduledFor: "2026-07-18T08:00:00Z", attempts: 0, eventCount: 3,
      items: [{ id: "item", category: "comments", priority: "normal", digestKey: "trip:x:comments", title: "Comments", message: "A collaborator commented.", metadata: {}, eventCount: 3, latestEventAt: "2026-07-17T10:00:00Z" }],
    };
    expect(digestPreviewLines(digest)).toEqual([{ id: "item", category: "comments", message: "A collaborator commented. (3 updates)" }]);
  });
});
