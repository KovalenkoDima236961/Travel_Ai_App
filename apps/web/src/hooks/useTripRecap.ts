"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  applyTripRecapLearning,
  archiveTripRecap,
  createTemplateFromTripRecap,
  finalizeTripRecap,
  generateTripRecap,
  getTripRecap,
  getTripRecapStatus,
  submitTripRecapFeedback,
  tripRecapKeys,
  updateTripRecap
} from "@/lib/api/recap";
import { tripKeys } from "@/lib/api/trips";
import type { LearningCandidate, TripRecapContent } from "@/types/recap";

export function useTripRecapStatus(tripId: string) {
  return useQuery({
    queryKey: tripRecapKeys.status(tripId),
    queryFn: () => getTripRecapStatus(tripId),
    enabled: Boolean(tripId),
    staleTime: 30_000
  });
}

export function useTripRecap(tripId: string, enabled = true) {
  return useQuery({
    queryKey: tripRecapKeys.detail(tripId),
    queryFn: () => getTripRecap(tripId),
    enabled: Boolean(tripId) && enabled
  });
}

export function useTripRecapMutations(tripId: string) {
  const queryClient = useQueryClient();
  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: tripRecapKeys.status(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripRecapKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) })
    ]);
  };

  return {
    generate: useMutation({ mutationFn: (forceRegenerate: boolean) => generateTripRecap(tripId, { forceRegenerate }), onSuccess: invalidate }),
    update: useMutation({ mutationFn: (recap: TripRecapContent) => updateTripRecap(tripId, recap), onSuccess: invalidate }),
    finalize: useMutation({ mutationFn: () => finalizeTripRecap(tripId), onSuccess: invalidate }),
    archive: useMutation({ mutationFn: () => archiveTripRecap(tripId), onSuccess: invalidate }),
    submitFeedback: useMutation({ mutationFn: (input: Parameters<typeof submitTripRecapFeedback>[1]) => submitTripRecapFeedback(tripId, input), onSuccess: invalidate }),
    applyLearning: useMutation({ mutationFn: (candidates: LearningCandidate[]) => applyTripRecapLearning(tripId, candidates), onSuccess: invalidate }),
    createTemplate: useMutation({ mutationFn: (input: Parameters<typeof createTemplateFromTripRecap>[1]) => createTemplateFromTripRecap(tripId, input), onSuccess: invalidate })
  };
}
