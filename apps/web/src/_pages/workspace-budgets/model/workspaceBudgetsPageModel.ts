import { formatAnalyticsDate } from "@/components/analytics/format";
import type { WorkspaceBudget } from "@/entities/workspace-budget/model";

export function formatBudgetPeriod(budget: WorkspaceBudget) {
  if (!budget.periodStart && !budget.periodEnd) {
    return "All trips";
  }
  return `${formatAnalyticsDate(budget.periodStart)} - ${formatAnalyticsDate(budget.periodEnd)}`;
}

export function mutationMessage(error: unknown) {
  return error instanceof Error ? error.message : null;
}
