import { apiFetch } from "@/lib/api/client";
import type {
  CreateGenerationJobRequest,
  GenerationJob,
  GenerationJobsListResponse
} from "@/types/generation-jobs";

type GenerationJobEnvelope = {
  job: GenerationJob;
};

export const generationJobKeys = {
  all: (tripId: string) => ["generation-jobs", tripId] as const,
  list: (tripId: string) => [...generationJobKeys.all(tripId), "list"] as const,
  detail: (tripId: string, jobId: string) =>
    [...generationJobKeys.all(tripId), "detail", jobId] as const
};

export async function createGenerationJob(
  tripId: string,
  input: CreateGenerationJobRequest
): Promise<GenerationJob> {
  const response = await apiFetch<GenerationJobEnvelope>(
    `/trips/${tripId}/generation-jobs`,
    {
      method: "POST",
      body: JSON.stringify(cleanCreatePayload(input))
    }
  );
  return response.job;
}

export async function getGenerationJob(
  tripId: string,
  jobId: string
): Promise<GenerationJob> {
  const response = await apiFetch<GenerationJobEnvelope>(
    `/trips/${tripId}/generation-jobs/${jobId}`
  );
  return response.job;
}

export async function listGenerationJobs(tripId: string): Promise<GenerationJob[]> {
  const response = await apiFetch<GenerationJobsListResponse>(
    `/trips/${tripId}/generation-jobs`
  );
  return response.items;
}

export async function cancelGenerationJob(
  tripId: string,
  jobId: string
): Promise<GenerationJob> {
  const response = await apiFetch<GenerationJobEnvelope>(
    `/trips/${tripId}/generation-jobs/${jobId}/cancel`,
    {
      method: "POST"
    }
  );
  return response.job;
}

function cleanCreatePayload(input: CreateGenerationJobRequest) {
  const instruction = input.instruction?.trim();
  return {
    jobType: input.jobType,
    expectedItineraryRevision: input.expectedItineraryRevision,
    ...(instruction ? { instruction } : {}),
    ...(input.dayNumber != null ? { dayNumber: input.dayNumber } : {}),
    ...(input.itemIndex != null ? { itemIndex: input.itemIndex } : {}),
    ...(input.payload != null ? { payload: input.payload } : {})
  };
}
