"use client";

import { useQuery } from "@tanstack/react-query";
import { checklistKeys, getTripChecklist } from "@/lib/api/checklists";

export function useTripChecklist(
  tripId: string,
  options: { enabled?: boolean } = {}
) {
  return useQuery({
    queryKey: checklistKeys.detail(tripId),
    queryFn: () => getTripChecklist(tripId),
    enabled: (options.enabled ?? true) && Boolean(tripId)
  });
}

