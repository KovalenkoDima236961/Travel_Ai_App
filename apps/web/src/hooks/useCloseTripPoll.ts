"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { notificationKeys } from "@/lib/api/notifications";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { closeTripPoll, tripDecisionKeys } from "@/lib/api/trip-decisions";

export function useCloseTripPoll(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (pollId: string) => closeTripPoll(tripId, pollId),
    onSuccess: async (_poll, pollId) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.polls(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.poll(tripId, pollId) }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.groupPreferences(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      ]);
    }
  });
}
