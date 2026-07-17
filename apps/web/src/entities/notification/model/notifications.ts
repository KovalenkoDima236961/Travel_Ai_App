// Notification types mirror the Notification Service API. Notifications are
// private, authenticated, per-user data and are never fetched from the public
// share page.

export type NotificationType =
  | "collaboration_invited"
  | "collaboration_accepted"
  | "collaborator_role_changed"
  | "collaborator_removed"
  | "comment_created"
  | "trip_poll_created"
  | "trip_poll_closed"
  | "itinerary_updated"
  | "itinerary_generated"
  | "day_regenerated"
  | "item_regenerated"
  | "version_restored"
  | "generation_job_failed"
  | "budget_optimization_ready"
  | "budget_optimization_failed"
  | "expense_added"
  | "settlement_paid"
  | "group_readiness_nudge"
  | "availability_nudge"
  | "checklist_assignment_nudge"
  | "reminder_task_nudge"
  | "poll_vote_nudge"
  | "settlement_nudge"
  | "workspace_budget_created"
  | "workspace_budget_updated"
  | "workspace_budget_archived"
  | "workspace_budget_exceeded"
  | "workspace_budget_nearing_limit"
  | "workspace_invited"
  | "workspace_invitation_accepted"
  | "workspace_invitation_declined"
  | "workspace_member_removed"
  | "workspace_role_changed"
  | "workspace_trip_created";

export type ExtendedNotificationType =
  | NotificationType
  | "date_option_applied"
  | "availability_requested"
  | "pre_trip_reminder_due"
  | "reminder_assigned"
  | "trip_submitted_for_approval"
  | "trip_approved"
  | "trip_changes_requested"
  | "trip_approval_cancelled"
  | "trip_approval_reset_to_draft"
  | "route_changed"
  | "checklist_item_assigned"
  | "checklist_item_completed"
  | "checklist_item_overdue"
  | "checklist_generated"
  | "settlement_pending"
  | "settlement_overdue"
  | "budget_confidence_changed"
  | "trip_health_issue"
  | "offline_sync_conflict"
  | "calendar_sync_failed"
  | "share_security_changed"
  | "notification_digest";

export type AppNotification = {
  id: string;
  userId: string;
  tripId?: string | null;
  actorUserId?: string | null;
  type: ExtendedNotificationType;
  title: string;
  message: string;
  entityType?: string | null;
  entityId?: string | null;
  metadata: Record<string, unknown>;
  readAt?: string | null;
  createdAt: string;
  priority: "low" | "normal" | "high" | "urgent";
  category: string;
  digestKey?: string | null;
  dedupeKey?: string | null;
  groupedCount: number;
  digestBatchId?: string | null;
  deliveryMode?: string | null;
  deliveryStatus?: string | null;
  expiresAt?: string | null;
  latestEventAt: string;
};

export type NotificationDigestItem = {
  id: string;
  tripId?: string | null;
  category: string;
  priority: string;
  digestKey: string;
  title: string;
  message: string;
  metadata: Record<string, unknown>;
  eventCount: number;
  latestEventAt: string;
};

export type NotificationDigest = {
  id: string;
  channel: "in_app" | "email" | "push";
  mode: string;
  status: string;
  scheduledFor: string;
  sentAt?: string | null;
  attempts: number;
  nextAttemptAt?: string | null;
  errorCode?: string | null;
  errorMessageSafe?: string | null;
  eventCount: number;
  items: NotificationDigestItem[];
};

export type NotificationDigestsResponse = { items: NotificationDigest[] };

export type NotificationsResponse = {
  items: AppNotification[];
  nextCursor?: string | null;
};

export type UnreadNotificationsResponse = {
  count: number;
};

export type MarkNotificationResponse = {
  success: boolean;
};

export type NotificationCreatedStreamPayload = {
  notification: AppNotification;
};
