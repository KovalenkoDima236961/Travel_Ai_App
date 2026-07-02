import { apiFetch } from "@/lib/api/client";
import type { GenerationJob } from "@/types/generation-jobs";
import type {
  ApplyBudgetOptimizationProposalResponse,
  BudgetOptimizationJobRequest,
  BudgetOptimizationProposal,
  BudgetOptimizationProposalEnvelope,
  BudgetOptimizationProposalListResponse,
  BudgetOptimizationProposalStatus
} from "@/types/budget-optimization";

type GenerationJobEnvelope = {
  job: GenerationJob;
};

export const budgetOptimizationKeys = {
  all: (tripId: string) => ["budget-optimization", tripId] as const,
  lists: (tripId: string) => [...budgetOptimizationKeys.all(tripId), "list"] as const,
  list: (tripId: string, status?: BudgetOptimizationProposalStatus) =>
    [...budgetOptimizationKeys.lists(tripId), status ?? "all"] as const,
  detail: (tripId: string, proposalId: string) =>
    [...budgetOptimizationKeys.all(tripId), "detail", proposalId] as const
};

export async function createBudgetOptimizationJob(
  tripId: string,
  input: BudgetOptimizationJobRequest
): Promise<GenerationJob> {
  const response = await apiFetch<GenerationJobEnvelope>(
    `/trips/${tripId}/budget-optimization-jobs`,
    {
      method: "POST",
      body: JSON.stringify(cleanCreatePayload(input))
    }
  );
  return response.job;
}

export async function listBudgetOptimizationProposals(
  tripId: string,
  status?: BudgetOptimizationProposalStatus,
  limit = 20
): Promise<BudgetOptimizationProposal[]> {
  const searchParams = new URLSearchParams();
  if (status) {
    searchParams.set("status", status);
  }
  if (limit > 0) {
    searchParams.set("limit", String(limit));
  }

  const query = searchParams.toString();
  const response = await apiFetch<BudgetOptimizationProposalListResponse>(
    `/trips/${tripId}/budget-optimization-proposals${query ? `?${query}` : ""}`
  );
  return response.proposals;
}

export async function getBudgetOptimizationProposal(
  tripId: string,
  proposalId: string
): Promise<BudgetOptimizationProposal> {
  const response = await apiFetch<BudgetOptimizationProposalEnvelope>(
    `/trips/${tripId}/budget-optimization-proposals/${proposalId}`
  );
  return response.proposal;
}

export function applyBudgetOptimizationProposal(
  tripId: string,
  proposalId: string,
  expectedItineraryRevision: number
) {
  return apiFetch<ApplyBudgetOptimizationProposalResponse>(
    `/trips/${tripId}/budget-optimization-proposals/${proposalId}/apply`,
    {
      method: "POST",
      body: JSON.stringify({ expectedItineraryRevision })
    }
  );
}

export async function discardBudgetOptimizationProposal(
  tripId: string,
  proposalId: string
): Promise<BudgetOptimizationProposal> {
  const response = await apiFetch<BudgetOptimizationProposalEnvelope>(
    `/trips/${tripId}/budget-optimization-proposals/${proposalId}/discard`,
    {
      method: "POST"
    }
  );
  return response.proposal;
}

function cleanCreatePayload(input: BudgetOptimizationJobRequest) {
  const currency = input.currency?.trim().toUpperCase();
  const instruction = input.instruction?.trim();
  const constraints = input.constraints
    ? {
        preserveMustSeeItems: Boolean(input.constraints.preserveMustSeeItems),
        keepMealCount: Boolean(input.constraints.keepMealCount),
        avoidReplacingManualCosts: Boolean(input.constraints.avoidReplacingManualCosts),
        ...(input.constraints.maxWalkingIncreaseKm != null
          ? { maxWalkingIncreaseKm: input.constraints.maxWalkingIncreaseKm }
          : {})
      }
    : undefined;

  return {
    scope: input.scope,
    dayNumber: input.dayNumber,
    expectedItineraryRevision: input.expectedItineraryRevision,
    ...(input.targetReductionAmount != null
      ? { targetReductionAmount: input.targetReductionAmount }
      : {}),
    ...(currency ? { currency } : {}),
    ...(constraints ? { constraints } : {}),
    ...(instruction ? { instruction } : {})
  };
}
