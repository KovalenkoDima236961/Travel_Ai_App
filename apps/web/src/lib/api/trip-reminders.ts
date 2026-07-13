import { apiFetch } from "@/shared/api/client";
import type {
  CreateReminderInput,
  GenerateRemindersInput,
  ReminderListParams,
  ReminderViewResponse,
  TripReminder,
  UpdateReminderInput
} from "@/entities/trip-reminder/model";

export const reminderKeys = {
  all: ["trip-reminders"] as const,
  detail: (tripId: string, params: ReminderListParams = {}) =>
    [...reminderKeys.all, tripId, params] as const,
  assigned: (params: ReminderListParams = {}) =>
    [...reminderKeys.all, "assigned-to-me", params] as const
};

export function getTripReminders(tripId: string, params: ReminderListParams = {}) {
  return apiFetch<ReminderViewResponse>(`/trips/${tripId}/reminders${toQuery(params)}`);
}

export function getAssignedReminders(params: ReminderListParams = {}) {
  return apiFetch<ReminderViewResponse>(`/reminders/assigned-to-me${toQuery(params)}`);
}

export function generateTripReminders(tripId: string, input: GenerateRemindersInput = {}) {
  return apiFetch<ReminderViewResponse>(`/trips/${tripId}/reminders/generate`, {
    method: "POST",
    body: JSON.stringify(cleanGeneratePayload(input))
  });
}

export function createTripReminder(tripId: string, input: CreateReminderInput) {
  return apiFetch<TripReminder>(`/trips/${tripId}/reminders`, {
    method: "POST",
    body: JSON.stringify(cleanReminderPayload(input))
  });
}

export function updateTripReminder(
  tripId: string,
  reminderId: string,
  input: UpdateReminderInput
) {
  return apiFetch<TripReminder>(`/trips/${tripId}/reminders/${reminderId}`, {
    method: "PATCH",
    body: JSON.stringify(cleanUpdatePayload(input))
  });
}

export function completeTripReminder(tripId: string, reminderId: string) {
  return apiFetch<TripReminder>(`/trips/${tripId}/reminders/${reminderId}/complete`, {
    method: "POST"
  });
}

export function reopenTripReminder(tripId: string, reminderId: string) {
  return apiFetch<TripReminder>(`/trips/${tripId}/reminders/${reminderId}/reopen`, {
    method: "POST"
  });
}

export function disableTripReminder(tripId: string, reminderId: string) {
  return apiFetch<TripReminder>(`/trips/${tripId}/reminders/${reminderId}/disable`, {
    method: "POST"
  });
}

export function enableTripReminder(tripId: string, reminderId: string) {
  return apiFetch<TripReminder>(`/trips/${tripId}/reminders/${reminderId}/enable`, {
    method: "POST"
  });
}

export function deleteTripReminder(tripId: string, reminderId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/reminders/${reminderId}`, {
    method: "DELETE"
  });
}

function toQuery(params: ReminderListParams) {
  const query = new URLSearchParams();
  if (params.status && params.status !== "all") {
    query.set("status", params.status);
  }
  if (params.category && params.category !== "all") {
    query.set("category", params.category);
  }
  if (params.assignedToMe) {
    query.set("assignedToMe", "true");
  }
  if (params.upcomingOnly) {
    query.set("upcomingOnly", "true");
  }
  if (params.highPriority) {
    query.set("highPriority", "true");
  }
  if (params.fromDate?.trim()) {
    query.set("fromDate", params.fromDate.trim());
  }
  if (params.toDate?.trim()) {
    query.set("toDate", params.toDate.trim());
  }
  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
}

function cleanGeneratePayload(input: GenerateRemindersInput) {
  return removeUndefined({
    mode: input.mode ?? "add_missing",
    categories: input.categories ?? [],
    preserveManualReminders: input.preserveManualReminders ?? true,
    preserveCompletedReminders: input.preserveCompletedReminders ?? true,
    replaceGeneratedPendingReminders: input.replaceGeneratedPendingReminders ?? false,
    instructions: emptyToUndefined(input.instructions)
  });
}

function cleanReminderPayload(input: CreateReminderInput) {
  return removeUndefined({
    title: input.title.trim(),
    description: emptyToUndefined(input.description),
    category: input.category,
    priority: input.priority ?? "medium",
    triggerDate: input.triggerDate,
    triggerTime: emptyToUndefined(input.triggerTime),
    timezone: emptyToUndefined(input.timezone),
    relativeOffsetDays: input.relativeOffsetDays ?? undefined,
    assignedToUserId: emptyToUndefined(input.assignedToUserId),
    checklistItemId: emptyToUndefined(input.checklistItemId),
    relatedDayNumber: input.relatedDayNumber ?? undefined,
    relatedItemIndex: input.relatedItemIndex ?? undefined,
    relatedItemId: emptyToUndefined(input.relatedItemId),
    metadata: input.metadata
  });
}

function cleanUpdatePayload(input: UpdateReminderInput) {
  return removeUndefined({
    title: input.title?.trim(),
    description: emptyToUndefined(input.description),
    clearDescription: input.clearDescription || undefined,
    category: input.category,
    priority: input.priority,
    triggerDate: input.triggerDate,
    triggerTime: emptyToUndefined(input.triggerTime),
    clearTriggerTime: input.clearTriggerTime || undefined,
    timezone: emptyToUndefined(input.timezone),
    clearTimezone: input.clearTimezone || undefined,
    relativeOffsetDays: input.relativeOffsetDays ?? undefined,
    clearRelativeOffset: input.clearRelativeOffset || undefined,
    assignedToUserId: emptyToUndefined(input.assignedToUserId),
    clearAssignee: input.clearAssignee || undefined,
    metadata: input.metadata
  });
}

function emptyToUndefined(value: string | null | undefined) {
  if (value == null) {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function removeUndefined<T extends Record<string, unknown>>(value: T) {
  return Object.fromEntries(
    Object.entries(value).filter(([, entry]) => entry !== undefined)
  ) as Partial<T>;
}
