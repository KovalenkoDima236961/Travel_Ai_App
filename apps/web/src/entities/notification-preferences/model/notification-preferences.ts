export type NotificationChannel = "in_app" | "email" | "push";

export type NotificationCategory =
  | "collaboration"
  | "comments"
  | "trip_updates"
  | "role_changes"
  | "pre_trip_reminders"
  | "checklist_reminders";

export type NotificationPreference = {
  channel: NotificationChannel;
  category: NotificationCategory;
  enabled: boolean;
};

export type NotificationPreferencesResponse = {
  items: NotificationPreference[];
};
