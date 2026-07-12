"use client";

import { useQuery } from "@tanstack/react-query";
import { getTripAvailability, tripAvailabilityKeys } from "@/lib/api/trip-availability";

export function useTripAvailability(tripId: string, enabled = true) {
  return useQuery({
    queryKey: tripAvailabilityKeys.list(tripId),
    queryFn: () => getTripAvailability(tripId),
    enabled: enabled && Boolean(tripId)
  });
}
