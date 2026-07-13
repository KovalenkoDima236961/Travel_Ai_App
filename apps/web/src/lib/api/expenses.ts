import { apiFetch } from "@/shared/api/client";
import type {
  CreateExpenseInput,
  ExpenseSummary,
  ListExpensesFilters,
  MarkSettlementPaidInput,
  SettlementsResponse,
  TripExpense,
  TripExpensesResponse,
  UpdateExpenseInput
} from "@/entities/expense/model";

export const expenseKeys = {
  all: ["expenses"] as const,
  list: (tripId: string, filters?: ListExpensesFilters) =>
    [...expenseKeys.all, "list", tripId, filtersKey(filters)] as const,
  summary: (tripId: string, currency?: string | null) =>
    [...expenseKeys.all, "summary", tripId, currency ?? null] as const,
  settlements: (tripId: string, currency?: string | null) =>
    [...expenseKeys.all, "settlements", tripId, currency ?? null] as const
};

export function listTripExpenses(tripId: string, filters?: ListExpensesFilters) {
  const query = expenseFilterParams(filters).toString();
  return apiFetch<TripExpensesResponse>(
    `/trips/${tripId}/expenses${query ? `?${query}` : ""}`
  );
}

export function getTripExpense(tripId: string, expenseId: string) {
  return apiFetch<TripExpense>(`/trips/${tripId}/expenses/${expenseId}`);
}

export function createTripExpense(tripId: string, input: CreateExpenseInput) {
  return apiFetch<TripExpense>(`/trips/${tripId}/expenses`, {
    method: "POST",
    body: JSON.stringify(cleanExpensePayload(input))
  });
}

export function updateTripExpense(
  tripId: string,
  expenseId: string,
  input: UpdateExpenseInput
) {
  return apiFetch<TripExpense>(`/trips/${tripId}/expenses/${expenseId}`, {
    method: "PATCH",
    body: JSON.stringify(cleanExpensePayload(input))
  });
}

export function deleteTripExpense(tripId: string, expenseId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/expenses/${expenseId}`, {
    method: "DELETE"
  });
}

export function getTripExpenseSummary(tripId: string, currency?: string | null) {
  const params = new URLSearchParams();
  if (currency) {
    params.set("currency", currency.trim().toUpperCase());
  }
  const query = params.toString();
  return apiFetch<ExpenseSummary>(
    `/trips/${tripId}/expenses/summary${query ? `?${query}` : ""}`
  );
}

export function getTripSettlements(tripId: string, currency?: string | null) {
  const params = new URLSearchParams();
  if (currency) {
    params.set("currency", currency.trim().toUpperCase());
  }
  const query = params.toString();
  return apiFetch<SettlementsResponse>(
    `/trips/${tripId}/settlements${query ? `?${query}` : ""}`
  );
}

export function recalculateTripSettlements(tripId: string, currency?: string | null) {
  const params = new URLSearchParams();
  if (currency) {
    params.set("currency", currency.trim().toUpperCase());
  }
  const query = params.toString();
  return apiFetch<SettlementsResponse>(
    `/trips/${tripId}/settlements/recalculate${query ? `?${query}` : ""}`,
    { method: "POST" }
  );
}

export function markSettlementPaid(
  tripId: string,
  settlementId: string,
  input: MarkSettlementPaidInput = {}
) {
  return apiFetch<SettlementsResponse>(
    `/trips/${tripId}/settlements/${encodeURIComponent(settlementId)}/mark-paid`,
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );
}

export function cancelSettlement(tripId: string, settlementId: string) {
  return apiFetch<SettlementsResponse>(
    `/trips/${tripId}/settlements/${encodeURIComponent(settlementId)}/cancel`,
    { method: "POST" }
  );
}

function expenseFilterParams(filters?: ListExpensesFilters) {
  const params = new URLSearchParams();
  if (!filters) {
    return params;
  }
  if (filters.category) {
    params.set("category", filters.category);
  }
  if (filters.paidByUserId) {
    params.set("paidByUserId", filters.paidByUserId);
  }
  if (filters.fromDate) {
    params.set("fromDate", filters.fromDate);
  }
  if (filters.toDate) {
    params.set("toDate", filters.toDate);
  }
  if (filters.linkedOnly) {
    params.set("linkedOnly", "true");
  }
  return params;
}

function filtersKey(filters?: ListExpensesFilters) {
  return {
    category: filters?.category ?? null,
    paidByUserId: filters?.paidByUserId ?? null,
    fromDate: filters?.fromDate ?? null,
    toDate: filters?.toDate ?? null,
    linkedOnly: filters?.linkedOnly ?? false
  };
}

function cleanExpensePayload(input: CreateExpenseInput | UpdateExpenseInput) {
  return {
    ...input,
    ...(input.title != null ? { title: input.title.trim() } : {}),
    ...(input.description != null ? { description: input.description.trim() } : {}),
    ...(input.notes != null ? { notes: input.notes.trim() } : {}),
    ...(input.amount
      ? {
          amount: {
            amount: input.amount.amount,
            currency: input.amount.currency.trim().toUpperCase()
          }
        }
      : {})
  };
}
