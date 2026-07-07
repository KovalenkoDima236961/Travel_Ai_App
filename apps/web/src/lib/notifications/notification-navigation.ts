import type { AppNotification } from "@/entities/notification/model";

/**
 * Resolves the in-app destination for a notification. Clicking a notification
 * navigates here.
 *
 * Rules:
 *  - collaboration_invited -> /trips (invitations are managed on the trips page)
 *  - workspace_invited -> /workspace-invitations
 *  - workspace entity notifications -> /workspaces/{workspaceId}
 *  - any notification with a tripId -> /trips/{tripId}
 *  - fallback -> /trips
 */
export function getNotificationHref(notification: AppNotification): string {
  const metadataURL = metadataString(notification.metadata, "url");
  if (metadataURL?.startsWith("/")) {
    return metadataURL;
  }

  if (notification.type === "workspace_invited") {
    return "/workspace-invitations";
  }
  if (notification.entityType === "workspace" && notification.entityId) {
    return `/workspaces/${notification.entityId}`;
  }
  const workspaceId = metadataString(notification.metadata, "workspaceId");
  if (workspaceId) {
    return `/workspaces/${workspaceId}`;
  }
  if (notification.type === "collaboration_invited") {
    return "/trips";
  }
  if (notification.tripId) {
    return `/trips/${notification.tripId}`;
  }
  return "/trips";
}

function metadataString(metadata: Record<string, unknown>, key: string) {
  const value = metadata[key];
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : null;
}
