import { apiFetch } from "@/shared/api/client";
import type {
  PolicyEvaluation,
  UpsertWorkspacePolicyInput,
  WorkspacePolicy,
  WorkspacePolicyResponse
} from "@/types/workspace-policy";

export const workspacePolicyKeys = {
  all: ["workspace-policies"] as const,
  workspace: (workspaceId: string) =>
    [...workspacePolicyKeys.all, "workspace", workspaceId] as const,
  evaluations: () => [...workspacePolicyKeys.all, "evaluation"] as const,
  evaluation: (tripId: string) =>
    [...workspacePolicyKeys.evaluations(), tripId] as const
};

export function getWorkspacePolicy(workspaceId: string) {
  return apiFetch<WorkspacePolicyResponse>(`/workspaces/${workspaceId}/policy`);
}

export async function upsertWorkspacePolicy(
  workspaceId: string,
  input: UpsertWorkspacePolicyInput
) {
  const response = await apiFetch<{ policy: WorkspacePolicy }>(
    `/workspaces/${workspaceId}/policy`,
    { method: "PUT", body: JSON.stringify(input) }
  );
  return response.policy;
}

export async function archiveWorkspacePolicy(workspaceId: string) {
  const response = await apiFetch<{ policy: WorkspacePolicy }>(
    `/workspaces/${workspaceId}/policy/archive`,
    { method: "POST", body: "{}" }
  );
  return response.policy;
}

export function evaluateTripPolicy(tripId: string) {
  return apiFetch<PolicyEvaluation>(`/trips/${tripId}/policy/evaluate`, {
    method: "POST",
    body: "{}"
  });
}

export function getTripPolicyEvaluation(tripId: string) {
  return apiFetch<PolicyEvaluation>(`/trips/${tripId}/policy/evaluation`);
}
