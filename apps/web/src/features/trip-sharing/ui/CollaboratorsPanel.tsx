"use client";

import { FormEvent, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import {
  createTripInvitation,
  listTripCollaborators,
  listTripInvitations,
  listTripMembers,
  removeTripCollaborator,
  resendTripInvitation,
  revokeTripInvitation,
  transferTripOwnership,
  tripKeys,
  updateTripCollaboratorRole
} from "@/lib/api/trips";
import { activityKeys } from "@/lib/api/activity";
import { formatDate, getErrorMessage } from "@/lib/utils";
import type {
  CollaboratorRole,
  TripCollaborator,
  TripInvitation,
  TripMember
} from "@/entities/collaboration/model";

type CollaboratorsPanelProps = {
  tripId: string;
  canManageCollaborators: boolean;
};

export function CollaboratorsPanel({
  tripId,
  canManageCollaborators
}: CollaboratorsPanelProps) {
  const queryClient = useQueryClient();
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<CollaboratorRole>("viewer");
  const [inviteMessage, setInviteMessage] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const collaboratorsQuery = useQuery({
    queryKey: tripKeys.collaborators(tripId),
    queryFn: () => listTripCollaborators(tripId),
    enabled: canManageCollaborators && Boolean(tripId)
  });

  const invitationsQuery = useQuery({
    queryKey: tripKeys.tripInvitations(tripId),
    queryFn: () => listTripInvitations(tripId),
    enabled: canManageCollaborators && Boolean(tripId)
  });

  const membersQuery = useQuery({
    queryKey: tripKeys.members(tripId),
    queryFn: () => listTripMembers(tripId),
    enabled: canManageCollaborators && Boolean(tripId)
  });

  async function refreshCollaboration() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: tripKeys.collaborators(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.tripInvitations(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.members(tripId) }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
    ]);
  }

  const inviteMutation = useMutation({
    mutationFn: () => createTripInvitation(tripId, { email, role, message: inviteMessage }),
    onSuccess: async () => {
      setEmail("");
      setRole("viewer");
      setInviteMessage("");
      setMessage("Invitation saved.");
      setError(null);
      await refreshCollaboration();
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not invite collaborator."));
      setMessage(null);
    }
  });

  const roleMutation = useMutation({
    mutationFn: ({ collaboratorId, nextRole }: { collaboratorId: string; nextRole: CollaboratorRole }) =>
      updateTripCollaboratorRole(tripId, collaboratorId, { role: nextRole }),
    onSuccess: async () => {
      setMessage("Collaborator role updated.");
      setError(null);
      await refreshCollaboration();
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not update collaborator."));
      setMessage(null);
    }
  });

  const removeMutation = useMutation({
    mutationFn: (collaboratorId: string) => removeTripCollaborator(tripId, collaboratorId),
    onSuccess: async () => {
      setMessage("Collaborator removed.");
      setError(null);
      await refreshCollaboration();
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not remove collaborator."));
      setMessage(null);
    }
  });

  const resendMutation = useMutation({
    mutationFn: (invitationId: string) => resendTripInvitation(tripId, invitationId),
    onSuccess: async () => {
      setMessage("Invitation resent.");
      setError(null);
      await refreshCollaboration();
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not resend invitation."));
      setMessage(null);
    }
  });

  const revokeMutation = useMutation({
    mutationFn: (invitationId: string) => revokeTripInvitation(tripId, invitationId),
    onSuccess: async () => {
      setMessage("Invitation revoked.");
      setError(null);
      await refreshCollaboration();
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not revoke invitation."));
      setMessage(null);
    }
  });

  const transferMutation = useMutation({
    mutationFn: (newOwnerUserId: string) => transferTripOwnership(tripId, newOwnerUserId),
    onSuccess: async () => {
      setMessage("Trip ownership transferred.");
      setError(null);
      await Promise.all([
        refreshCollaboration(),
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) })
      ]);
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not transfer ownership."));
      setMessage(null);
    }
  });

  if (!canManageCollaborators) {
    return null;
  }

  const collaborators = collaboratorsQuery.data ?? [];
  const invitations = (invitationsQuery.data ?? []).filter(
    (invitation) => invitation.status === "pending"
  );
  const members = membersQuery.data ?? [];
  const busy =
    inviteMutation.isPending ||
    roleMutation.isPending ||
    removeMutation.isPending ||
    resendMutation.isPending ||
    revokeMutation.isPending ||
    transferMutation.isPending;

  function submitInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = email.trim();
    if (!trimmed || !trimmed.includes("@")) {
      setError("Enter an email address.");
      setMessage(null);
      return;
    }
    setError(null);
    inviteMutation.mutate();
  }

  function removeCollaborator(collaborator: TripCollaborator) {
    if (!window.confirm("Remove this collaborator from the trip?")) {
      return;
    }
    removeMutation.mutate(collaborator.id);
  }

  function revokeInvitation(invitation: TripInvitation) {
    if (!window.confirm("Revoke this pending invitation?")) {
      return;
    }
    revokeMutation.mutate(invitation.id);
  }

  function transferOwnership(member: TripMember) {
    if (!window.confirm("Transfer trip ownership to this member?")) {
      return;
    }
    transferMutation.mutate(member.userId);
  }

  return (
    <Card>
      <div className="flex flex-col gap-4">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Collaborators</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Invite people by email, review pending invites, and manage accepted members.
          </p>
        </div>

        {message ? (
          <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">
            {message}
          </div>
        ) : null}

        {error ? (
          <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {error}
          </div>
        ) : null}

        <form className="space-y-3" onSubmit={submitInvite}>
          <div>
            <label className="block text-sm font-medium text-slate-700" htmlFor="collaborator-email">
              Email
            </label>
            <Input
              autoComplete="email"
              disabled={busy}
              id="collaborator-email"
              onChange={(event) => setEmail(event.target.value)}
              placeholder="friend@example.com"
              type="email"
              value={email}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700" htmlFor="collaborator-message">
              Message
            </label>
            <Textarea
              disabled={busy}
              id="collaborator-message"
              maxLength={500}
              onChange={(event) => setInviteMessage(event.target.value)}
              placeholder="Optional note for the invite"
              value={inviteMessage}
            />
          </div>
          <div className="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
            <div>
              <label className="block text-sm font-medium text-slate-700" htmlFor="collaborator-role">
                Role
              </label>
              <Select
                disabled={busy}
                id="collaborator-role"
                onChange={(event) => setRole(event.target.value as CollaboratorRole)}
                value={role}
              >
                <option value="viewer">Viewer</option>
                <option value="editor">Editor</option>
              </Select>
            </div>
            <Button disabled={busy} type="submit">
              {inviteMutation.isPending ? "Inviting..." : "Invite"}
            </Button>
          </div>
        </form>

        {invitations.length > 0 ? (
          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-900">Pending invitations</h3>
            {invitations.map((invitation) => (
              <div className="rounded-lg border border-slate-200 bg-white p-3" key={invitation.id}>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-slate-950">{invitation.email}</p>
                    <p className="mt-1 text-xs text-slate-500">
                      {invitation.role}
                      {" · expires "}
                      {formatDate(invitation.expiresAt, { dateStyle: "medium" })}
                    </p>
                    {invitation.message ? (
                      <p className="mt-2 line-clamp-2 text-xs text-slate-600">{invitation.message}</p>
                    ) : null}
                  </div>
                  <div className="flex gap-2">
                    <Button
                      disabled={busy}
                      onClick={() => resendMutation.mutate(invitation.id)}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      Resend
                    </Button>
                    <Button
                      disabled={busy}
                      onClick={() => revokeInvitation(invitation)}
                      size="sm"
                      type="button"
                      variant="danger"
                    >
                      Revoke
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : null}

        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-slate-900">Collaborators</h3>
          {collaboratorsQuery.isPending ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              Loading collaborators...
            </div>
          ) : null}

          {collaboratorsQuery.isError ? (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {getErrorMessage(collaboratorsQuery.error, "Could not load collaborators.")}
            </div>
          ) : null}

          {!collaboratorsQuery.isPending && !collaboratorsQuery.isError && collaborators.length === 0 ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              No collaborators invited yet.
            </div>
          ) : null}

          {collaborators.map((collaborator) => (
            <div className="rounded-lg border border-slate-200 bg-white p-3" key={collaborator.id}>
              <div className="flex flex-col gap-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-slate-950">
                    {collaborator.displayName || collaborator.email || collaborator.userId}
                  </p>
                  <p className="mt-1 text-xs text-slate-500">
                    {collaborator.status}
                    {" · invited "}
                    {formatDate(collaborator.invitedAt, { dateStyle: "medium" })}
                    {collaborator.expiresAt
                      ? ` · expires ${formatDate(collaborator.expiresAt, { dateStyle: "medium" })}`
                      : ""}
                    {collaborator.acceptedAt
                      ? ` · accepted ${formatDate(collaborator.acceptedAt, { dateStyle: "medium" })}`
                      : ""}
                  </p>
                  {collaborator.message ? (
                    <p className="mt-2 line-clamp-2 text-xs text-slate-600">{collaborator.message}</p>
                  ) : null}
                </div>
                <div className="grid gap-2 sm:grid-cols-[1fr_auto]">
                  <Select
                    disabled={busy}
                    onChange={(event) =>
                      roleMutation.mutate({
                        collaboratorId: collaborator.id,
                        nextRole: event.target.value as CollaboratorRole
                      })
                    }
                    value={collaborator.role}
                  >
                    <option value="viewer">Viewer</option>
                    <option value="editor">Editor</option>
                  </Select>
                  <Button
                    disabled={busy}
                    onClick={() => removeCollaborator(collaborator)}
                    size="sm"
                    type="button"
                    variant="danger"
                  >
                    Remove
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>

        {members.length > 0 ? (
          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-900">Members</h3>
            {members.map((member) => (
              <div className="rounded-lg border border-slate-200 bg-white p-3" key={`${member.role}-${member.userId}`}>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-slate-950">
                      {member.displayName || member.email || member.userId}
                      {member.isSelf ? " (you)" : ""}
                    </p>
                    <p className="mt-1 text-xs text-slate-500">
                      {member.role} · {member.status}
                      {member.lastSeenAt
                        ? ` · seen ${formatDate(member.lastSeenAt, { dateStyle: "medium" })}`
                        : ""}
                    </p>
                  </div>
                  {member.role !== "owner" && member.status === "active" ? (
                    <Button
                      disabled={busy}
                      onClick={() => transferOwnership(member)}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      Make owner
                    </Button>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </div>
    </Card>
  );
}
