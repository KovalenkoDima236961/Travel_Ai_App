"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { ButtonSpinner, ErrorState } from "@/components/ui";
import { ContextualTip } from "@/components/onboarding/ContextualTip";
import { Button } from "@/shared/ui/button";
import { isItineraryConflictError } from "@/shared/api/client";
import { createGenerationJob, generationJobKeys } from "@/lib/api/generation-jobs";
import { tripKeys } from "@/lib/api/trips";
import type { GenerationJob } from "@/entities/generation-job/model";

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
  const tripsT = useTranslations("trips");
  const qualityT = useTranslations("generationQuality");
  const errorsT = useTranslations("errors");
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
      <div className="max-w-sm text-left"><ContextualTip tipId="ai_generation" /></div>
      <Button data-generate-itinerary disabled={disabled || mutation.isPending} onClick={() => mutation.mutate()}>
        {mutation.isPending ? <ButtonSpinner className="mr-2" /> : null}
        {mutation.isPending ? qualityT("queueing") : tripsT("generate")}
      </Button>
      <p className="max-w-xs text-left text-xs leading-5 text-slate-500 sm:text-right">
        {qualityT("preferencesUsed")}
      </p>
      {mutation.isError ? (
        <ErrorState
          className="max-w-sm text-left"
          compact
          description={
            isItineraryConflictError(mutation.error)
              ? errorsT("itineraryConflict")
              : errorsT("itineraryGenerationDescription")
          }
          retryAction={{ onRetry: () => mutation.mutate(), pending: mutation.isPending }}
          title={errorsT("itineraryGenerationTitle")}
        />
      ) : null}
    </div>
  );
}
