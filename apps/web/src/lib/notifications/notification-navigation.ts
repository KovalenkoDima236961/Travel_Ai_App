import type { AppNotification } from "@/types/notifications";

/**
 * Resolves the in-app destination for a notification. Clicking a notification
 * navigates here.
 *
 * Rules:
 *  - collaboration_invited -> /trips (invitations are managed on the trips page)
 *  - any notification with a tripId -> /trips/{tripId}
 *  - fallback -> /trips
 */
export function getNotificationHref(notification: AppNotification): string {
  if (notification.type === "collaboration_invited") {
    return "/trips";
  }
  if (notification.tripId) {
    return `/trips/${notification.tripId}`;
  }
  return "/trips";
}
