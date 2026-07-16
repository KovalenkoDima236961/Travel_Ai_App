import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { budgetConfidenceKeys } from "@/lib/api/budget-confidence";
import { expenseKeys } from "@/lib/api/expenses";
import { createExpenseFromReceipt, receiptKeys } from "@/lib/api/receipts";
import { tripHealthKeys } from "@/lib/api/trip-health";
import type { CreateExpenseInput } from "@/entities/expense/model";

export function useCreateExpenseFromReceipt(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ receiptId, input }: { receiptId: string; input: CreateExpenseInput }) =>
      createExpenseFromReceipt(tripId, receiptId, input),
    onSuccess: (_, variables) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: receiptKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: receiptKeys.detail(tripId, variables.receiptId) }),
        queryClient.invalidateQueries({ queryKey: expenseKeys.all }),
        queryClient.invalidateQueries({ queryKey: budgetConfidenceKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ])
  });
}
