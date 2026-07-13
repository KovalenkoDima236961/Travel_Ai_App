import { useQuery } from "@tanstack/react-query";
import { getTripReceipts, receiptKeys } from "@/lib/api/receipts";
import type { ListReceiptsParams } from "@/entities/receipt/model";

export function useTripReceipts({
  tripId,
  params,
  enabled = true
}: {
  tripId: string;
  params?: ListReceiptsParams;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: receiptKeys.list(tripId, params),
    queryFn: () => getTripReceipts(tripId, params),
    enabled: enabled && Boolean(tripId)
  });
}
