export type TripReminderCategory =
  | "documents"
  | "packing"
  | "transport"
  | "accommodation"
  | "weather"
  | "activities"
  | "group"
  | "checklist"
  | "before_departure"
  | "route"
  | "safety"
  | "other";

export type TripReminderPriority = "low" | "medium" | "high" | "critical";

export type TripReminderSource =
  | "checklist"
  | "route"
  | "transport"
  | "accommodation"
  | "weather"
  | "manual"
  | "system"
  | "regenerated";

export type TripReminderStatus =
  | "pending"
  | "sent"
  | "completed"
  | "disabled"
  | "cancelled"
  | "failed";

export type GenerateRemindersMode = "full" | "add_missing" | "category";

export const TRIP_REMINDER_CATEGORIES: TripReminderCategory[] = [
  "documents",
  "packing",
  "transport",
  "accommodation",
  "weather",
  "activities",
  "group",
  "checklist",
  "before_departure",
  "route",
  "safety",
  "other"
];

export const TRIP_REMINDER_PRIORITIES: TripReminderPriority[] = [
  "critical",
  "high",
  "medium",
  "low"
];

export const TRIP_REMINDER_STATUSES: TripReminderStatus[] = [
  "pending",
  "sent",
  "completed",
  "disabled",
  "cancelled",
  "failed"
];

export type TripReminder = {
  id: string;
  tripId: string;
  title: string;
  description?: string | null;
  category: TripReminderCategory;
  priority: TripReminderPriority;
  source: TripReminderSource;
  status: TripReminderStatus;
  triggerDate: string;
  triggerTime?: string | null;
  timezone?: string | null;
  relativeOffsetDays?: number | null;
  assignedToUserId?: string | null;
  assignedToDisplayName?: string | null;
  checklistItemId?: string | null;
  relatedDayNumber?: number | null;
  relatedItemIndex?: number | null;
  relatedItemId?: string | null;
  sentAt?: string | null;
  completedAt?: string | null;
  completedByUserId?: string | null;
  disabledAt?: string | null;
  disabledByUserId?: string | null;
  failureReason?: string | null;
  metadata?: Record<string, unknown>;
  createdByUserId?: string | null;
  updatedByUserId?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type ReminderSummary = {
  total: number;
  pending: number;
  completed: number;
  overdue: number;
  dueToday: number;
  highPriorityPending: number;
  assignedToMe: number;
  stale: boolean;
};

export type ReminderViewResponse = {
  reminders: TripReminder[];
  summary: ReminderSummary;
};

export type ReminderListParams = {
  status?: TripReminderStatus | "all";
  category?: TripReminderCategory | "all";
  assignedToMe?: boolean;
  upcomingOnly?: boolean;
  highPriority?: boolean;
  fromDate?: string;
  toDate?: string;
};

export type GenerateRemindersInput = {
  mode?: GenerateRemindersMode;
  categories?: TripReminderCategory[];
  preserveManualReminders?: boolean;
  preserveCompletedReminders?: boolean;
  replaceGeneratedPendingReminders?: boolean;
  instructions?: string;
};

export type CreateReminderInput = {
  title: string;
  description?: string | null;
  category: TripReminderCategory;
  priority?: TripReminderPriority;
  triggerDate: string;
  triggerTime?: string | null;
  timezone?: string | null;
  relativeOffsetDays?: number | null;
  assignedToUserId?: string | null;
  checklistItemId?: string | null;
  relatedDayNumber?: number | null;
  relatedItemIndex?: number | null;
  relatedItemId?: string | null;
  metadata?: Record<string, unknown>;
};

export type UpdateReminderInput = Partial<CreateReminderInput> & {
  clearDescription?: boolean;
  clearTriggerTime?: boolean;
  clearTimezone?: boolean;
  clearRelativeOffset?: boolean;
  clearAssignee?: boolean;
};
