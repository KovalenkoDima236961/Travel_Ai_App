import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  cancelSettlement,
  createTripExpense,
  deleteTripExpense,
  expenseKeys,
  getTripExpenseSummary,
  getTripSettlements,
  listTripExpenses,
  markSettlementPaid,
  recalculateTripSettlements,
  updateTripExpense
} from "@/lib/api/expenses";
import { activityKeys } from "@/lib/api/activity";
import { tripHealthKeys } from "@/lib/api/trip-health";
import type {
  CreateExpenseInput,
  ListExpensesFilters,
  MarkSettlementPaidInput,
  UpdateExpenseInput
} from "@/entities/expense/model";

export function useTripExpenses({
  tripId,
  filters,
  enabled = true
}: {
  tripId: string;
  filters?: ListExpensesFilters;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: expenseKeys.list(tripId, filters),
    queryFn: () => listTripExpenses(tripId, filters),
    enabled: enabled && Boolean(tripId)
  });
}

export function useTripExpenseSummary({
  tripId,
  currency,
  enabled = true
}: {
  tripId: string;
  currency?: string | null;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: expenseKeys.summary(tripId, currency),
    queryFn: () => getTripExpenseSummary(tripId, currency),
    enabled: enabled && Boolean(tripId)
  });
}

export function useTripSettlements({
  tripId,
  currency,
  enabled = true
}: {
  tripId: string;
  currency?: string | null;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: expenseKeys.settlements(tripId, currency),
    queryFn: () => getTripSettlements(tripId, currency),
    enabled: enabled && Boolean(tripId)
  });
}

export function useCreateTripExpense(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateExpenseInput) => createTripExpense(tripId, input),
    onSuccess: () => invalidateExpenseQueries(queryClient, tripId)
  });
}

export function useUpdateTripExpense(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ expenseId, input }: { expenseId: string; input: UpdateExpenseInput }) =>
      updateTripExpense(tripId, expenseId, input),
    onSuccess: () => invalidateExpenseQueries(queryClient, tripId)
  });
}

export function useDeleteTripExpense(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (expenseId: string) => deleteTripExpense(tripId, expenseId),
    onSuccess: () => invalidateExpenseQueries(queryClient, tripId)
  });
}

export function useMarkSettlementPaid(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      settlementId,
      input
    }: {
      settlementId: string;
      input?: MarkSettlementPaidInput;
    }) => markSettlementPaid(tripId, settlementId, input),
    onSuccess: () => invalidateExpenseQueries(queryClient, tripId)
  });
}

export function useCancelSettlement(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (settlementId: string) => cancelSettlement(tripId, settlementId),
    onSuccess: () => invalidateExpenseQueries(queryClient, tripId)
  });
}

export function useRecalculateTripSettlements(tripId: string, currency?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => recalculateTripSettlements(tripId, currency),
    onSuccess: () => invalidateExpenseQueries(queryClient, tripId)
  });
}

function invalidateExpenseQueries(queryClient: QueryClient, tripId: string) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: expenseKeys.all }),
    queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
    queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
  ]);
}
