import { apiFetch } from "@/shared/api/client";
import type { GenerationJob } from "@/entities/generation-job/model";
import type {
  ApplyRepairProposalResponse,
  CreateRepairJobInput,
  RepairProposal,
  RepairProposalDetail,
  RepairProposalEnvelope,
  RepairProposalListResponse,
  RepairProposalStatus
} from "@/entities/trip-repair/model";

type GenerationJobEnvelope = {
  job: GenerationJob;
};

export const tripRepairKeys = {
  all: (tripId: string) => ["trip-repair", tripId] as const,
  lists: (tripId: string) => [...tripRepairKeys.all(tripId), "list"] as const,
  list: (tripId: string, status?: RepairProposalStatus) =>
    [...tripRepairKeys.lists(tripId), status ?? "all"] as const,
  detail: (tripId: string, proposalId: string) =>
    [...tripRepairKeys.all(tripId), "detail", proposalId] as const,
  job: (tripId: string, jobId: string) => [...tripRepairKeys.all(tripId), "job", jobId] as const
};

export async function createTripRepairJob(
  tripId: string,
  input: CreateRepairJobInput
): Promise<GenerationJob> {
  const response = await apiFetch<GenerationJobEnvelope>(`/trips/${tripId}/repair-jobs`, {
    method: "POST",
    body: JSON.stringify(cleanCreatePayload(input))
  });
  return response.job;
}

export async function getTripRepairJob(tripId: string, jobId: string): Promise<GenerationJob> {
  const response = await apiFetch<GenerationJobEnvelope>(
    `/trips/${tripId}/repair-jobs/${jobId}`
  );
  return response.job;
}

export async function listTripRepairProposals(
  tripId: string,
  status?: RepairProposalStatus,
  limit = 20
): Promise<RepairProposal[]> {
  const searchParams = new URLSearchParams();
  if (status) {
    searchParams.set("status", status);
  }
  if (limit > 0) {
    searchParams.set("limit", String(limit));
  }
  const query = searchParams.toString();
  const response = await apiFetch<RepairProposalListResponse>(
    `/trips/${tripId}/repair-proposals${query ? `?${query}` : ""}`
  );
  return response.proposals;
}

export async function getTripRepairProposal(
  tripId: string,
  proposalId: string
): Promise<RepairProposalDetail> {
  const response = await apiFetch<RepairProposalEnvelope>(
    `/trips/${tripId}/repair-proposals/${proposalId}`
  );
  return response.proposal;
}

export function applyTripRepairProposal(
  tripId: string,
  proposalId: string,
  expectedItineraryRevision: number
) {
  return apiFetch<ApplyRepairProposalResponse>(
    `/trips/${tripId}/repair-proposals/${proposalId}/apply`,
    {
      method: "POST",
      body: JSON.stringify({ expectedItineraryRevision })
    }
  );
}

export async function discardTripRepairProposal(
  tripId: string,
  proposalId: string,
  reason?: string | null
): Promise<RepairProposalDetail> {
  const response = await apiFetch<RepairProposalEnvelope>(
    `/trips/${tripId}/repair-proposals/${proposalId}/discard`,
    {
      method: "POST",
      body: reason?.trim() ? JSON.stringify({ reason: reason.trim() }) : undefined
    }
  );
  return response.proposal;
}

function cleanCreatePayload(input: CreateRepairJobInput) {
  const specialInstructions = input.specialInstructions?.trim();
  const constraints = input.constraints
    ? {
        preserveConfirmedItems: input.constraints.preserveConfirmedItems ?? true,
        minimizeChanges: input.constraints.minimizeChanges ?? true,
        preserveUserEditedItems: input.constraints.preserveUserEditedItems ?? true,
        doNotChangeAccommodation: input.constraints.doNotChangeAccommodation ?? false,
        doNotChangeDates: input.constraints.doNotChangeDates ?? true,
        ...(input.constraints.maxChangedItems != null
          ? { maxChangedItems: input.constraints.maxChangedItems }
          : {})
      }
    : undefined;

  return {
    expectedItineraryRevision: input.expectedItineraryRevision,
    repairMode: input.repairMode,
    selectedIssueTypes: input.selectedIssueTypes ?? [],
    selectedRiskFactorTypes: input.selectedRiskFactorTypes ?? [],
    ...(constraints ? { constraints } : {}),
    ...(specialInstructions ? { specialInstructions } : {})
  };
}
