export type GenerationJobType =
  | "full_generation"
  | "day_regeneration"
  | "item_regeneration"
  | "quality_improvement_day"
  | "quality_improvement_item";

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
};

export type GenerationJobsListResponse = {
  items: GenerationJob[];
  limit: number;
};
