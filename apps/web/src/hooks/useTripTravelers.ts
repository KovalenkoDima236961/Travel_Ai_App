import { useQuery } from "@tanstack/react-query";
import {
  costSplittingKeys,
  listTripTravelers
} from "@/lib/api/cost-splitting";

export function useTripTravelers({
  tripId,
  enabled = true
}: {
  tripId: string;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: costSplittingKeys.travelers(tripId),
    queryFn: () => listTripTravelers(tripId),
    enabled: enabled && Boolean(tripId)
  });
}
