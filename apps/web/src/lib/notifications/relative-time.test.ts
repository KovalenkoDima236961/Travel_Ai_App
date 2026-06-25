import { describe, expect, it } from "vitest";
import { formatRelativeTime } from "@/lib/notifications/relative-time";

describe("formatRelativeTime", () => {
  const now = new Date("2026-06-24T12:00:00Z");

  it("renders 'just now' for very recent timestamps", () => {
    expect(formatRelativeTime("2026-06-24T11:59:30Z", now)).toBe("just now");
  });

  it("renders minutes ago", () => {
    expect(formatRelativeTime("2026-06-24T11:45:00Z", now)).toBe("15m ago");
  });

  it("renders hours ago", () => {
    expect(formatRelativeTime("2026-06-24T09:00:00Z", now)).toBe("3h ago");
  });

  it("renders days ago", () => {
    expect(formatRelativeTime("2026-06-22T12:00:00Z", now)).toBe("2d ago");
  });

  it("returns empty string for an invalid date", () => {
    expect(formatRelativeTime("not-a-date", now)).toBe("");
  });

  it("treats future timestamps as 'just now'", () => {
    expect(formatRelativeTime("2026-06-24T12:05:00Z", now)).toBe("just now");
  });
});
