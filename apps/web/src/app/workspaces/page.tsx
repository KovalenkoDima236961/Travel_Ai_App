"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { Card } from "@/components/ui/Card";
import { buttonStyles } from "@/components/ui/Button";
import { formatWorkspaceRole } from "@/components/workspaces/WorkspaceProvider";
import { listWorkspaces, workspaceKeys } from "@/lib/api/workspaces";
import { formatDate } from "@/lib/utils";
import type { WorkspaceRole } from "@/types/workspace";

export default function WorkspacesPage() {
  return (
    <ProtectedRoute>
      <WorkspacesPageContent />
    </ProtectedRoute>
  );
}

function WorkspacesPageContent() {
  const workspacesQuery = useQuery({
    queryKey: workspaceKeys.list(),
    queryFn: listWorkspaces
  });

  return (
    <PageContainer>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase text-primary-700">Workspaces</p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">Workspaces</h1>
          <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
            Manage shared planning spaces for groups and teams.
          </p>
        </div>
        <Link className={buttonStyles()} href="/workspaces/new">
          Create workspace
        </Link>
      </div>

      {workspacesQuery.isPending ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading workspaces...
        </div>
      ) : null}

      {workspacesQuery.isError ? (
        <div className="mt-8 rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {workspacesQuery.error instanceof Error
            ? workspacesQuery.error.message
            : "Could not load workspaces."}
        </div>
      ) : null}

      {workspacesQuery.isSuccess && workspacesQuery.data.length === 0 ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-8 text-center">
          <h2 className="text-lg font-semibold text-slate-950">No workspaces yet</h2>
          <p className="mt-2 text-sm text-slate-600">
            Create a workspace when a group needs access to more than one trip.
          </p>
          <Link className={buttonStyles({ className: "mt-5" })} href="/workspaces/new">
            Create workspace
          </Link>
        </div>
      ) : null}

      {workspacesQuery.isSuccess && workspacesQuery.data.length > 0 ? (
        <div className="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {workspacesQuery.data.map((workspace) => (
            <Link key={workspace.id} className="block h-full" href={`/workspaces/${workspace.id}`}>
              <Card className="flex h-full flex-col gap-5 transition hover:-translate-y-0.5 hover:border-primary-100 hover:shadow-lg">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="truncate text-lg font-semibold text-slate-950">
                      {workspace.name}
                    </h2>
                    <p className="mt-1 text-sm text-slate-500">
                      Created {formatDate(workspace.createdAt)}
                    </p>
                  </div>
                  <RoleBadge role={workspace.currentUserRole} />
                </div>
                {workspace.description ? (
                  <p className="line-clamp-3 text-sm leading-6 text-slate-600">
                    {workspace.description}
                  </p>
                ) : (
                  <p className="text-sm text-slate-500">No description</p>
                )}
                <div className="mt-auto text-sm font-medium text-slate-700">
                  {workspace.memberCount} {workspace.memberCount === 1 ? "member" : "members"}
                </div>
              </Card>
            </Link>
          ))}
        </div>
      ) : null}
    </PageContainer>
  );
}

function RoleBadge({ role }: { role: WorkspaceRole }) {
  return (
    <span className="shrink-0 rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700">
      {formatWorkspaceRole(role)}
    </span>
  );
}
