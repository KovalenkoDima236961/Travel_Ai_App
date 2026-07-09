"use client";

import { useQuery } from "@tanstack/react-query";
import { getTripRepairProposal, tripRepairKeys } from "@/lib/api/trip-repair";

type UseTripRepairProposalInput = {
  tripId: string;
  proposalId?: string | null;
  enabled?: boolean;
};

export function useTripRepairProposal({
  tripId,
  proposalId,
  enabled = true
}: UseTripRepairProposalInput) {
  return useQuery({
    queryKey: proposalId ? tripRepairKeys.detail(tripId, proposalId) : tripRepairKeys.all(tripId),
    queryFn: () => getTripRepairProposal(tripId, proposalId ?? ""),
    enabled: enabled && Boolean(tripId) && Boolean(proposalId)
  });
}
