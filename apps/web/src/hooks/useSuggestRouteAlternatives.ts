"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  routeAlternativeKeys,
  suggestRouteAlternatives
} from "@/lib/api/route-alternatives";

export function useSuggestRouteAlternatives() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: suggestRouteAlternatives,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: routeAlternativeKeys.all });
    }
  });
}
