"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PageContainer } from "@/components/layout/PageContainer";
import { buttonStyles } from "@/shared/ui/button";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import {
  acceptWorkspaceInvitation,
  declineWorkspaceInvitation,
  listWorkspaceInvitations,
  workspaceKeys
} from "@/lib/api/workspaces";
import { getErrorMessage } from "@/lib/utils";
import { InvitationCard } from "./InvitationCard";

export function WorkspaceInvitationsPageContent() {
  const queryClient = useQueryClient();
  const { refreshWorkspaces, setCurrentWorkspace } = useWorkspaces();
  const invitationsQuery = useQuery({
    queryKey: workspaceKeys.invitations(),
    queryFn: listWorkspaceInvitations
  });

  const acceptMutation = useMutation({
    mutationFn: acceptWorkspaceInvitation,
    onSuccess: async (workspace) => {
      await queryClient.invalidateQueries({ queryKey: workspaceKeys.all });
      await refreshWorkspaces();
      setCurrentWorkspace(workspace.id);
    }
  });

  const declineMutation = useMutation({
    mutationFn: declineWorkspaceInvitation,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: workspaceKeys.invitations() });
      await refreshWorkspaces();
    }
  });

  return (
    <PageContainer>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase text-primary-700">
            Workspace invitations
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">Invitations</h1>
          <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
            Accept or decline pending workspace invitations.
          </p>
        </div>
        <Link className={buttonStyles({ variant: "secondary" })} href="/workspaces">
          Workspaces
        </Link>
      </div>

      {invitationsQuery.isPending ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading invitations...
        </div>
      ) : null}

      {invitationsQuery.isError ? (
        <div className="mt-8 rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {invitationsQuery.error instanceof Error
            ? invitationsQuery.error.message
            : "Could not load invitations."}
        </div>
      ) : null}

      {acceptMutation.isError || declineMutation.isError ? (
        <div className="mt-8 rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {getErrorMessage(
            acceptMutation.error ?? declineMutation.error,
            "Could not update invitation."
          )}
        </div>
      ) : null}

      {invitationsQuery.isSuccess && invitationsQuery.data.length === 0 ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-8 text-center">
          <h2 className="text-lg font-semibold text-slate-950">No pending invitations</h2>
          <p className="mt-2 text-sm text-slate-600">
            Workspace invitations sent to your email will appear here.
          </p>
        </div>
      ) : null}

      {invitationsQuery.isSuccess && invitationsQuery.data.length > 0 ? (
        <div className="mt-8 grid gap-4 md:grid-cols-2">
          {invitationsQuery.data.map((invitation) => (
            <InvitationCard
              key={invitation.id}
              invitation={invitation}
              pending={acceptMutation.isPending || declineMutation.isPending}
              onAccept={() => acceptMutation.mutate(invitation.id)}
              onDecline={() => declineMutation.mutate(invitation.id)}
            />
          ))}
        </div>
      ) : null}
    </PageContainer>
  );
}
