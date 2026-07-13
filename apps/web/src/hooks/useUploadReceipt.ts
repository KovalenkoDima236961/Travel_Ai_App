import { useMutation, useQueryClient } from "@tanstack/react-query";
import { activityKeys } from "@/lib/api/activity";
import { expenseKeys } from "@/lib/api/expenses";
import { receiptKeys, uploadReceipt } from "@/lib/api/receipts";
import type { ReceiptUploadInput } from "@/entities/receipt/model";

export function useUploadReceipt(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ReceiptUploadInput) => uploadReceipt(tripId, input),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: receiptKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: expenseKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ])
  });
}
