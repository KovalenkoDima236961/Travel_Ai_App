import { apiFetch } from "@/shared/api/client";
import type { ApprovalRiskResponse } from "@/entities/approval-risk/model";

export const approvalRiskKeys = {
  all: ["approval-risk"] as const,
  trip: (tripId: string) => ["approval-risk", "trip", tripId] as const
};

export function getTripApprovalRisk(tripId: string): Promise<ApprovalRiskResponse> {
  return apiFetch<ApprovalRiskResponse>(`/trips/${tripId}/approval-risk`);
}

