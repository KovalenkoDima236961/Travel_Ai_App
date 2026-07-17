import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { TripRoute } from "@/entities/route/model";
import { approvalRiskKeys } from "@/lib/api/approval-risk";
import { approvalKeys } from "@/lib/api/approvals";
import { activityKeys } from "@/lib/api/activity";
import { budgetKeys } from "@/lib/api/budget";
import { budgetConfidenceKeys } from "@/lib/api/budget-confidence";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { groupReadinessKeys } from "@/lib/api/group-readiness";
import { planningConstraintKeys } from "@/lib/api/planning-constraints";
import { reminderKeys } from "@/lib/api/trip-reminders";
import { getTripRoute, tripKeys, updateTripRoute } from "@/lib/api/trips";
import { queryKeys } from "@/lib/query-keys";

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
        queryClient.invalidateQueries({ queryKey: approvalKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.commandCenter(tripId) }),
        queryClient.invalidateQueries({ queryKey: reminderKeys.all }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: groupReadinessKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: planningConstraintKeys.all }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
      queryClient.setQueryData(tripKeys.detail(tripId), trip);
    }
  });
}
