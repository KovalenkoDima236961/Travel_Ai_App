import { apiFetch } from "@/shared/api/client";
import type { BudgetConfidence } from "@/types/budget-confidence";

type BudgetConfidenceParams = {
  currency?: string | null;
  includeDebug?: boolean;
};

export const budgetConfidenceKeys = {
  all: (tripId: string) => ["trips", "detail", tripId, "budget-confidence"] as const,
  detail: (tripId: string, currency?: string | null) =>
    [...budgetConfidenceKeys.all(tripId), normalizeCurrency(currency)] as const
};

export function getTripBudgetConfidence(tripId: string, params: BudgetConfidenceParams = {}) {
  const query = new URLSearchParams();
  const currency = normalizeCurrency(params.currency);
  if (currency) {
    query.set("currency", currency);
  }
  if (params.includeDebug) {
    query.set("includeDebug", "true");
  }
  const suffix = query.toString();
  return apiFetch<BudgetConfidence>(
    `/trips/${tripId}/budget-confidence${suffix ? `?${suffix}` : ""}`
  );
}

function normalizeCurrency(currency?: string | null) {
  const normalized = currency?.trim().toUpperCase();
  return normalized || null;
}
