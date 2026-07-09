"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createTripRepairJob, tripRepairKeys } from "@/lib/api/trip-repair";
import { generationJobKeys } from "@/lib/api/generation-jobs";
import type { CreateRepairJobInput } from "@/entities/trip-repair/model";

export function useCreateTripRepairJob(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateRepairJobInput) => createTripRepairJob(tripId, input),
    onSuccess: (job) => {
      queryClient.setQueryData(generationJobKeys.detail(tripId, job.id), job);
      void queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
      void queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) });
    }
  });
}
