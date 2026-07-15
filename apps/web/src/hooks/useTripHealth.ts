"use client";

import { useQuery } from "@tanstack/react-query";
import { getTripHealth, tripHealthKeys } from "@/lib/api/trip-health";

export function useTripHealth(
  tripId: string,
  options: { enabled?: boolean } = {}
) {
  return useQuery({
    queryKey: tripHealthKeys.detail(tripId),
    queryFn: () => getTripHealth(tripId),
    enabled: (options.enabled ?? true) && Boolean(tripId),
    staleTime: 45 * 1000
  });
}
