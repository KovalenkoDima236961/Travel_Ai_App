"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  aiDatasetKeys,
  getAITrainingConsent,
  updateAITrainingConsent
} from "@/lib/api/ai-datasets";
import { getErrorMessage } from "@/lib/utils";
import { SaveNotice, SectionHeading, SettingsCard, Switch } from "./controls";

export function AIImprovementConsentSection() {
  const queryClient = useQueryClient();
  const consent = useQuery({
    queryKey: aiDatasetKeys.consent,
    queryFn: getAITrainingConsent
  });
  const mutation = useMutation({
    mutationFn: updateAITrainingConsent,
    onSuccess: async (value) => {
      queryClient.setQueryData(aiDatasetKeys.consent, value);
      await queryClient.invalidateQueries({ queryKey: aiDatasetKeys.consent });
    }
  });
  const granted = mutation.data?.granted ?? consent.data?.granted ?? false;
  const excluded = consent.data?.excludedData ?? [];

  return (
    <SettingsCard id="ai-improvement-consent">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <SectionHeading
          title="AI improvement consent"
          subtitle="Allow reviewed, sanitized future examples to be considered for AI training datasets."
        />
        <Switch
          checked={granted}
          disabled={consent.isPending || mutation.isPending}
          label="AI improvement consent"
          onChange={(next) => mutation.mutate(next)}
        />
      </div>
      <div className="mt-4 rounded-xl border border-sand-200 bg-sand-50 px-4 py-3">
        <p className="text-[13.5px] font-semibold text-cocoa-800">
          {granted ? "Consent granted" : "Consent not granted"}
        </p>
        <p className="mt-1 text-[13px] leading-relaxed text-cocoa-500">
          Revoking consent marks user-derived candidates ineligible for approval and future export.
        </p>
      </div>
      {excluded.length ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {excluded.map((item) => (
            <span
              className="rounded-full border border-sand-300 bg-white px-3 py-1 text-[12px] text-cocoa-500"
              key={item}
            >
              {item}
            </span>
          ))}
        </div>
      ) : null}
      <div className="mt-4">
        <SaveNotice
          errorMessage={
            mutation.isError || consent.isError
              ? getErrorMessage(mutation.error ?? consent.error, "Could not update AI consent.")
              : null
          }
          successMessage={mutation.isSuccess ? "AI consent updated." : null}
        />
      </div>
    </SettingsCard>
  );
}
