"use client";

import { useQuery } from "@tanstack/react-query";
import { getTripPolls, tripDecisionKeys } from "@/lib/api/trip-decisions";

export function useTripPolls(tripId: string, enabled = true) {
  return useQuery({
    queryKey: tripDecisionKeys.polls(tripId),
    queryFn: () => getTripPolls(tripId),
    enabled: enabled && Boolean(tripId)
  });
}
