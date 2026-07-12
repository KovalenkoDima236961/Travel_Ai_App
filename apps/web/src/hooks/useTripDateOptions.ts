"use client";

import { useQuery } from "@tanstack/react-query";
import { getTripDateOptions, tripAvailabilityKeys } from "@/lib/api/trip-availability";
import type { DateOptionsInput } from "@/types/trip-availability";

export function useTripDateOptions(
  tripId: string,
  input: DateOptionsInput = {},
  enabled = true
) {
  return useQuery({
    queryKey: tripAvailabilityKeys.dateOptions(tripId, input),
    queryFn: () => getTripDateOptions(tripId, input),
    enabled: enabled && Boolean(tripId)
  });
}
