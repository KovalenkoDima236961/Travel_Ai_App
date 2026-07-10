"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  routeAlternativeKeys,
  suggestTripRouteAlternatives
} from "@/lib/api/route-alternatives";
import type { SuggestTripRouteAlternativesInput } from "@/types/route-alternatives";

export function useSuggestTripRouteAlternatives(tripId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: SuggestTripRouteAlternativesInput) => {
      if (!tripId) {
        throw new Error("Trip is required.");
      }
      return suggestTripRouteAlternatives(tripId, input);
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.all }),
        tripId
          ? queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.tripSessions(tripId) })
          : Promise.resolve()
      ]);
    }
  });
}
