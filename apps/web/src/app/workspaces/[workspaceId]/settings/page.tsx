"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { useAuth } from "@/components/auth/AuthProvider";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import {
  canArchiveWorkspace,
  canInviteWorkspaceMembers,
  canManageWorkspace,
  formatWorkspaceRole,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import {
  archiveWorkspace,
  getWorkspace,
  inviteWorkspaceMember,
  listWorkspaceMembers,
  removeWorkspaceMember,
  updateWorkspace,
  updateWorkspaceMemberRole,
  workspaceKeys
} from "@/lib/api/workspaces";
import { getErrorMessage } from "@/lib/utils";
import type { WorkspaceMember, WorkspaceRole } from "@/types/workspace";

const inviteRoles: Array<Exclude<WorkspaceRole, "owner">> = ["admin", "member", "viewer"];

export default function WorkspaceSettingsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceSettingsPageContent />
    </ProtectedRoute>
  );
}

function WorkspaceSettingsPageContent() {
  const params = useParams<{ workspaceId: string }>();
  const workspaceId = params.workspaceId;
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { setAllTrips, refreshWorkspaces } = useWorkspaces();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<Exclude<WorkspaceRole, "owner">>("member");
  const [formError, setFormError] = useState<string | null>(null);

  const workspaceQuery = useQuery({
    queryKey: workspaceKeys.detail(workspaceId),
    queryFn: () => getWorkspace(workspaceId),
    enabled: Boolean(workspaceId)
  });
  const membersQuery = useQuery({
    queryKey: workspaceKeys.members(workspaceId),
    queryFn: () => listWorkspaceMembers(workspaceId),
    enabled: Boolean(workspaceId)
  });

  const workspace = workspaceQuery.data;
  const canManage = workspace ? canManageWorkspace(workspace.currentUserRole) : false;
  const canInvite = workspace ? canInviteWorkspaceMembers(workspace.currentUserRole) : false;
  const canArchive = workspace ? canArchiveWorkspace(workspace.currentUserRole) : false;

  useEffect(() => {
    if (workspace) {
      setName(workspace.name);
      setDescription(workspace.description ?? "");
    }
  }, [workspace]);

  const updateMutation = useMutation({
    mutationFn: () =>
      updateWorkspace(workspaceId, {
        name,
        description: description.trim() || null
      }),
    onSuccess: async () => {
      await invalidateWorkspace(queryClient, workspaceId);
      await refreshWorkspaces();
    }
  });

  const inviteMutation = useMutation({
    mutationFn: () =>
      inviteWorkspaceMember(workspaceId, {
        email: inviteEmail,
        role: inviteRole
      }),
    onSuccess: async () => {
      setInviteEmail("");
      setInviteRole("member");
      await invalidateWorkspace(queryClient, workspaceId);
    }
  });

  const roleMutation = useMutation({
    mutationFn: ({ memberId, role }: { memberId: string; role: Exclude<WorkspaceRole, "owner"> }) =>
      updateWorkspaceMemberRole(workspaceId, memberId, { role }),
    onSuccess: async () => {
      await invalidateWorkspace(queryClient, workspaceId);
      await refreshWorkspaces();
    }
  });

  const removeMutation = useMutation({
    mutationFn: (memberId: string) => removeWorkspaceMember(workspaceId, memberId),
    onSuccess: async () => {
      await invalidateWorkspace(queryClient, workspaceId);
      await refreshWorkspaces();
    }
  });

  const archiveMutation = useMutation({
    mutationFn: () => archiveWorkspace(workspaceId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: workspaceKeys.all });
      await refreshWorkspaces();
      setAllTrips();
      router.push("/workspaces");
    }
  });

  function handleGeneralSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canManage) {
      return;
    }
    if (name.trim().length < 2 || name.trim().length > 80) {
      setFormError("Name must be between 2 and 80 characters.");
      return;
    }
    if (description.trim().length > 500) {
      setFormError("Description must be at most 500 characters.");
      return;
    }
    setFormError(null);
    updateMutation.mutate();
  }

  function handleInviteSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canInvite || !inviteEmail.trim()) {
      return;
    }
    inviteMutation.mutate();
  }

  function handleRemove(member: WorkspaceMember) {
    const label = member.displayName || member.email || "this member";
    if (window.confirm(`Remove ${label} from this workspace?`)) {
      removeMutation.mutate(member.id);
    }
  }

  function handleArchive() {
    if (window.confirm("Archive this workspace? Existing trips remain stored but the workspace is hidden.")) {
      archiveMutation.mutate();
    }
  }

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase text-primary-700">Workspace settings</p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">
            {workspace?.name ?? "Workspace"}
          </h1>
          {workspace ? (
            <p className="mt-3 text-sm text-slate-600">
              Your role: {formatWorkspaceRole(workspace.currentUserRole)}
            </p>
          ) : null}
        </div>
        <Link className={buttonStyles({ variant: "secondary" })} href={`/workspaces/${workspaceId}`}>
          Back to workspace
        </Link>
      </div>

      {workspaceQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {workspaceQuery.error instanceof Error
            ? workspaceQuery.error.message
            : "Could not load workspace."}
        </div>
      ) : null}

      {workspace ? (
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
          <div className="space-y-6">
            <Card>
              <h2 className="text-lg font-semibold text-slate-950">General</h2>
              <form className="mt-5 space-y-5" onSubmit={handleGeneralSubmit}>
                <label className="block">
                  <span className="text-sm font-medium text-slate-800">Name</span>
                  <span className="mt-2 block">
                    <Input
                      disabled={!canManage}
                      value={name}
                      maxLength={80}
                      onChange={(event) => setName(event.target.value)}
                    />
                  </span>
                </label>
                <label className="block">
                  <span className="text-sm font-medium text-slate-800">Description</span>
                  <span className="mt-2 block">
                    <Textarea
                      disabled={!canManage}
                      value={description}
                      maxLength={500}
                      onChange={(event) => setDescription(event.target.value)}
                    />
                  </span>
                </label>
                {formError || updateMutation.isError ? (
                  <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
                    {formError ?? getErrorMessage(updateMutation.error, "Could not update workspace.")}
                  </div>
                ) : null}
                {canManage ? (
                  <div className="flex justify-end">
                    <Button disabled={updateMutation.isPending} type="submit">
                      {updateMutation.isPending ? "Saving..." : "Save changes"}
                    </Button>
                  </div>
                ) : null}
              </form>
            </Card>

            <Card>
              <div className="flex flex-col gap-1">
                <h2 className="text-lg font-semibold text-slate-950">Members</h2>
                <p className="text-sm text-slate-600">
                  Workspace roles grant access to all trips in this workspace.
                </p>
              </div>

              {membersQuery.isPending ? (
                <p className="mt-5 text-sm text-slate-600">Loading members...</p>
              ) : null}
              {membersQuery.isError ? (
                <div className="mt-5 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
                  {membersQuery.error instanceof Error
                    ? membersQuery.error.message
                    : "Could not load members."}
                </div>
              ) : null}
              {membersQuery.isSuccess ? (
                <div className="mt-5 divide-y divide-slate-100 rounded-md border border-slate-200">
                  {membersQuery.data.map((member) => (
                    <MemberRow
                      key={member.id}
                      actorRole={workspace.currentUserRole}
                      currentUserId={user?.id}
                      disabled={roleMutation.isPending || removeMutation.isPending}
                      member={member}
                      onRemove={handleRemove}
                      onRoleChange={(role) => roleMutation.mutate({ memberId: member.id, role })}
                    />
                  ))}
                </div>
              ) : null}
            </Card>
          </div>

          <div className="space-y-6">
            {canInvite ? (
              <Card>
                <h2 className="text-lg font-semibold text-slate-950">Invite member</h2>
                <form className="mt-5 space-y-4" onSubmit={handleInviteSubmit}>
                  <label className="block">
                    <span className="text-sm font-medium text-slate-800">Email</span>
                    <span className="mt-2 block">
                      <Input
                        type="email"
                        value={inviteEmail}
                        placeholder="friend@example.com"
                        onChange={(event) => setInviteEmail(event.target.value)}
                      />
                    </span>
                  </label>
                  <label className="block">
                    <span className="text-sm font-medium text-slate-800">Role</span>
                    <span className="mt-2 block">
                      <Select
                        value={inviteRole}
                        onChange={(event) =>
                          setInviteRole(event.target.value as Exclude<WorkspaceRole, "owner">)
                        }
                      >
                        {inviteRoles
                          .filter((role) => workspace.currentUserRole === "owner" || role !== "admin")
                          .map((role) => (
                            <option key={role} value={role}>
                              {formatWorkspaceRole(role)}
                            </option>
                          ))}
                      </Select>
                    </span>
                  </label>
                  {inviteMutation.isError ? (
                    <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
                      {getErrorMessage(inviteMutation.error, "Could not invite member.")}
                    </div>
                  ) : null}
                  <Button disabled={inviteMutation.isPending || !inviteEmail.trim()} type="submit">
                    {inviteMutation.isPending ? "Inviting..." : "Send invitation"}
                  </Button>
                </form>
              </Card>
            ) : null}

            {canArchive ? (
              <Card className="border-red-200">
                <h2 className="text-lg font-semibold text-red-900">Danger zone</h2>
                <p className="mt-2 text-sm leading-6 text-slate-600">
                  Archive hides the workspace from normal lists and blocks new workspace trips.
                </p>
                {archiveMutation.isError ? (
                  <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
                    {getErrorMessage(archiveMutation.error, "Could not archive workspace.")}
                  </div>
                ) : null}
                <Button
                  className="mt-5"
                  disabled={archiveMutation.isPending}
                  variant="danger"
                  onClick={handleArchive}
                >
                  {archiveMutation.isPending ? "Archiving..." : "Archive workspace"}
                </Button>
              </Card>
            ) : null}
          </div>
        </div>
      ) : null}
    </PageContainer>
  );
}

function MemberRow({
  actorRole,
  currentUserId,
  disabled,
  member,
  onRemove,
  onRoleChange
}: {
  actorRole: WorkspaceRole;
  currentUserId?: string;
  disabled: boolean;
  member: WorkspaceMember;
  onRemove: (member: WorkspaceMember) => void;
  onRoleChange: (role: Exclude<WorkspaceRole, "owner">) => void;
}) {
  const canManage = canManageWorkspace(actorRole);
  const isSelf = currentUserId === member.userId;
  const canChangeRole =
    canManage &&
    member.role !== "owner" &&
    !(actorRole === "admin" && member.role === "admin");
  const canRemove =
    (isSelf || canManage) &&
    !(member.role === "owner" && !isSelf) &&
    !(actorRole === "admin" && (member.role === "admin" || member.role === "owner"));
  const roleOptions = inviteRoles.filter((role) => actorRole === "owner" || role !== "admin");

  return (
    <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="truncate text-sm font-semibold text-slate-950">
          {member.displayName || member.email || member.userId}
        </p>
        <p className="mt-1 text-xs text-slate-500">
          {member.email ? `${member.email} · ` : ""}
          {member.status}
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {canChangeRole ? (
          <Select
            className="h-9 w-32"
            disabled={disabled}
            value={member.role}
            onChange={(event) =>
              onRoleChange(event.target.value as Exclude<WorkspaceRole, "owner">)
            }
          >
            {roleOptions.map((role) => (
              <option key={role} value={role}>
                {formatWorkspaceRole(role)}
              </option>
            ))}
          </Select>
        ) : (
          <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700">
            {formatWorkspaceRole(member.role)}
          </span>
        )}
        {canRemove ? (
          <Button disabled={disabled} size="sm" variant="secondary" onClick={() => onRemove(member)}>
            {isSelf ? "Leave" : "Remove"}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

async function invalidateWorkspace(queryClient: ReturnType<typeof useQueryClient>, workspaceId: string) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: workspaceKeys.all }),
    queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(workspaceId) }),
    queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) })
  ]);
}
