"use client";

import { useQuery } from "@tanstack/react-query";
import { listTripRepairProposals, tripRepairKeys } from "@/lib/api/trip-repair";
import type { RepairProposalStatus } from "@/entities/trip-repair/model";

type UseTripRepairProposalsInput = {
  tripId: string;
  status?: RepairProposalStatus;
  enabled?: boolean;
};

export function useTripRepairProposals({
  tripId,
  status = "pending",
  enabled = true
}: UseTripRepairProposalsInput) {
  return useQuery({
    queryKey: tripRepairKeys.list(tripId, status),
    queryFn: () => listTripRepairProposals(tripId, status),
    enabled: enabled && Boolean(tripId)
  });
}
