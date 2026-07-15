"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { notificationKeys } from "@/lib/api/notifications";
import { tripHealthKeys } from "@/lib/api/trip-health";
import {
  completeTripReminder,
  createTripReminder,
  deleteTripReminder,
  disableTripReminder,
  enableTripReminder,
  generateTripReminders,
  getAssignedReminders,
  getTripReminders,
  reminderKeys,
  reopenTripReminder,
  updateTripReminder
} from "@/lib/api/trip-reminders";
import type {
  CreateReminderInput,
  GenerateRemindersInput,
  ReminderListParams,
  UpdateReminderInput
} from "@/entities/trip-reminder/model";

export function useTripReminders(
  tripId: string,
  params: ReminderListParams = {},
  options: { enabled?: boolean } = {}
) {
  return useQuery({
    queryKey: reminderKeys.detail(tripId, params),
    queryFn: () => getTripReminders(tripId, params),
    enabled: (options.enabled ?? true) && Boolean(tripId)
  });
}

export function useAssignedReminders(
  params: ReminderListParams = {},
  options: { enabled?: boolean } = {}
) {
  return useQuery({
    queryKey: reminderKeys.assigned(params),
    queryFn: () => getAssignedReminders(params),
    enabled: options.enabled ?? true
  });
}

export function useGenerateTripReminders(tripId: string) {
  const queryClient = useReminderQueryClient(tripId);
  return useMutation({
    mutationFn: (input: GenerateRemindersInput) => generateTripReminders(tripId, input),
    onSuccess: async (data) => {
      queryClient.setLatest(data);
      await queryClient.invalidateAll();
    }
  });
}

export function useCreateTripReminder(tripId: string) {
  const queryClient = useReminderQueryClient(tripId);
  return useMutation({
    mutationFn: (input: CreateReminderInput) => createTripReminder(tripId, input),
    onSuccess: queryClient.invalidateAll
  });
}

export function useUpdateTripReminder(tripId: string) {
  const queryClient = useReminderQueryClient(tripId);
  return useMutation({
    mutationFn: ({
      reminderId,
      input
    }: {
      reminderId: string;
      input: UpdateReminderInput;
    }) => updateTripReminder(tripId, reminderId, input),
    onSuccess: queryClient.invalidateAll
  });
}

export function useCompleteTripReminder(tripId: string) {
  const queryClient = useReminderQueryClient(tripId);
  return useMutation({
    mutationFn: ({
      reminderId,
      completed
    }: {
      reminderId: string;
      completed: boolean;
    }) =>
      completed
        ? completeTripReminder(tripId, reminderId)
        : reopenTripReminder(tripId, reminderId),
    onSuccess: queryClient.invalidateAll
  });
}

export function useDisableTripReminder(tripId: string) {
  const queryClient = useReminderQueryClient(tripId);
  return useMutation({
    mutationFn: ({
      reminderId,
      disabled
    }: {
      reminderId: string;
      disabled: boolean;
    }) =>
      disabled
        ? disableTripReminder(tripId, reminderId)
        : enableTripReminder(tripId, reminderId),
    onSuccess: queryClient.invalidateAll
  });
}

export function useDeleteTripReminder(tripId: string) {
  const queryClient = useReminderQueryClient(tripId);
  return useMutation({
    mutationFn: (reminderId: string) => deleteTripReminder(tripId, reminderId),
    onSuccess: queryClient.invalidateAll
  });
}

function useReminderQueryClient(tripId: string) {
  const queryClient = useQueryClient();
  return {
    setLatest(data: unknown) {
      queryClient.setQueryData(reminderKeys.detail(tripId, {}), data);
    },
    invalidateSideEffects() {
      return Promise.all([
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      ]);
    },
    invalidateAll() {
      return Promise.all([
        queryClient.invalidateQueries({ queryKey: reminderKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      ]);
    }
  };
}
