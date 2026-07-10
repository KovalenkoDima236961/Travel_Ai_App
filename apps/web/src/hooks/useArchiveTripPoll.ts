"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { archiveTripPoll, tripDecisionKeys } from "@/lib/api/trip-decisions";

export function useArchiveTripPoll(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (pollId: string) => archiveTripPoll(tripId, pollId),
    onSuccess: async (_poll, pollId) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.polls(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.poll(tripId, pollId) }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.groupPreferences(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
    }
  });
}
