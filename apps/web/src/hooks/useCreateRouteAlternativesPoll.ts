"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { createRouteAlternativesPoll, routeAlternativeKeys } from "@/lib/api/route-alternatives";
import { tripDecisionKeys } from "@/lib/api/trip-decisions";
import type { CreateRouteAlternativesPollInput } from "@/types/route-alternatives";

export function useCreateRouteAlternativesPoll(tripId?: string, sessionId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateRouteAlternativesPollInput) => {
      if (!tripId || !sessionId) {
        throw new Error("Route alternative session is required.");
      }
      return createRouteAlternativesPoll(tripId, sessionId, input);
    },
    onSuccess: async () => {
      if (!tripId) {
        return;
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.all }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.polls(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripDecisionKeys.groupPreferences(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
    }
  });
}
