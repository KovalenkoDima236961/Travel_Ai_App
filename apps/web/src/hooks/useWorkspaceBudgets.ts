import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { costAnalyticsKeys } from "@/lib/api/cost-analytics";
import {
  archiveWorkspaceBudget,
  createWorkspaceBudget,
  listWorkspaceBudgets,
  makeWorkspaceBudgetPrimary,
  updateWorkspaceBudget,
  workspaceBudgetKeys
} from "@/lib/api/workspace-budgets";
import type {
  CreateWorkspaceBudgetInput,
  UpdateWorkspaceBudgetInput,
  WorkspaceBudgetStatus
} from "@/types/workspace-budget";

export function useWorkspaceBudgets({
  workspaceId,
  status,
  enabled = true
}: {
  workspaceId: string;
  status?: WorkspaceBudgetStatus;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: workspaceBudgetKeys.list(workspaceId, status),
    queryFn: () => listWorkspaceBudgets(workspaceId, status),
    enabled: enabled && Boolean(workspaceId)
  });
}

export function useWorkspaceBudgetMutations(workspaceId: string) {
  const queryClient = useQueryClient();

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: workspaceBudgetKeys.all });
    void queryClient.invalidateQueries({ queryKey: costAnalyticsKeys.all });
  }

  return {
    createBudget: useMutation({
      mutationFn: (input: CreateWorkspaceBudgetInput) =>
        createWorkspaceBudget(workspaceId, input),
      onSuccess: invalidate
    }),
    updateBudget: useMutation({
      mutationFn: ({ budgetId, input }: { budgetId: string; input: UpdateWorkspaceBudgetInput }) =>
        updateWorkspaceBudget(workspaceId, budgetId, input),
      onSuccess: invalidate
    }),
    archiveBudget: useMutation({
      mutationFn: ({ budgetId, reason }: { budgetId: string; reason?: string }) =>
        archiveWorkspaceBudget(workspaceId, budgetId, reason),
      onSuccess: invalidate
    }),
    makePrimary: useMutation({
      mutationFn: (budgetId: string) => makeWorkspaceBudgetPrimary(workspaceId, budgetId),
      onSuccess: invalidate
    })
  };
}
