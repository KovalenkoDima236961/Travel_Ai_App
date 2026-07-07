"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/shared/ui/button";
import {
  acceptCollaborationInvitation,
  declineCollaborationInvitation,
  listCollaborationInvitations,
  tripKeys
} from "@/lib/api/trips";
import { formatDate, getErrorMessage } from "@/lib/utils";
import type { CollaborationInvitation } from "@/entities/collaboration/model";

export function CollaborationInvitationsPanel() {
  const queryClient = useQueryClient();
  const invitationsQuery = useQuery({
    queryKey: tripKeys.invitations(),
    queryFn: listCollaborationInvitations
  });

  const acceptMutation = useMutation({
    mutationFn: (invitation: CollaborationInvitation) =>
      acceptCollaborationInvitation(invitation.tripId, invitation.collaboratorId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.invitations() }),
        queryClient.invalidateQueries({ queryKey: tripKeys.shared() })
      ]);
    }
  });

  const declineMutation = useMutation({
    mutationFn: (invitation: CollaborationInvitation) =>
      declineCollaborationInvitation(invitation.tripId, invitation.collaboratorId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: tripKeys.invitations() });
    }
  });

  if (invitationsQuery.isPending) {
    return null;
  }

  if (invitationsQuery.isError) {
    return (
      <div className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
        {getErrorMessage(invitationsQuery.error, "Could not load collaboration invitations.")}
      </div>
    );
  }

  const invitations = invitationsQuery.data ?? [];
  if (invitations.length === 0) {
    return null;
  }

  const busy = acceptMutation.isPending || declineMutation.isPending;

  return (
    <section className="mt-6 rounded-lg border border-primary-200 bg-primary-50 p-5">
      <h2 className="text-lg font-semibold text-slate-950">Pending invitations</h2>
      <div className="mt-4 space-y-3">
        {invitations.map((invitation) => (
          <div
            className="rounded-lg border border-primary-100 bg-white p-4"
            key={invitation.collaboratorId}
          >
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="font-semibold text-slate-950">{invitation.destination}</p>
                <p className="mt-1 text-sm text-slate-600">
                  Invited as {invitation.role} on{" "}
                  {formatDate(invitation.invitedAt, { dateStyle: "medium" })}
                </p>
              </div>
              <div className="flex gap-2">
                <Button
                  disabled={busy}
                  onClick={() => acceptMutation.mutate(invitation)}
                  size="sm"
                  type="button"
                >
                  {acceptMutation.isPending ? "Accepting..." : "Accept"}
                </Button>
                <Button
                  disabled={busy}
                  onClick={() => declineMutation.mutate(invitation)}
                  size="sm"
                  type="button"
                  variant="secondary"
                >
                  Decline
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
