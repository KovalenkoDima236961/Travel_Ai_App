"use client";

import { useQuery } from "@tanstack/react-query";
import {
  budgetOptimizationKeys,
  listBudgetOptimizationProposals
} from "@/lib/api/budget-optimization";
import type { BudgetOptimizationProposalStatus } from "@/entities/budget-optimization/model";

type UseBudgetOptimizationProposalsInput = {
  tripId: string;
  status?: BudgetOptimizationProposalStatus;
  enabled?: boolean;
};

export function useBudgetOptimizationProposals({
  tripId,
  status = "pending",
  enabled = true
}: UseBudgetOptimizationProposalsInput) {
  return useQuery({
    queryKey: budgetOptimizationKeys.list(tripId, status),
    queryFn: () => listBudgetOptimizationProposals(tripId, status),
    enabled: enabled && Boolean(tripId)
  });
}
