import { useQuery } from "@tanstack/react-query";

import { approvalKeys, listWorkspaceApprovals } from "@/lib/api/approvals";
import type { WorkspaceApprovalStatusFilter } from "@/types/approval";

// useWorkspaceApprovals loads the workspace approvals queue for a status tab.
// It refetches on window focus and on a light interval so a reviewer sees new
// submissions without a manual reload, without hammering the backend.
export function useWorkspaceApprovals({
  workspaceId,
  status,
  enabled = true
}: {
  workspaceId: string;
  status?: WorkspaceApprovalStatusFilter;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: approvalKeys.workspace(workspaceId, status),
    queryFn: () => listWorkspaceApprovals(workspaceId, { status }),
    enabled: enabled && Boolean(workspaceId),
    refetchOnWindowFocus: true,
    refetchInterval: 60_000
  });
}
