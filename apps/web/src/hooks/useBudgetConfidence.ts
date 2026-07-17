import { useQuery } from "@tanstack/react-query";
import {
  budgetConfidenceKeys,
  getTripBudgetConfidence
} from "@/lib/api/budget-confidence";

export function useBudgetConfidence({
  tripId,
  currency,
  enabled = true
}: {
  tripId: string;
  currency?: string | null;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: budgetConfidenceKeys.detail(tripId, currency),
    queryFn: () => getTripBudgetConfidence(tripId, { currency }),
    enabled: enabled && Boolean(tripId),
    staleTime: 45 * 1000
  });
}
