"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { tripKeys } from "@/lib/api/trips";
import {
  createTripFromRouteAlternative,
  routeAlternativeKeys
} from "@/lib/api/route-alternatives";
import type { CreateTripFromRouteAlternativeInput } from "@/types/route-alternatives";

export function useCreateTripFromRouteAlternative(sessionId?: string, alternativeId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTripFromRouteAlternativeInput) => {
      if (!sessionId || !alternativeId) {
        throw new Error("Select a route before creating a trip.");
      }
      return createTripFromRouteAlternative(sessionId, alternativeId, input);
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() }),
        queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.all })
      ]);
    }
  });
}
