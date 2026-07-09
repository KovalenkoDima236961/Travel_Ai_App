"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { discardTripRepairProposal, tripRepairKeys } from "@/lib/api/trip-repair";
import type { RepairProposal } from "@/entities/trip-repair/model";

export function useDiscardTripRepairProposal(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (proposal: RepairProposal) => discardTripRepairProposal(tripId, proposal.id),
    onSuccess: () => {
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
    }
  });
}
