import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { approvalKeys } from "@/lib/api/approvals";
import {
  archiveWorkspacePolicy,
  getWorkspacePolicy,
  upsertWorkspacePolicy,
  workspacePolicyKeys
} from "@/lib/api/workspace-policies";
import type { UpsertWorkspacePolicyInput } from "@/types/workspace-policy";

export function useWorkspacePolicy(workspaceId: string) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: workspacePolicyKeys.workspace(workspaceId),
    queryFn: () => getWorkspacePolicy(workspaceId),
    enabled: Boolean(workspaceId)
  });

  function invalidate() {
    void queryClient.invalidateQueries({
      queryKey: workspacePolicyKeys.workspace(workspaceId)
    });
    void queryClient.invalidateQueries({ queryKey: workspacePolicyKeys.evaluations() });
    void queryClient.invalidateQueries({ queryKey: approvalKeys.all });
    void queryClient.invalidateQueries({ queryKey: ["trips"] });
  }

  const upsert = useMutation({
    mutationFn: (input: UpsertWorkspacePolicyInput) =>
      upsertWorkspacePolicy(workspaceId, input),
    onSuccess: invalidate
  });
  const archive = useMutation({
    mutationFn: () => archiveWorkspacePolicy(workspaceId),
    onSuccess: invalidate
  });

  return { query, upsert, archive };
}
