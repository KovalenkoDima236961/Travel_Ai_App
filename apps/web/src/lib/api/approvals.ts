import { apiFetch } from "@/shared/api/client";
import type {
  ApprovalDecisionInput,
  CancelApprovalInput,
  SubmitApprovalInput,
  TripApprovalEventsResponse,
  TripApprovalState,
  WorkspaceApprovalsResponse,
  WorkspaceApprovalStatusFilter
} from "@/entities/approval/model";

export const approvalKeys = {
  all: ["approvals"] as const,
  trip: (tripId: string) => ["trips", "detail", tripId, "approval"] as const,
  tripEvents: (tripId: string) => ["trips", "detail", tripId, "approval", "events"] as const,
  workspace: (workspaceId: string, status?: WorkspaceApprovalStatusFilter) =>
    ["approvals", "workspace", workspaceId, status ?? "default"] as const
};

export function getTripApproval(tripId: string): Promise<TripApprovalState> {
  return apiFetch<TripApprovalState>(`/trips/${tripId}/approval`);
}

export function submitTripApproval(
  tripId: string,
  input: SubmitApprovalInput
): Promise<TripApprovalState> {
  return apiFetch<TripApprovalState>(`/trips/${tripId}/approval/submit`, {
    method: "POST",
    body: JSON.stringify({
      note: input.note?.trim() || undefined,
      acknowledgedWarnings: input.acknowledgedWarnings ?? []
    })
  });
}

export function approveTrip(
  tripId: string,
  input: ApprovalDecisionInput
): Promise<TripApprovalState> {
  return apiFetch<TripApprovalState>(`/trips/${tripId}/approval/approve`, {
    method: "POST",
    body: JSON.stringify({ decisionNote: input.decisionNote?.trim() || undefined })
  });
}

export function requestTripChanges(
  tripId: string,
  input: ApprovalDecisionInput
): Promise<TripApprovalState> {
  return apiFetch<TripApprovalState>(`/trips/${tripId}/approval/request-changes`, {
    method: "POST",
    body: JSON.stringify({ decisionNote: input.decisionNote?.trim() || undefined })
  });
}

export function cancelTripApproval(
  tripId: string,
  input: CancelApprovalInput
): Promise<TripApprovalState> {
  return apiFetch<TripApprovalState>(`/trips/${tripId}/approval/cancel`, {
    method: "POST",
    body: JSON.stringify({ note: input.note?.trim() || undefined })
  });
}

export function listTripApprovalEvents(tripId: string): Promise<TripApprovalEventsResponse> {
  return apiFetch<TripApprovalEventsResponse>(`/trips/${tripId}/approval/events`);
}

export function listWorkspaceApprovals(
  workspaceId: string,
  params: { status?: WorkspaceApprovalStatusFilter; limit?: number; offset?: number } = {}
): Promise<WorkspaceApprovalsResponse> {
  const searchParams = new URLSearchParams();
  if (params.status) {
    searchParams.set("status", params.status);
  }
  if (params.limit != null) {
    searchParams.set("limit", String(params.limit));
  }
  if (params.offset != null) {
    searchParams.set("offset", String(params.offset));
  }
  const query = searchParams.toString();
  return apiFetch<WorkspaceApprovalsResponse>(
    `/workspaces/${workspaceId}/approvals${query ? `?${query}` : ""}`
  );
}
