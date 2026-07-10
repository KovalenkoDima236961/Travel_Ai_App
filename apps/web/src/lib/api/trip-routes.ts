import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { TripRoute } from "@/entities/route/model";
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
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
      queryClient.setQueryData(tripKeys.detail(tripId), trip);
    }
  });
}
