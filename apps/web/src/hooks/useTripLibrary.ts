"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { archiveTrip, getTripLibrary, getTripLibraryInsights, restoreTrip, tripLibraryKeys } from "@/lib/api/library";
import { tripKeys } from "@/lib/api/trips";
import type { ArchiveTripInput, TripLibraryFilters } from "@/types/library";

export function useTripLibrary(filters: TripLibraryFilters) {
  return useQuery({ queryKey: tripLibraryKeys.list(filters), queryFn: () => getTripLibrary(filters) });
}

export function useTripLibraryInsights(params: { workspaceId?: string; year?: number }) {
  return useQuery({ queryKey: tripLibraryKeys.insights(params), queryFn: () => getTripLibraryInsights(params) });
}

export function useArchiveTrip() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tripId, input }: { tripId: string; input?: ArchiveTripInput }) => archiveTrip(tripId, input),
    onSuccess: async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: tripLibraryKeys.all }), queryClient.invalidateQueries({ queryKey: tripKeys.lists() })]); }
  });
}

export function useRestoreTrip() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tripId }: { tripId: string }) => restoreTrip(tripId),
    onSuccess: async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: tripLibraryKeys.all }), queryClient.invalidateQueries({ queryKey: tripKeys.lists() })]); }
  });
}

// The page owns URL/local form state in v1; this small helper makes its defaults
// reusable without silently persisting potentially sensitive search text.
export function useTripLibraryFilters(initial: TripLibraryFilters = {}) {
  return { lifecycle: "all" as const, sort: "recently_updated" as const, ...initial };
}
