"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { notificationKeys } from "@/lib/api/notifications";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { createTripPoll, tripDecisionKeys } from "@/lib/api/trip-decisions";
import type { CreateTripPollInput } from "@/types/trip-decisions";

export function useCreateTripPoll(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTripPollInput) => createTripPoll(tripId, input),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.polls(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.groupPreferences(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      ]);
    }
  });
}
