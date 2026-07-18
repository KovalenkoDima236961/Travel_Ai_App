"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getTripVerification,
  runVerificationAction,
  verificationKeys
} from "@/lib/api/verification";
import { queryKeys } from "@/lib/query-keys";
import type { RunVerificationActionInput } from "@/types/verification";

export function useTripVerification(tripId: string, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: verificationKeys.detail(tripId),
    queryFn: () => getTripVerification(tripId),
    enabled: (options.enabled ?? true) && Boolean(tripId),
    staleTime: 45 * 1000
  });
}

export function useRunVerificationAction(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: RunVerificationActionInput) => runVerificationAction(tripId, input),
    onSuccess: async (result) => {
      queryClient.setQueryData(verificationKeys.detail(tripId), result.updatedVerification);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.health(tripId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.commandCenter(tripId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.budgetConfidence(tripId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.approval(tripId) })
      ]);
    }
  });
}
