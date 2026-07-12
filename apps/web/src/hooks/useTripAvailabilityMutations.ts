"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { generationJobKeys } from "@/lib/api/generation-jobs";
import { notificationKeys } from "@/lib/api/notifications";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { tripAvailabilityKeys } from "@/lib/api/trip-availability";
import {
  applyTripDateOption,
  createDateOptionsPoll,
  deleteMyTripAvailability,
  generateTripDateOptions,
  requestTripAvailability,
  upsertMyTripAvailability
} from "@/lib/api/trip-availability";
import { tripDecisionKeys } from "@/lib/api/trip-decisions";
import { tripKeys } from "@/lib/api/trips";
import type {
  ApplyDateOptionInput,
  CreateDateOptionsPollInput,
  DateOptionsInput,
  RequestTripAvailabilityInput,
  UpsertTripAvailabilityInput
} from "@/types/trip-availability";

export function useUpsertTripAvailability(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpsertTripAvailabilityInput) => upsertMyTripAvailability(tripId, input),
    onSuccess: async () => {
      await invalidateAvailability(queryClient, tripId);
    }
  });
}

export function useDeleteTripAvailability(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => deleteMyTripAvailability(tripId),
    onSuccess: async () => {
      await invalidateAvailability(queryClient, tripId);
    }
  });
}

export function useRequestTripAvailability(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: RequestTripAvailabilityInput) => requestTripAvailability(tripId, input),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      ]);
    }
  });
}

export function useGenerateTripDateOptions(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: DateOptionsInput) => generateTripDateOptions(tripId, input),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: tripAvailabilityKeys.all(tripId) });
    }
  });
}

export function useApplyTripDateOption(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ optionId, input }: { optionId: string; input: ApplyDateOptionInput }) =>
      applyTripDateOption(tripId, optionId, input),
    onSuccess: async (result) => {
      queryClient.setQueryData(tripKeys.detail(tripId), result.trip);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripAvailabilityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all }),
        queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) })
      ]);
    }
  });
}

export function useCreateDateOptionsPoll(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateDateOptionsPollInput) => createDateOptionsPoll(tripId, input),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.polls(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.groupPreferences(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      ]);
    }
  });
}

async function invalidateAvailability(
  queryClient: ReturnType<typeof useQueryClient>,
  tripId: string
) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: tripAvailabilityKeys.all(tripId) }),
    queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
    queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
    queryClient.invalidateQueries({ queryKey: notificationKeys.all })
  ]);
}
