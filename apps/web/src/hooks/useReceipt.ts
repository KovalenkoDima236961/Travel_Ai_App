import { useQuery } from "@tanstack/react-query";
import { getReceipt, receiptKeys } from "@/lib/api/receipts";

export function useReceipt({
  tripId,
  receiptId,
  enabled = true
}: {
  tripId: string;
  receiptId: string;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: receiptKeys.detail(tripId, receiptId),
    queryFn: () => getReceipt(tripId, receiptId),
    enabled: enabled && Boolean(tripId) && Boolean(receiptId)
  });
}
