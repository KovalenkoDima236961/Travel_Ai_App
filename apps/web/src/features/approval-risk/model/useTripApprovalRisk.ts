import { useQuery } from "@tanstack/react-query";

import { approvalRiskKeys, getTripApprovalRisk } from "@/lib/api/approval-risk";

export function useTripApprovalRisk(tripId: string, enabled = true) {
  return useQuery({
    queryKey: approvalRiskKeys.trip(tripId),
    queryFn: () => getTripApprovalRisk(tripId),
    enabled: enabled && Boolean(tripId),
    refetchOnWindowFocus: true
  });
}

