"use client";

import { useQuery } from "@tanstack/react-query";
import { getGroupPreferences, tripDecisionKeys } from "@/lib/api/trip-decisions";

export function useGroupPreferences(tripId: string, enabled = true) {
  return useQuery({
    queryKey: tripDecisionKeys.groupPreferences(tripId),
    queryFn: () => getGroupPreferences(tripId),
    enabled: enabled && Boolean(tripId)
  });
}
