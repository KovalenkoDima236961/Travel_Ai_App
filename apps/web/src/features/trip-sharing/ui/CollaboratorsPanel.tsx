"use client";

import { FormEvent, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import {
  inviteTripCollaborator,
  listTripCollaborators,
  removeTripCollaborator,
  tripKeys,
  updateTripCollaboratorRole
} from "@/lib/api/trips";
import { activityKeys } from "@/lib/api/activity";
import { formatDate, getErrorMessage } from "@/lib/utils";
import type { CollaboratorRole, TripCollaborator } from "@/entities/collaboration/model";

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
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const collaboratorsQuery = useQuery({
    queryKey: tripKeys.collaborators(tripId),
    queryFn: () => listTripCollaborators(tripId),
    enabled: canManageCollaborators && Boolean(tripId)
  });

  const inviteMutation = useMutation({
    mutationFn: () => inviteTripCollaborator(tripId, { email, role }),
    onSuccess: async () => {
      setEmail("");
      setRole("viewer");
      setMessage("Invitation saved.");
      setError(null);
      await queryClient.invalidateQueries({ queryKey: tripKeys.collaborators(tripId) });
      await queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) });
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
      await queryClient.invalidateQueries({ queryKey: tripKeys.collaborators(tripId) });
      await queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) });
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
      await queryClient.invalidateQueries({ queryKey: tripKeys.collaborators(tripId) });
      await queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) });
    },
    onError: (err) => {
      setError(getErrorMessage(err, "Could not remove collaborator."));
      setMessage(null);
    }
  });

  if (!canManageCollaborators) {
    return null;
  }

  const collaborators = collaboratorsQuery.data ?? [];
  const busy = inviteMutation.isPending || roleMutation.isPending || removeMutation.isPending;

  function submitInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = email.trim();
    if (!trimmed || !trimmed.includes("@")) {
      setError("Enter a registered user email address.");
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

  return (
    <Card>
      <div className="flex flex-col gap-4">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Collaborators</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Invite registered users to view or edit this private trip.
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

        <div className="space-y-3">
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
            <div
              className="rounded-lg border border-slate-200 bg-white p-3"
              key={collaborator.id}
            >
              <div className="flex flex-col gap-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-slate-950">
                    {collaborator.displayName || collaborator.email || collaborator.userId}
                  </p>
                  <p className="mt-1 text-xs text-slate-500">
                    {collaborator.status}
                    {" · invited "}
                    {formatDate(collaborator.invitedAt, { dateStyle: "medium" })}
                    {collaborator.acceptedAt
                      ? ` · accepted ${formatDate(collaborator.acceptedAt, { dateStyle: "medium" })}`
                      : ""}
                  </p>
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
      </div>
    </Card>
  );
}
