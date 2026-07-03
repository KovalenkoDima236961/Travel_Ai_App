// Notification types mirror the Notification Service API. Notifications are
// private, authenticated, per-user data and are never fetched from the public
// share page.

export type NotificationType =
  | "collaboration_invited"
  | "collaboration_accepted"
  | "collaborator_role_changed"
  | "collaborator_removed"
  | "comment_created"
  | "itinerary_updated"
  | "itinerary_generated"
  | "day_regenerated"
  | "item_regenerated"
  | "version_restored"
  | "generation_job_failed"
  | "budget_optimization_ready"
  | "budget_optimization_failed";

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
