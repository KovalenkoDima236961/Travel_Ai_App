export type WorkspaceRole = "owner" | "admin" | "member" | "viewer";

export type WorkspaceMemberStatus = "active" | "invited" | "removed";

export type WorkspaceInvitationStatus =
  | "pending"
  | "accepted"
  | "declined"
  | "revoked"
  | "expired";

export type Workspace = {
  id: string;
  name: string;
  slug: string;
  description?: string | null;
  currentUserRole: WorkspaceRole;
  memberCount: number;
  createdAt: string;
  updatedAt: string;
  archivedAt?: string | null;
};

export type WorkspaceMember = {
  id: string;
  workspaceId: string;
  userId: string;
  email?: string | null;
  displayName?: string | null;
  role: WorkspaceRole;
  status: WorkspaceMemberStatus;
  invitedByUserId?: string | null;
  invitedAt?: string | null;
  joinedAt?: string | null;
  removedAt?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type WorkspaceInvitation = {
  id: string;
  workspaceId: string;
  workspaceName: string;
  email: string;
  invitedUserId?: string | null;
  role: Exclude<WorkspaceRole, "owner">;
  status: WorkspaceInvitationStatus;
  invitedByUserId: string;
  expiresAt?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type WorkspaceAccess = {
  hasAccess: boolean;
  role?: WorkspaceRole;
  status?: WorkspaceMemberStatus;
  workspaceArchived: boolean;
};

export type CreateWorkspaceInput = {
  name: string;
  description?: string | null;
};

export type UpdateWorkspaceInput = {
  name?: string;
  description?: string | null;
};

export type InviteWorkspaceMemberInput = {
  email: string;
  role: Exclude<WorkspaceRole, "owner">;
};

export type UpdateWorkspaceMemberRoleInput = {
  role: Exclude<WorkspaceRole, "owner">;
};

export type WorkspacesResponse = {
  workspaces: Workspace[];
};

export type WorkspaceMembersResponse = {
  members: WorkspaceMember[];
};

export type WorkspaceInvitationsResponse = {
  invitations: WorkspaceInvitation[];
};
