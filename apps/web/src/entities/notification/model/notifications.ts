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

export type AppNotification = {
  id: string;
  userId: string;
  tripId?: string | null;
  actorUserId?: string | null;
  type: NotificationType;
  title: string;
  message: string;
  entityType?: string | null;
  entityId?: string | null;
  metadata: Record<string, unknown>;
  readAt?: string | null;
  createdAt: string;
};

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
