"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  refineRouteAlternatives,
  routeAlternativeKeys
} from "@/lib/api/route-alternatives";
import type { RefineRouteAlternativesInput } from "@/types/route-alternatives";

export function useRefineRouteAlternatives(sessionId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: RefineRouteAlternativesInput) => {
      if (!sessionId) {
        throw new Error("Route alternative session is required.");
      }
      return refineRouteAlternatives(sessionId, input);
    },
    onSuccess: async (_session, _input) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.all }),
        sessionId
          ? queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.session(sessionId) })
          : Promise.resolve()
      ]);
    }
  });
}
