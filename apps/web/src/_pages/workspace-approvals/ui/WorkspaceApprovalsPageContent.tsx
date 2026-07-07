"use client";

import Link from "next/link";
import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";

import { WorkspaceApprovalsQueue } from "@/features/trip-approval";
import { PageContainer } from "@/components/layout/PageContainer";
import { canManageWorkspace, useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";

export function WorkspaceApprovalsPageContent() {
  const params = useParams<{ workspaceId: string }>();
  const workspaceId = params.workspaceId;
  const { setCurrentWorkspace } = useWorkspaces();

  const workspaceQuery = useQuery({
    queryKey: workspaceKeys.detail(workspaceId),
    queryFn: () => getWorkspace(workspaceId),
    enabled: Boolean(workspaceId)
  });

  useEffect(() => {
    if (workspaceQuery.isSuccess) {
      setCurrentWorkspace(workspaceId);
    }
  }, [setCurrentWorkspace, workspaceId, workspaceQuery.isSuccess]);

  const workspace = workspaceQuery.data ?? null;
  const canManage = workspace ? canManageWorkspace(workspace.currentUserRole) : false;

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link
            className="text-sm font-medium text-primary-700 hover:text-primary-600"
            href={`/workspaces/${workspaceId}`}
          >
            Back to workspace
          </Link>
          <h1 className="mt-3 text-3xl font-semibold text-slate-950">Approvals</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
            Review trips submitted for approval. Owners and admins can approve or request changes.
          </p>
        </div>
      </div>

      {workspaceQuery.isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading workspace…
        </div>
      ) : (
        <WorkspaceApprovalsQueue workspaceId={workspaceId} canManage={canManage} />
      )}
    </PageContainer>
  );
}
