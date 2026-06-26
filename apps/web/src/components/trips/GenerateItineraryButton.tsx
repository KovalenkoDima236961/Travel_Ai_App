"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { isItineraryConflictError } from "@/lib/api/client";
import { createGenerationJob, generationJobKeys } from "@/lib/api/generation-jobs";
import { tripKeys } from "@/lib/api/trips";
import { getErrorMessage } from "@/lib/utils";
import type { GenerationJob } from "@/types/generation-jobs";

type GenerateItineraryButtonProps = {
  tripId: string;
  itineraryRevision: number;
  disabled?: boolean;
  onJobCreated?: (job: GenerationJob) => void;
};

export function GenerateItineraryButton({
  tripId,
  itineraryRevision,
  disabled = false,
  onJobCreated
}: GenerateItineraryButtonProps) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      createGenerationJob(tripId, {
        jobType: "full_generation",
        expectedItineraryRevision: itineraryRevision
      }),
    onSuccess: async (job) => {
      onJobCreated?.(job);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
    },
    onError: async (error) => {
      if (isItineraryConflictError(error)) {
        await queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) });
      }
    }
  });

  return (
    <div className="flex flex-col items-start gap-2 sm:items-end">
      <Button disabled={disabled || mutation.isPending} onClick={() => mutation.mutate()}>
        {mutation.isPending ? "Queueing..." : "Generate itinerary"}
      </Button>
      <p className="max-w-xs text-left text-xs leading-5 text-slate-500 sm:text-right">
        Your saved travel preferences will be used when generating this itinerary.
      </p>
      {mutation.isError ? (
        <p className="max-w-xs text-sm text-red-700" role="alert">
          {isItineraryConflictError(mutation.error)
            ? "This itinerary changed. Reload latest version before trying again."
            : getErrorMessage(mutation.error, "Could not generate itinerary.")}
        </p>
      ) : null}
    </div>
  );
}
