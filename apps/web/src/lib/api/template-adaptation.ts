import { apiFetch } from "@/shared/api/client";
import { getGenerationJob } from "@/lib/api/generation-jobs";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { TemplateAdaptationInput } from "@/entities/template-adaptation/model";

type GenerationJobEnvelope = {
  job: GenerationJob;
};

export const templateAdaptationKeys = {
  all: ["template-adaptation-jobs"] as const,
  job: (tripId: string, jobId: string) =>
    [...templateAdaptationKeys.all, tripId, jobId] as const
};

/** Starts an AI template adaptation. The returned job's `tripId` is the draft
 * trip created up front, so the caller can poll status and open the trip. */
export async function createTemplateAdaptationJob(
  templateId: string,
  input: TemplateAdaptationInput
): Promise<GenerationJob> {
  const response = await apiFetch<GenerationJobEnvelope>(
    `/trip-templates/${templateId}/adaptation-jobs`,
    {
      method: "POST",
      body: JSON.stringify(cleanPayload(input))
    }
  );
  return response.job;
}

/** Reuses the per-trip generation job status endpoint; the draft trip exists
 * immediately so `tripId` is known from the create response. */
export function getTemplateAdaptationJob(
  tripId: string,
  jobId: string
): Promise<GenerationJob> {
  return getGenerationJob(tripId, jobId);
}

function cleanPayload(input: TemplateAdaptationInput) {
  const interests = normalizeList(input.interests);
  const avoid = normalizeList(input.avoid);
  const special = input.specialInstructions?.trim();
  return {
    title: input.title.trim(),
    destination: input.destination.trim(),
    startDate: input.startDate,
    durationDays: input.durationDays,
    ...(input.workspaceId ? { workspaceId: input.workspaceId } : {}),
    ...(input.budget?.amount != null
      ? {
          budget: {
            amount: input.budget.amount,
            currency: input.budget.currency.trim().toUpperCase()
          }
        }
      : {}),
    ...(input.travelers != null ? { travelers: input.travelers } : {}),
    ...(input.pace ? { pace: input.pace } : {}),
    ...(interests.length ? { interests } : {}),
    ...(avoid.length ? { avoid } : {}),
    ...(special ? { specialInstructions: special } : {}),
    fallbackToDeterministic: input.fallbackToDeterministic ?? true
  };
}

function normalizeList(values?: string[]) {
  if (!values) {
    return [] as string[];
  }
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of values.flatMap((value) => value.split(","))) {
    const trimmed = raw.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    out.push(trimmed);
  }
  return out;
}
