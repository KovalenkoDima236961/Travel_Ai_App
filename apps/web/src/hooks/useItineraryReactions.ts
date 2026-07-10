"use client";

import { useQuery } from "@tanstack/react-query";
import { getItineraryReactions, tripDecisionKeys } from "@/lib/api/trip-decisions";

export function useItineraryReactions(tripId: string, enabled = true) {
  return useQuery({
    queryKey: tripDecisionKeys.reactions(tripId),
    queryFn: () => getItineraryReactions(tripId),
    enabled: enabled && Boolean(tripId)
  });
}
