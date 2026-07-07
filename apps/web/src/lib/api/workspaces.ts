import { apiFetch } from "@/shared/api/client";
import { getUserApiBaseUrl } from "@/shared/config";
import type {
  CreateWorkspaceInput,
  InviteWorkspaceMemberInput,
  UpdateWorkspaceInput,
  UpdateWorkspaceMemberRoleInput,
  Workspace,
  WorkspaceInvitation,
  WorkspaceInvitationsResponse,
  WorkspaceMember,
  WorkspaceMembersResponse,
  WorkspacesResponse
} from "@/entities/workspace/model";

export const workspaceKeys = {
  all: ["workspaces"] as const,
  lists: () => [...workspaceKeys.all, "list"] as const,
  list: () => [...workspaceKeys.lists()] as const,
  details: () => [...workspaceKeys.all, "detail"] as const,
  detail: (workspaceId: string) => [...workspaceKeys.details(), workspaceId] as const,
  members: (workspaceId: string) => [...workspaceKeys.detail(workspaceId), "members"] as const,
  invitations: () => [...workspaceKeys.all, "invitations"] as const
};

export function createWorkspace(input: CreateWorkspaceInput) {
  return workspaceFetch<Workspace>("/workspaces", {
    method: "POST",
    body: JSON.stringify(cleanWorkspacePayload(input))
  });
}

export async function listWorkspaces() {
  const response = await workspaceFetch<WorkspacesResponse>("/workspaces");
  return response.workspaces;
}

export function getWorkspace(workspaceId: string) {
  return workspaceFetch<Workspace>(`/workspaces/${workspaceId}`);
}

export function updateWorkspace(workspaceId: string, input: UpdateWorkspaceInput) {
  return workspaceFetch<Workspace>(`/workspaces/${workspaceId}`, {
    method: "PATCH",
    body: JSON.stringify(cleanWorkspacePayload(input))
  });
}

export function archiveWorkspace(workspaceId: string) {
  return workspaceFetch<Workspace>(`/workspaces/${workspaceId}`, {
    method: "DELETE"
  });
}

export async function listWorkspaceMembers(workspaceId: string) {
  const response = await workspaceFetch<WorkspaceMembersResponse>(
    `/workspaces/${workspaceId}/members`
  );
  return response.members;
}

export function inviteWorkspaceMember(workspaceId: string, input: InviteWorkspaceMemberInput) {
  return workspaceFetch<WorkspaceInvitation>(`/workspaces/${workspaceId}/members/invite`, {
    method: "POST",
    body: JSON.stringify({
      email: input.email.trim().toLowerCase(),
      role: input.role
    })
  });
}

export function updateWorkspaceMemberRole(
  workspaceId: string,
  memberId: string,
  input: UpdateWorkspaceMemberRoleInput
) {
  return workspaceFetch<WorkspaceMember>(`/workspaces/${workspaceId}/members/${memberId}`, {
    method: "PATCH",
    body: JSON.stringify({ role: input.role })
  });
}

export function removeWorkspaceMember(workspaceId: string, memberId: string) {
  return workspaceFetch<{ success: boolean }>(`/workspaces/${workspaceId}/members/${memberId}`, {
    method: "DELETE"
  });
}

export async function listWorkspaceInvitations() {
  const response = await workspaceFetch<WorkspaceInvitationsResponse>("/workspace-invitations");
  return response.invitations;
}

export function acceptWorkspaceInvitation(invitationId: string) {
  return workspaceFetch<Workspace>(`/workspace-invitations/${invitationId}/accept`, {
    method: "POST"
  });
}

export function declineWorkspaceInvitation(invitationId: string) {
  return workspaceFetch<{ success: boolean }>(`/workspace-invitations/${invitationId}/decline`, {
    method: "POST"
  });
}

function workspaceFetch<T>(path: string, init: RequestInit = {}) {
  return apiFetch<T>(path, init, {
    baseUrl: getUserApiBaseUrl(),
    serviceName: "User Service"
  });
}

function cleanWorkspacePayload(input: CreateWorkspaceInput | UpdateWorkspaceInput) {
  const description = input.description?.trim();

  return {
    ...("name" in input && input.name != null ? { name: input.name.trim() } : {}),
    ...(input.description !== undefined ? { description: description ? description : null } : {})
  };
}
