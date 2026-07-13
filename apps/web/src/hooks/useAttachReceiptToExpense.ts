import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { expenseKeys } from "@/lib/api/expenses";
import { attachReceiptToExpense, receiptKeys } from "@/lib/api/receipts";

export function useAttachReceiptToExpense(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ expenseId, receiptId }: { expenseId: string; receiptId: string }) =>
      attachReceiptToExpense(tripId, expenseId, receiptId),
    onSuccess: (_, variables) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: receiptKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: receiptKeys.detail(tripId, variables.receiptId) }),
        queryClient.invalidateQueries({ queryKey: expenseKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ])
  });
}
