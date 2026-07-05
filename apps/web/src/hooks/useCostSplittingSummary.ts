import { useQuery } from "@tanstack/react-query";
import {
  costSplittingKeys,
  getCostSplittingSummary
} from "@/lib/api/cost-splitting";

export function useCostSplittingSummary({
  tripId,
  currency,
  enabled = true
}: {
  tripId: string;
  currency?: string | null;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: costSplittingKeys.summary(tripId, currency),
    queryFn: () => getCostSplittingSummary(tripId, currency),
    enabled: enabled && Boolean(tripId)
  });
}
