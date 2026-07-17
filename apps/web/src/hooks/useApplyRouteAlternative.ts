"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { applyRouteAlternative, routeAlternativeKeys } from "@/lib/api/route-alternatives";
import { tripKeys } from "@/lib/api/trips";
import type { ApplyRouteAlternativeInput } from "@/types/route-alternatives";
import { queryKeys } from "@/lib/query-keys";

export function useApplyRouteAlternative(tripId?: string, sessionId?: string, alternativeId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ApplyRouteAlternativeInput) => {
      if (!tripId || !sessionId || !alternativeId) {
        throw new Error("Select a route alternative before applying.");
      }
      return applyRouteAlternative(tripId, sessionId, alternativeId, input);
    },
    onSuccess: async () => {
      if (!tripId) {
        return;
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.route(tripId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.commandCenter(tripId) }),
        queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.all }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
    }
  });
}
