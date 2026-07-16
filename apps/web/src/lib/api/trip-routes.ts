import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { TripRoute } from "@/entities/route/model";
import { approvalRiskKeys } from "@/lib/api/approval-risk";
import { budgetKeys } from "@/lib/api/budget";
import { budgetConfidenceKeys } from "@/lib/api/budget-confidence";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { getTripRoute, tripKeys, updateTripRoute } from "@/lib/api/trips";

export function useTripRoute(tripId: string, enabled = true) {
  return useQuery({
    queryKey: tripKeys.route(tripId),
    queryFn: () => getTripRoute(tripId),
    enabled: enabled && Boolean(tripId)
  });
}

export function useUpdateTripRoute(tripId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { expectedItineraryRevision?: number; route: TripRoute | null }) =>
      updateTripRoute(tripId, input),
    onSuccess: async (trip) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.route(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetConfidenceKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
      queryClient.setQueryData(tripKeys.detail(tripId), trip);
    }
  });
}
