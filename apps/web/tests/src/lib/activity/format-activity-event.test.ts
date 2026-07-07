import { describe, expect, it } from "vitest";

import { formatActivityEvent } from "@/entities/activity/model";
import type { TripActivityEvent } from "@/entities/activity/model";

const CURRENT_USER = "11111111-1111-1111-1111-111111111111";
const OTHER_USER = "22222222-2222-2222-2222-222222222222";

function makeEvent(overrides: Partial<TripActivityEvent> = {}): TripActivityEvent {
  return {
    id: "event-1",
    tripId: "trip-1",
    actorUserId: CURRENT_USER,
    eventType: "trip_created",
    entityType: "trip",
    entityId: "trip-1",
    metadata: {},
    createdAt: "2026-06-24T10:00:00.000Z",
    ...overrides
  };
}

describe("formatActivityEvent actor labels", () => {
  it("shows You for the current user", () => {
    const result = formatActivityEvent(makeEvent({ actorUserId: CURRENT_USER }), CURRENT_USER);
    expect(result.actorLabel).toBe("You");
    expect(result.title).toBe("You created the trip");
  });

  it("shows Collaborator for another user", () => {
    const result = formatActivityEvent(makeEvent({ actorUserId: OTHER_USER }), CURRENT_USER);
    expect(result.actorLabel).toBe("Collaborator");
    expect(result.title).toBe("Collaborator created the trip");
  });

  it("shows System for a null actor", () => {
    const result = formatActivityEvent(makeEvent({ actorUserId: null }), CURRENT_USER);
    expect(result.actorLabel).toBe("System");
    expect(result.title).toBe("System created the trip");
  });
});

describe("formatActivityEvent titles", () => {
  const cases: Array<{ event: Partial<TripActivityEvent>; expected: string }> = [
    { event: { eventType: "trip_created" }, expected: "You created the trip" },
    { event: { eventType: "itinerary_generated" }, expected: "You generated the itinerary" },
    { event: { eventType: "itinerary_updated" }, expected: "You updated the itinerary" },
    {
      event: { eventType: "day_regenerated", metadata: { dayNumber: 2 } },
      expected: "You regenerated Day 2"
    },
    {
      event: {
        eventType: "item_regenerated",
        metadata: { dayNumber: 2, itemIndex: 3, itemName: "Louvre Museum" }
      },
      expected: "You regenerated Day 2 item: Louvre Museum"
    },
    { event: { eventType: "version_restored" }, expected: "You restored an itinerary version" },
    {
      event: {
        eventType: "comment_created",
        metadata: { dayNumber: 2, itemIndex: 3, itemName: "Louvre Museum" }
      },
      expected: "You commented on Day 2 · Louvre Museum"
    },
    {
      event: {
        eventType: "comment_updated",
        metadata: { dayNumber: 2, itemName: "Louvre Museum" }
      },
      expected: "You edited a comment on Day 2 · Louvre Museum"
    },
    {
      event: {
        eventType: "comment_deleted",
        metadata: { dayNumber: 2, itemName: "Louvre Museum" }
      },
      expected: "You deleted a comment on Day 2 · Louvre Museum"
    },
    {
      event: {
        eventType: "collaborator_invited",
        metadata: { collaboratorEmail: "anna@example.com", role: "editor" }
      },
      expected: "You invited anna@example.com as editor"
    },
    { event: { eventType: "collaborator_accepted" }, expected: "You accepted the invitation" },
    { event: { eventType: "collaborator_declined" }, expected: "You declined the invitation" },
    {
      event: {
        eventType: "collaborator_role_changed",
        metadata: { oldRole: "viewer", newRole: "editor" }
      },
      expected: "You changed a collaborator from viewer to editor"
    },
    { event: { eventType: "collaborator_removed" }, expected: "You removed a collaborator" },
    { event: { eventType: "share_created" }, expected: "You created a share link" },
    { event: { eventType: "share_updated" }, expected: "You updated share settings" },
    { event: { eventType: "share_disabled" }, expected: "You disabled the share link" },
    {
      event: { eventType: "accommodation_added", metadata: { name: "Hotel Roma" } },
      expected: "You added Hotel Roma"
    },
    {
      event: { eventType: "accommodation_updated", metadata: { name: "Hotel Roma" } },
      expected: "You updated Hotel Roma"
    },
    {
      event: { eventType: "accommodation_removed", metadata: { name: "Hotel Roma" } },
      expected: "You removed Hotel Roma"
    }
  ];

  for (const { event, expected } of cases) {
    it(`formats ${event.eventType}`, () => {
      const result = formatActivityEvent(makeEvent(event), CURRENT_USER);
      expect(result.title).toBe(expected);
    });
  }
});

describe("formatActivityEvent resilience", () => {
  it("degrades gracefully when metadata fields are missing", () => {
    expect(
      formatActivityEvent(makeEvent({ eventType: "day_regenerated", metadata: {} }), CURRENT_USER)
        .title
    ).toBe("You regenerated a day");
    expect(
      formatActivityEvent(makeEvent({ eventType: "comment_created", metadata: {} }), CURRENT_USER)
        .title
    ).toBe("You commented on an item");
    expect(
      formatActivityEvent(
        makeEvent({ eventType: "collaborator_invited", metadata: {} }),
        CURRENT_USER
      ).title
    ).toBe("You invited a collaborator");
    expect(
      formatActivityEvent(
        makeEvent({ eventType: "collaborator_role_changed", metadata: {} }),
        CURRENT_USER
      ).title
    ).toBe("You changed a collaborator's role");
  });

  it("exposes day/item numbers from metadata", () => {
    const result = formatActivityEvent(
      makeEvent({ eventType: "comment_created", metadata: { dayNumber: 4, itemIndex: 1 } }),
      CURRENT_USER
    );
    expect(result.dayNumber).toBe(4);
    expect(result.itemIndex).toBe(1);
  });

  it("does not crash on an unknown event type", () => {
    const result = formatActivityEvent(
      makeEvent({ eventType: "something_new" as TripActivityEvent["eventType"] }),
      CURRENT_USER
    );
    expect(result.title).toBe("Activity recorded");
  });
});
