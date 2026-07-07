import { useQuery } from "@tanstack/react-query";
import {
  getPrimaryWorkspaceBudgetSummary,
  workspaceBudgetKeys
} from "@/lib/api/workspace-budgets";

export function usePrimaryWorkspaceBudgetSummary({
  workspaceId,
  enabled = true
}: {
  workspaceId: string;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: workspaceBudgetKeys.primarySummary(workspaceId),
    queryFn: () => getPrimaryWorkspaceBudgetSummary(workspaceId),
    enabled: enabled && Boolean(workspaceId),
    retry: false
  });
}
