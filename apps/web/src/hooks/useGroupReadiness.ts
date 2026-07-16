"use client";

import { useQuery } from "@tanstack/react-query";
import { getGroupReadiness, groupReadinessKeys } from "@/lib/api/group-readiness";

export function useGroupReadiness(tripId: string, enabled = true) {
  return useQuery({
    queryKey: groupReadinessKeys.detail(tripId),
    queryFn: () => getGroupReadiness(tripId),
    enabled: enabled && Boolean(tripId)
  });
}

