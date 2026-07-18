import type { AppNotification } from "@/entities/notification/model";
import { TEST_TRIP_ID } from "./trips";
import { TEST_USER_IDS } from "./users";

export const notificationsFixture: AppNotification[] = [
  {
    id: "notification-1",
    userId: TEST_USER_IDS.owner,
    tripId: TEST_TRIP_ID,
    actorUserId: TEST_USER_IDS.editor,
    type: "itinerary_updated",
    title: "Itinerary updated",
    message: "The Vienna itinerary has a new revision.",
    entityType: "trip",
    entityId: TEST_TRIP_ID,
    metadata: { itineraryRevision: 3 },
    readAt: null,
    createdAt: "2026-02-01T10:10:00Z",
    priority: "normal",
    category: "trip_updates",
    groupedCount: 1,
    latestEventAt: "2026-02-01T10:10:00Z"
  },
  {
    id: "notification-2",
    userId: TEST_USER_IDS.owner,
    tripId: TEST_TRIP_ID,
    actorUserId: null,
    type: "itinerary_generated",
    title: "Itinerary ready",
    message: "Your deterministic mock itinerary is ready.",
    metadata: {},
    readAt: "2026-02-01T10:05:00Z",
    createdAt: "2026-02-01T10:00:00Z",
    priority: "normal",
    category: "trip_updates",
    groupedCount: 1,
    latestEventAt: "2026-02-01T10:00:00Z"
  }
];
