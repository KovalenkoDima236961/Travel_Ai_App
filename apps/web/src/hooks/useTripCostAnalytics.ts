import { useQuery } from "@tanstack/react-query";
import {
  costAnalyticsKeys,
  getTripCostAnalytics
} from "@/lib/api/cost-analytics";

export function useTripCostAnalytics({
  tripId,
  currency,
  enabled = true
}: {
  tripId: string;
  currency?: string | null;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: costAnalyticsKeys.trip(tripId, currency),
    queryFn: () => getTripCostAnalytics(tripId, currency),
    enabled: enabled && Boolean(tripId)
  });
}
