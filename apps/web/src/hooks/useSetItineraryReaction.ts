"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { setItineraryItemReaction, tripDecisionKeys } from "@/lib/api/trip-decisions";
import type { SetItineraryItemReactionInput } from "@/types/trip-decisions";

export function useSetItineraryReaction(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: SetItineraryItemReactionInput) =>
      setItineraryItemReaction(tripId, input),
    onSuccess: async (summary) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.reactions(tripId) }),
        queryClient.invalidateQueries({
          queryKey: tripDecisionKeys.itemReactions(
            tripId,
            summary.dayNumber,
            summary.itemIndex
          )
        }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.groupPreferences(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all })
      ]);
    }
  });
}
