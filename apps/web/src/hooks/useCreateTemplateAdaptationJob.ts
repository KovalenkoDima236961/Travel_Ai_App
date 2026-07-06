"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createTemplateAdaptationJob } from "@/lib/api/template-adaptation";
import { tripTemplateKeys } from "@/lib/api/trip-templates";
import { tripKeys } from "@/lib/api/trips";
import type { GenerationJob } from "@/types/generation-jobs";
import type { TemplateAdaptationInput } from "@/types/template-adaptation";

type CreateInput = {
  templateId: string;
  input: TemplateAdaptationInput;
};

/** Creates a template adaptation job (which also creates the draft trip) and
 * invalidates trip/template lists so the new draft appears. */
export function useCreateTemplateAdaptationJob() {
  const queryClient = useQueryClient();
  return useMutation<GenerationJob, unknown, CreateInput>({
    mutationFn: ({ templateId, input }) => createTemplateAdaptationJob(templateId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tripKeys.lists() });
      queryClient.invalidateQueries({ queryKey: tripTemplateKeys.all });
    }
  });
}
