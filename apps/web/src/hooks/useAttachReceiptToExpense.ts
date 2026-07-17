import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { budgetConfidenceKeys } from "@/lib/api/budget-confidence";
import { expenseKeys } from "@/lib/api/expenses";
import { attachReceiptToExpense, receiptKeys } from "@/lib/api/receipts";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { queryKeys } from "@/lib/query-keys";

export function useAttachReceiptToExpense(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ expenseId, receiptId }: { expenseId: string; receiptId: string }) =>
      attachReceiptToExpense(tripId, expenseId, receiptId),
    onSuccess: (_, variables) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: receiptKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: receiptKeys.detail(tripId, variables.receiptId) }),
        queryClient.invalidateQueries({ queryKey: expenseKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetConfidenceKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.commandCenter(tripId) })
      ])
  });
}
