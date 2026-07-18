import { apiFetch } from "@/shared/api/client";
import type { Budget, BudgetSummary } from "@/entities/budget/model";
import type { BudgetSummaryContract } from "@/lib/api/contracts";

export const budgetKeys = {
  summary: (tripId: string) => ["trips", "detail", tripId, "budget-summary"] as const
};

export function getTripBudgetSummary(tripId: string) {
  return apiFetch<BudgetSummary & BudgetSummaryContract>(`/trips/${tripId}/budget-summary`);
}

type BudgetEnvelope = {
  budget: Budget | null;
};

/**
 * updateTripBudget sets or clears the trip-level budget. Passing null clears it.
 * This does not mutate the itinerary revision.
 */
export async function updateTripBudget(
  tripId: string,
  budget: Budget | null
): Promise<Budget | null> {
  const body =
    budget == null
      ? { budget: null }
      : {
          budget: {
            amount: budget.amount,
            currency: budget.currency.trim().toUpperCase()
          }
        };

  const response = await apiFetch<BudgetEnvelope>(`/trips/${tripId}/budget`, {
    method: "PUT",
    body: JSON.stringify(body)
  });
  return response.budget;
}
