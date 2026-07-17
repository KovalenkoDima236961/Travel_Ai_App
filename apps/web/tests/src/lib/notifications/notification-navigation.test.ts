import { describe, expect, it } from "vitest";
import { getNotificationHref } from "@/lib/notifications/notification-navigation";
import type { AppNotification } from "@/entities/notification/model";

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
    priority: "normal",
    category: "comments",
    groupedCount: 1,
    latestEventAt: "2026-06-24T12:00:00Z",
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

  it("routes workspace invitations to the invitations page", () => {
    expect(getNotificationHref(notification({ type: "workspace_invited" }))).toBe(
      "/workspace-invitations"
    );
  });

  it("routes workspace entity notifications to the workspace page", () => {
    expect(
      getNotificationHref(
        notification({
          type: "workspace_role_changed",
          entityType: "workspace",
          entityId: "workspace-1"
        })
      )
    ).toBe("/workspaces/workspace-1");
  });

  it("honors safe app-relative notification urls", () => {
    expect(
      getNotificationHref(
        notification({
          type: "workspace_invitation_accepted",
          metadata: { url: "/workspaces/workspace-2" }
        })
      )
    ).toBe("/workspaces/workspace-2");
  });

  it("falls back to the trips page when there is no tripId", () => {
    expect(getNotificationHref(notification({ type: "version_restored", tripId: null }))).toBe(
      "/trips"
    );
  });
});
