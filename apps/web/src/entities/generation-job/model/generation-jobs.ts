export type GenerationJobType =
  | "full_generation"
  | "day_regeneration"
  | "item_regeneration"
  | "quality_improvement_day"
  | "quality_improvement_item"
  | "budget_optimization_day"
  | "template_adaptation";

export type GenerationJobStatus =
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export type GenerationJob = {
  id: string;
  tripId: string;
  requestedByUserId: string;
  jobType: GenerationJobType;
  status: GenerationJobStatus;
  expectedItineraryRevision: number;
  instruction?: string | null;
  dayNumber?: number | null;
  itemIndex?: number | null;
  payload?: unknown;
  resultPayload?: unknown;
  errorCode?: string | null;
  errorMessage?: string | null;
  resultItineraryRevision?: number | null;
  createdAt: string;
  startedAt?: string | null;
  completedAt?: string | null;
  cancelledAt?: string | null;
  updatedAt: string;
};

export type CreateGenerationJobRequest = {
  jobType: GenerationJobType;
  expectedItineraryRevision: number;
  instruction?: string | null;
  dayNumber?: number | null;
  itemIndex?: number | null;
  payload?: unknown;
};

export type GenerationJobsListResponse = {
  items: GenerationJob[];
  limit: number;
};
