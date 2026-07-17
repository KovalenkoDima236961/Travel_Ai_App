export type NotificationChannel = "in_app" | "email" | "push";
export type NotificationDeliveryMode =
  | "instant"
  | "hourly_digest"
  | "daily_digest"
  | "weekly_digest"
  | "muted";

export type NotificationCategory =
  | "collaboration" | "comments" | "role_changes" | "trip_updates"
  | "checklist" | "checklist_reminders" | "reminders" | "pre_trip_reminders"
  | "expenses" | "settlements" | "approval" | "budget" | "health"
  | "offline_sync" | "calendar" | "ai_generation" | "security" | "system";

export type NotificationPreference = {
  channel: NotificationChannel;
  category: NotificationCategory;
  enabled: boolean;
  deliveryMode: NotificationDeliveryMode;
};

export type NotificationSettings = {
  quietHoursEnabled: boolean;
  quietHoursStart: string;
  quietHoursEnd: string;
  quietHoursTimezone: string;
  urgentBypassesQuietHours: boolean;
  dailyDigestTime: string;
  weeklyDigestDay: number;
  weeklyDigestTime: string;
};

export type NotificationPreferencesResponse = {
  items: NotificationPreference[];
  settings: NotificationSettings;
};

export type NotificationTripMute = {
  id: string;
  tripId: string;
  category?: NotificationCategory | null;
  mutedUntil?: string | null;
  createdAt: string;
  updatedAt: string;
};
export type NotificationTripMutesResponse = { items: NotificationTripMute[] };
