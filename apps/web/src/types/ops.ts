import type { GenerationJobStatus, GenerationJobType } from "@/types/generation-jobs";

export type OpsPayloadSummary = {
  dayNumber?: number | null;
  itemIndex?: number | null;
  scope?: string;
  targetReductionAmount?: number | null;
  currency?: string | null;
  hasInstruction: boolean;
  hasConstraints: boolean;
};

export type OpsJob = {
  id: string;
  tripId: string;
  workspaceId?: string | null;
  scope?: "personal" | "workspace" | string;
  requestedByUserId: string;
  jobType: GenerationJobType;
  status: GenerationJobStatus;
  payloadSummary?: OpsPayloadSummary;
  errorCode?: string | null;
  errorMessage?: string | null;
  expectedItineraryRevision: number;
  resultItineraryRevision?: number | null;
  correlationId?: string | null;
  requestId?: string | null;
  retriedFromJobId?: string | null;
  createdAt: string;
  startedAt?: string | null;
  completedAt?: string | null;
  cancelledAt?: string | null;
  updatedAt: string;
  durationMs?: number | null;
  attemptCount: number;
  canRetry: boolean;
  canCancel: boolean;
  canMarkFailed: boolean;
};

export type OpsJobSummary = {
  countsByStatus: Record<string, number>;
  countsByType: Record<string, number>;
  recentFailures: Array<{
    jobId: string;
    jobType: GenerationJobType;
    errorCode: string;
    createdAt: string;
  }>;
  staleRunningCount: number;
};

export type OpsJobsResponse = {
  jobs: OpsJob[];
  nextCursor: string | null;
  nextOffset?: number;
};

export type WorkerStatus = {
  service: string;
  enabled: boolean;
  healthy: boolean;
  rabbitmqConnected: boolean;
  dbConnected: boolean;
  concurrency: number;
  prefetch: number;
  activeJobs: Array<{
    jobId: string;
    tripId: string;
    jobType: string;
    startedAt: string;
    durationMs: number;
    correlationId?: string;
  }>;
  startedAt: string;
  version: string;
};

export type QueueStatus = {
  name: string;
  messagesReady: number;
  messagesUnacked: number;
  consumers: number;
  publishRate?: number;
  deliverRate?: number;
};

export type DLQMessage = {
  messageId: string;
  jobId?: string;
  tripId?: string;
  jobType?: string;
  attempts: number;
  reason?: string;
  correlationId?: string;
  createdAt?: string;
  deadLetteredAt?: string;
  payloadPreview?: Record<string, unknown>;
};

export type ProviderStatus = {
  name: string;
  activeProvider: string;
  enabled: boolean;
  fallbackEnabled: boolean;
  lastSuccessAt?: string | null;
  lastFailureAt?: string | null;
  recentSuccessCount: number;
  recentFailureCount: number;
  status: "healthy" | "degraded" | "down" | "unknown";
  lastErrorCode?: string;
};

export type ProviderQuotaStatus =
  | "healthy"
  | "nearing_quota"
  | "quota_exceeded"
  | "rate_limited_recently"
  | "disabled"
  | "unknown";

export type ProviderQuotaOperationUsage = {
  operation: string;
  usedToday: number;
  blockedToday: number;
  fallbackToday: number;
  lastAllowedAt?: string | null;
  lastBlockedAt?: string | null;
  lastFallbackAt?: string | null;
};

export type ProviderQuotaSummary = {
  provider: string;
  category: string;
  enabled: boolean;
  rateLimitPerMinute: number;
  dailyQuota: number;
  usedToday: number;
  remainingToday: number;
  blockedToday: number;
  fallbackToday: number;
  status: ProviderQuotaStatus;
  lastBlockedAt?: string | null;
  lastFallbackAt?: string | null;
  operations: ProviderQuotaOperationUsage[];
};

export type ProviderQuotasResponse = {
  date: string;
  enabled: boolean;
  resetAllowed: boolean;
  providers: ProviderQuotaSummary[];
};

export type ProviderQuotaDayUsage = {
  date: string;
  usedCount: number;
  blockedCount: number;
  fallbackCount: number;
};

export type ProviderQuotaDetail = {
  date: string;
  enabled: boolean;
  resetAllowed: boolean;
  provider: ProviderQuotaSummary;
  history: ProviderQuotaDayUsage[];
};
