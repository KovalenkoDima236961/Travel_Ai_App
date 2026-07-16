"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { approvalRiskKeys } from "@/lib/api/approval-risk";
import { budgetKeys } from "@/lib/api/budget";
import { budgetConfidenceKeys } from "@/lib/api/budget-confidence";
import { activityKeys } from "@/lib/api/activity";
import { applyTripRepairProposal, tripRepairKeys } from "@/lib/api/trip-repair";
import { tripKeys } from "@/lib/api/trips";
import type { RepairProposal } from "@/entities/trip-repair/model";

export function useApplyTripRepairProposal(tripId: string, expectedItineraryRevision: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (proposal: RepairProposal) =>
      applyTripRepairProposal(tripId, proposal.id, expectedItineraryRevision),
    onSuccess: (result) => {
      queryClient.setQueryData(tripKeys.detail(tripId), result.trip);
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetConfidenceKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
    }
  });
}
