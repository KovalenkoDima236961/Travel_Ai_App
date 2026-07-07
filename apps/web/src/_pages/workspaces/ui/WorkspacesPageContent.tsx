"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { PageContainer } from "@/components/layout/PageContainer";
import { buttonStyles } from "@/shared/ui/button";
import { listWorkspaces, workspaceKeys } from "@/lib/api/workspaces";
import { WorkspaceCard } from "./WorkspaceCard";

export function WorkspacesPageContent() {
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
            <WorkspaceCard key={workspace.id} workspace={workspace} />
          ))}
        </div>
      ) : null}
    </PageContainer>
  );
}
