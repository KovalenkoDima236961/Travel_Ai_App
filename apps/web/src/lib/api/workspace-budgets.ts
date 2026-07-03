import { apiFetch } from "@/lib/api/client";
import type {
  CreateWorkspaceBudgetInput,
  UpdateWorkspaceBudgetInput,
  WorkspaceBudget,
  WorkspaceBudgetEnvelope,
  WorkspaceBudgetsEnvelope,
  WorkspaceBudgetStatus,
  WorkspaceBudgetSummary
} from "@/types/workspace-budget";

export const workspaceBudgetKeys = {
  all: ["workspace-budgets"] as const,
  lists: () => [...workspaceBudgetKeys.all, "list"] as const,
  list: (workspaceId: string, status?: WorkspaceBudgetStatus) =>
    [...workspaceBudgetKeys.lists(), workspaceId, status ?? "all"] as const,
  details: () => [...workspaceBudgetKeys.all, "detail"] as const,
  detail: (workspaceId: string, budgetId: string) =>
    [...workspaceBudgetKeys.details(), workspaceId, budgetId] as const,
  summary: (workspaceId: string, budgetId: string) =>
    [...workspaceBudgetKeys.detail(workspaceId, budgetId), "summary"] as const,
  primarySummary: (workspaceId: string) =>
    [...workspaceBudgetKeys.all, "primary-summary", workspaceId] as const
};

export async function listWorkspaceBudgets(
  workspaceId: string,
  status?: WorkspaceBudgetStatus
): Promise<WorkspaceBudget[]> {
  const searchParams = new URLSearchParams();
  if (status) {
    searchParams.set("status", status);
  }
  const query = searchParams.toString();
  const response = await apiFetch<WorkspaceBudgetsEnvelope>(
    `/workspaces/${workspaceId}/budgets${query ? `?${query}` : ""}`
  );
  return response.budgets;
}

export async function createWorkspaceBudget(
  workspaceId: string,
  input: CreateWorkspaceBudgetInput
): Promise<WorkspaceBudget> {
  const response = await apiFetch<WorkspaceBudgetEnvelope>(
    `/workspaces/${workspaceId}/budgets`,
    {
      method: "POST",
      body: JSON.stringify(cleanBudgetPayload(input))
    }
  );
  return response.budget;
}

export async function getWorkspaceBudget(
  workspaceId: string,
  budgetId: string
): Promise<WorkspaceBudget> {
  const response = await apiFetch<WorkspaceBudgetEnvelope>(
    `/workspaces/${workspaceId}/budgets/${budgetId}`
  );
  return response.budget;
}

export async function updateWorkspaceBudget(
  workspaceId: string,
  budgetId: string,
  input: UpdateWorkspaceBudgetInput
): Promise<WorkspaceBudget> {
  const response = await apiFetch<WorkspaceBudgetEnvelope>(
    `/workspaces/${workspaceId}/budgets/${budgetId}`,
    {
      method: "PATCH",
      body: JSON.stringify(cleanBudgetPayload(input))
    }
  );
  return response.budget;
}

export async function archiveWorkspaceBudget(
  workspaceId: string,
  budgetId: string,
  reason?: string
): Promise<WorkspaceBudget> {
  const response = await apiFetch<WorkspaceBudgetEnvelope>(
    `/workspaces/${workspaceId}/budgets/${budgetId}/archive`,
    {
      method: "POST",
      body: JSON.stringify({ reason: reason?.trim() || undefined })
    }
  );
  return response.budget;
}

export async function makeWorkspaceBudgetPrimary(
  workspaceId: string,
  budgetId: string
): Promise<WorkspaceBudget> {
  const response = await apiFetch<WorkspaceBudgetEnvelope>(
    `/workspaces/${workspaceId}/budgets/${budgetId}/make-primary`,
    { method: "POST" }
  );
  return response.budget;
}

export function getWorkspaceBudgetSummary(workspaceId: string, budgetId: string) {
  return apiFetch<WorkspaceBudgetSummary>(
    `/workspaces/${workspaceId}/budgets/${budgetId}/summary`
  );
}

export function getPrimaryWorkspaceBudgetSummary(workspaceId: string) {
  return apiFetch<WorkspaceBudgetSummary>(
    `/workspaces/${workspaceId}/budgets/primary/summary`
  );
}

function cleanBudgetPayload(input: UpdateWorkspaceBudgetInput | CreateWorkspaceBudgetInput) {
  return {
    ...("name" in input && input.name != null ? { name: input.name.trim() } : {}),
    ...("description" in input
      ? { description: input.description?.trim() ? input.description.trim() : null }
      : {}),
    ...("amount" in input && input.amount != null ? { amount: input.amount } : {}),
    ...("currency" in input && input.currency != null
      ? { currency: input.currency.trim().toUpperCase() }
      : {}),
    ...("periodStart" in input ? { periodStart: input.periodStart || null } : {}),
    ...("periodEnd" in input ? { periodEnd: input.periodEnd || null } : {}),
    ...("isPrimary" in input && input.isPrimary != null ? { isPrimary: input.isPrimary } : {})
  };
}
