import { apiFetch } from "@/lib/api/client";
import {
  getExternalIntegrationsApiBaseUrl,
  getWorkerApiBaseUrl
} from "@/lib/config";
import type {
  DLQMessage,
  OpsJob,
  OpsJobSummary,
  OpsJobsResponse,
  ProviderStatus,
  QueueStatus,
  WorkerStatus
} from "@/types/ops";

type OpsJobEnvelope = { job: OpsJob };
type OpsRetryResponse = { retried: boolean; newJob: OpsJob };

export type OpsJobFilters = {
  status?: string;
  jobType?: string;
  errorCode?: string;
  tripId?: string;
  userId?: string;
  createdAfter?: string;
  createdBefore?: string;
};

export const opsKeys = {
  summary: ["ops", "summary"] as const,
  jobs: (filters: OpsJobFilters) => ["ops", "jobs", filters] as const,
  job: (jobId: string | null) => ["ops", "job", jobId] as const,
  worker: ["ops", "worker"] as const,
  queues: ["ops", "queues"] as const,
  dlq: ["ops", "dlq"] as const,
  providers: ["ops", "providers"] as const
};

export async function getOpsJobSummary(): Promise<OpsJobSummary> {
  return apiFetch<OpsJobSummary>("/ops/jobs/summary");
}

export async function getOpsJobs(filters: OpsJobFilters): Promise<OpsJobsResponse> {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => {
    if (value?.trim()) {
      params.set(key, value.trim());
    }
  });
  const query = params.toString();
  return apiFetch<OpsJobsResponse>(`/ops/jobs${query ? `?${query}` : ""}`);
}

export async function getOpsJob(jobId: string): Promise<OpsJob> {
  const response = await apiFetch<OpsJobEnvelope>(`/ops/jobs/${jobId}`);
  return response.job;
}

export async function retryOpsJob(jobId: string, reason: string): Promise<OpsJob> {
  const response = await apiFetch<OpsRetryResponse>(`/ops/jobs/${jobId}/retry`, {
    method: "POST",
    body: JSON.stringify({ reason })
  });
  return response.newJob;
}

export async function cancelOpsJob(jobId: string, reason: string): Promise<OpsJob> {
  const response = await apiFetch<OpsJobEnvelope>(`/ops/jobs/${jobId}/cancel`, {
    method: "POST",
    body: JSON.stringify({ reason })
  });
  return response.job;
}

export async function markOpsJobFailed(jobId: string, reason: string): Promise<OpsJob> {
  const response = await apiFetch<OpsJobEnvelope>(`/ops/jobs/${jobId}/mark-failed`, {
    method: "POST",
    body: JSON.stringify({ reason })
  });
  return response.job;
}

export async function getWorkerStatus(): Promise<WorkerStatus> {
  return apiFetch<WorkerStatus>("/ops/worker/status", {}, {
    baseUrl: getWorkerApiBaseUrl(),
    serviceName: "Worker Service"
  });
}

export async function getQueuesStatus(): Promise<{ queues: QueueStatus[] }> {
  return apiFetch<{ queues: QueueStatus[] }>("/ops/queues/status", {}, {
    baseUrl: getWorkerApiBaseUrl(),
    serviceName: "Worker Service"
  });
}

export async function getDlqMessages(limit = 20): Promise<{ messages: DLQMessage[] }> {
  return apiFetch<{ messages: DLQMessage[] }>(`/ops/dlq/messages?limit=${limit}`, {}, {
    baseUrl: getWorkerApiBaseUrl(),
    serviceName: "Worker Service"
  });
}

export async function requeueDlqMessage(messageId: string, reason: string) {
  return apiFetch<{ requeued: boolean }>(`/ops/dlq/messages/${messageId}/requeue`, {
    method: "POST",
    body: JSON.stringify({ reason })
  }, {
    baseUrl: getWorkerApiBaseUrl(),
    serviceName: "Worker Service"
  });
}

export async function discardDlqMessage(messageId: string, reason: string) {
  return apiFetch<{ discarded: boolean }>(`/ops/dlq/messages/${messageId}/discard`, {
    method: "POST",
    body: JSON.stringify({ reason })
  }, {
    baseUrl: getWorkerApiBaseUrl(),
    serviceName: "Worker Service"
  });
}

export async function getProviderStatus(): Promise<{ providers: ProviderStatus[] }> {
  return apiFetch<{ providers: ProviderStatus[] }>("/ops/providers/status", {}, {
    baseUrl: getExternalIntegrationsApiBaseUrl(),
    serviceName: "External Integrations Service"
  });
}
