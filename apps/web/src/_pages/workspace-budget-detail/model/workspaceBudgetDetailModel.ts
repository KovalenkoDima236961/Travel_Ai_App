import { formatAnalyticsDate } from "@/components/analytics/format";

export function formatBudgetPeriod(budget: { periodStart?: string | null; periodEnd?: string | null }) {
  if (!budget.periodStart && !budget.periodEnd) {
    return "All trips";
  }
  return `${formatAnalyticsDate(budget.periodStart)} - ${formatAnalyticsDate(budget.periodEnd)}`;
}

export function mutationMessage(error: unknown) {
  return error instanceof Error ? error.message : null;
}
