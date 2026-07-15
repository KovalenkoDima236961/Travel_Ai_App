"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { tripDecisionKeys, voteTripPoll } from "@/lib/api/trip-decisions";
import type { VoteTripPollInput } from "@/types/trip-decisions";

export function useVoteTripPoll(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ pollId, input }: { pollId: string; input: VoteTripPollInput }) =>
      voteTripPoll(tripId, pollId, input),
    onSuccess: async (_poll, variables) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.polls(tripId) }),
        queryClient.invalidateQueries({
          queryKey: tripDecisionKeys.poll(tripId, variables.pollId)
        }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.groupPreferences(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all })
      ]);
    }
  });
}
