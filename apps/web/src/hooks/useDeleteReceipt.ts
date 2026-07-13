import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { expenseKeys } from "@/lib/api/expenses";
import { deleteReceipt, receiptKeys } from "@/lib/api/receipts";

export function useDeleteReceipt(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (receiptId: string) => deleteReceipt(tripId, receiptId),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: receiptKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: expenseKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ])
  });
}
