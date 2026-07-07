import { useQuery } from "@tanstack/react-query";
import {
  getWorkspaceBudgetSummary,
  workspaceBudgetKeys
} from "@/lib/api/workspace-budgets";

export function useWorkspaceBudgetSummary({
  workspaceId,
  budgetId,
  enabled = true
}: {
  workspaceId: string;
  budgetId: string;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: workspaceBudgetKeys.summary(workspaceId, budgetId),
    queryFn: () => getWorkspaceBudgetSummary(workspaceId, budgetId),
    enabled: enabled && Boolean(workspaceId) && Boolean(budgetId)
  });
}
