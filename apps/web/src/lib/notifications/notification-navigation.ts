import type { AppNotification } from "@/entities/notification/model";
import {
  buildTripWorkspaceHref,
  normalizeTripWorkspaceHref,
  type TripWorkspaceSectionId
} from "@/lib/trip-workspace/navigation";

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
    return normalizeTripWorkspaceHref(metadataURL);
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
    const destination = notificationDestination(notification);
    const target = entityTarget(notification, destination.section, destination.view);
    return buildTripWorkspaceHref(notification.tripId, destination.section, {
      view: destination.view,
      ...target
    });
  }
  return "/trips";
}

function notificationDestination(notification: AppNotification): {
  section: TripWorkspaceSectionId;
  view: string;
} {
  const type = notification.type;
  if (
    type === "comment_created" ||
    type === "collaboration_accepted" ||
    type === "collaborator_role_changed" ||
    type === "collaborator_removed"
  ) {
    return { section: "group", view: type === "comment_created" ? "discussion" : "people" };
  }
  if (
    type === "trip_poll_created" ||
    type === "trip_poll_closed" ||
    type === "poll_vote_nudge" ||
    type === "trip_submitted_for_approval" ||
    type === "trip_approved" ||
    type === "trip_changes_requested" ||
    type === "trip_approval_cancelled" ||
    type === "trip_approval_reset_to_draft"
  ) {
    return { section: "group", view: type.includes("approval") || type.includes("approved") || type.includes("changes_requested") ? "approvals" : "decisions" };
  }
  if (type === "availability_nudge" || type === "availability_requested" || type === "date_option_applied") {
    return { section: "group", view: "availability" };
  }
  if (type === "group_readiness_nudge") {
    return { section: "group", view: "activity" };
  }
  if (
    type === "expense_added" ||
    type === "settlement_paid" ||
    type === "settlement_nudge" ||
    type === "settlement_pending" ||
    type === "settlement_overdue" ||
    type === "budget_confidence_changed" ||
    type === "budget_optimization_ready" ||
    type === "budget_optimization_failed"
  ) {
    return {
      section: "money",
      view: type.startsWith("settlement") ? "settlements" : type.startsWith("budget") ? "budget" : "expenses"
    };
  }
  if (
    type === "checklist_assignment_nudge" ||
    type === "checklist_item_assigned" ||
    type === "checklist_item_completed" ||
    type === "checklist_item_overdue" ||
    type === "checklist_generated"
  ) {
    return { section: "prepare", view: "checklist" };
  }
  if (type === "reminder_task_nudge" || type === "reminder_assigned" || type === "pre_trip_reminder_due") {
    return { section: "prepare", view: "reminders" };
  }
  if (type === "offline_sync_conflict") {
    return { section: "prepare", view: "offline" };
  }
  if (type === "version_restored") {
    return { section: "more", view: "versions" };
  }
  if (type === "trip_health_issue" || type === "generation_job_failed") {
    return { section: "more", view: "health" };
  }
  if (type === "share_security_changed") {
    return { section: "more", view: "sharing" };
  }
  if (type === "calendar_sync_failed") {
    return { section: "more", view: "tools" };
  }
  return { section: "plan", view: "itinerary" };
}

function entityTarget(
  notification: AppNotification,
  section: TripWorkspaceSectionId,
  view: string
): Record<string, string> {
  if (!notification.entityId) {
    return {};
  }
  const entityType = notification.entityType?.toLowerCase() ?? "";
  if (entityType.includes("comment")) return { comment: notification.entityId };
  if (entityType.includes("expense")) return { expense: notification.entityId };
  if (entityType.includes("receipt")) return { receipt: notification.entityId };
  if (entityType.includes("checklist")) return { item: notification.entityId };
  if (entityType.includes("reminder")) return { reminder: notification.entityId };
  if (entityType.includes("poll") || entityType.includes("decision")) return { decision: notification.entityId };
  if (entityType.includes("version")) return { version: notification.entityId };
  if (entityType.includes("health")) return { issue: notification.entityId };
  if (section === "group" && view === "people") return { member: notification.entityId };
  return {};
}

function metadataString(metadata: Record<string, unknown>, key: string) {
  const value = metadata[key];
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : null;
}
