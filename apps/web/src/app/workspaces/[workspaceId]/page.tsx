"use client";

import Link from "next/link";
import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { TripCard } from "@/components/trips/TripCard";
import { Card } from "@/components/ui/Card";
import { buttonStyles } from "@/components/ui/Button";
import {
  canManageWorkspace,
  formatWorkspaceRole,
  useWorkspaces
} from "@/components/workspaces/WorkspaceProvider";
import { listTrips, tripKeys } from "@/lib/api/trips";
import {
  getWorkspace,
  listWorkspaceMembers,
  workspaceKeys
} from "@/lib/api/workspaces";
import { formatDate } from "@/lib/utils";

export default function WorkspaceDetailPage() {
  return (
    <ProtectedRoute>
      <WorkspaceDetailPageContent />
    </ProtectedRoute>
  );
}

function WorkspaceDetailPageContent() {
  const params = useParams<{ workspaceId: string }>();
  const workspaceId = params.workspaceId;
  const { setCurrentWorkspace } = useWorkspaces();

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
  const tripsQuery = useQuery({
    queryKey: tripKeys.list({ limit: 20, offset: 0, scope: "workspace", workspaceId }),
    queryFn: () => listTrips({ limit: 20, offset: 0, scope: "workspace", workspaceId }),
    enabled: Boolean(workspaceId)
  });

  useEffect(() => {
    if (workspaceQuery.isSuccess) {
      setCurrentWorkspace(workspaceId);
    }
  }, [setCurrentWorkspace, workspaceId, workspaceQuery.isSuccess]);

  const workspace = workspaceQuery.data;
  const canManage = workspace ? canManageWorkspace(workspace.currentUserRole) : false;

  return (
    <PageContainer>
      {workspaceQuery.isPending ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading workspace...
        </div>
      ) : null}

      {workspaceQuery.isError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {workspaceQuery.error instanceof Error
            ? workspaceQuery.error.message
            : "Could not load workspace."}
        </div>
      ) : null}

      {workspace ? (
        <>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="text-sm font-semibold uppercase text-primary-700">Workspace</p>
              <h1 className="mt-2 text-3xl font-semibold text-slate-950">{workspace.name}</h1>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
                {workspace.description || "No description"}
              </p>
              <div className="mt-4 flex flex-wrap gap-2 text-xs font-semibold">
                <span className="rounded-full bg-slate-100 px-2.5 py-1 text-slate-700">
                  {formatWorkspaceRole(workspace.currentUserRole)}
                </span>
                <span className="rounded-full bg-slate-100 px-2.5 py-1 text-slate-700">
                  {workspace.memberCount} {workspace.memberCount === 1 ? "member" : "members"}
                </span>
                <span className="rounded-full bg-slate-100 px-2.5 py-1 text-slate-700">
                  Created {formatDate(workspace.createdAt)}
                </span>
              </div>
            </div>
            <div className="flex flex-wrap gap-2">
              <Link className={buttonStyles({ variant: "secondary" })} href={`/workspaces/${workspace.id}/analytics`}>
                Analytics
              </Link>
              <Link className={buttonStyles({ variant: "secondary" })} href={`/workspaces/${workspace.id}/budgets`}>
                Budgets
              </Link>
              <Link className={buttonStyles({ variant: "secondary" })} href={`/workspaces/${workspace.id}/templates`}>
                Templates
              </Link>
              <Link className={buttonStyles({ variant: "secondary" })} href="/trips/new">
                Create trip
              </Link>
              {canManage ? (
                <Link className={buttonStyles()} href={`/workspaces/${workspace.id}/settings`}>
                  Settings
                </Link>
              ) : null}
            </div>
          </div>

          <section className="mt-8">
            <div className="flex items-center justify-between gap-4">
              <h2 className="text-xl font-semibold text-slate-950">Workspace trips</h2>
            </div>
            {tripsQuery.isPending ? (
              <div className="mt-4 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
                Loading workspace trips...
              </div>
            ) : null}
            {tripsQuery.isError ? (
              <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
                {tripsQuery.error instanceof Error
                  ? tripsQuery.error.message
                  : "Could not load workspace trips."}
              </div>
            ) : null}
            {tripsQuery.isSuccess && tripsQuery.data.items.length === 0 ? (
              <div className="mt-4 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
                No trips in this workspace yet.
              </div>
            ) : null}
            {tripsQuery.isSuccess && tripsQuery.data.items.length > 0 ? (
              <div className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {tripsQuery.data.items.map((trip) => (
                  <TripCard key={trip.id} trip={trip} />
                ))}
              </div>
            ) : null}
          </section>

          <section className="mt-10">
            <h2 className="text-xl font-semibold text-slate-950">Members</h2>
            {membersQuery.isPending ? (
              <div className="mt-4 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
                Loading members...
              </div>
            ) : null}
            {membersQuery.isSuccess ? (
              <Card className="mt-4 divide-y divide-slate-100 p-0">
                {membersQuery.data.slice(0, 6).map((member) => (
                  <div key={member.id} className="flex items-center justify-between gap-4 p-4">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-slate-900">
                        {member.displayName || member.email || member.userId}
                      </p>
                      <p className="mt-1 text-xs text-slate-500">{member.status}</p>
                    </div>
                    <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700">
                      {formatWorkspaceRole(member.role)}
                    </span>
                  </div>
                ))}
                {membersQuery.data.length > 6 ? (
                  <div className="p-4 text-sm text-slate-500">
                    {membersQuery.data.length - 6} more members
                  </div>
                ) : null}
              </Card>
            ) : null}
          </section>
        </>
      ) : null}
    </PageContainer>
  );
}
