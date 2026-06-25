import { describe, expect, it } from "vitest";
import { getNotificationHref } from "@/lib/notifications/notification-navigation";
import type { AppNotification } from "@/types/notifications";

function notification(overrides: Partial<AppNotification>): AppNotification {
  return {
    id: "n1",
    userId: "u1",
    tripId: null,
    actorUserId: null,
    type: "comment_created",
    title: "New comment",
    message: "A collaborator commented.",
    entityType: null,
    entityId: null,
    metadata: {},
    readAt: null,
    createdAt: "2026-06-24T12:00:00Z",
    ...overrides
  };
}

describe("getNotificationHref", () => {
  it("routes collaboration invites to the trips page", () => {
    expect(
      getNotificationHref(notification({ type: "collaboration_invited", tripId: "trip-1" }))
    ).toBe("/trips");
  });

  it("routes comment notifications with a tripId to the trip detail page", () => {
    expect(
      getNotificationHref(notification({ type: "comment_created", tripId: "trip-1" }))
    ).toBe("/trips/trip-1");
  });

  it("routes itinerary updates with a tripId to the trip detail page", () => {
    expect(
      getNotificationHref(notification({ type: "itinerary_updated", tripId: "trip-9" }))
    ).toBe("/trips/trip-9");
  });

  it("falls back to the trips page when there is no tripId", () => {
    expect(getNotificationHref(notification({ type: "version_restored", tripId: null }))).toBe(
      "/trips"
    );
  });
});
