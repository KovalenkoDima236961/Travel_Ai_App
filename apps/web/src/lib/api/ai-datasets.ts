import { apiFetch } from "@/shared/api/client";

export type TrainingConsentResponse = {
  status: string;
  granted: boolean;
  consentVersion: string;
  excludedData: string[];
  record?: {
    id: string;
    scopeType: string;
    scopeId?: string | null;
    status: string;
    grantedAt?: string | null;
    revokedAt?: string | null;
    updatedAt: string;
  } | null;
};

export type AIDatasetReadiness = {
  ready: boolean;
  blockers: string[];
  approvedExampleCount: number;
  taskDistribution: Record<string, number>;
  languageDistribution: Record<string, number>;
  consentCoverage: Record<string, number>;
  sanitizationFailureCount: number;
  duplicateCount: number;
  holdoutCount: number;
  baselineEvalStatus: string;
  recommendation: string;
};

export type AIDatasetProject = {
  id: string;
  key: string;
  name: string;
  taskType: string;
  schemaVersion: string;
  status: string;
  minimumQualityScore: number;
  consentRequired: boolean;
};

export type AIDatasetExample = {
  id: string;
  datasetProjectId: string;
  sourceType: string;
  taskType: string;
  language: string;
  consentStatus: string;
  sanitizationStatus: string;
  qualityStatus: string;
  reviewStatus: string;
  qualityScore?: number | null;
  split?: string | null;
  exportStatus: string;
  createdAt: string;
  updatedAt: string;
};

export type AIDatasetExampleFilters = {
  datasetProjectId?: string;
  reviewStatus?: string;
  sanitizationStatus?: string;
  qualityStatus?: string;
  consentStatus?: string;
  taskType?: string;
  sourceType?: string;
  limit?: number;
  offset?: number;
};

export const aiDatasetKeys = {
  consent: ["ai-datasets", "consent"] as const,
  readiness: ["ai-datasets", "readiness"] as const,
  projects: ["ai-datasets", "projects"] as const,
  examples: (filters: AIDatasetExampleFilters) => ["ai-datasets", "examples", filters] as const
};

export function getAITrainingConsent() {
  return apiFetch<TrainingConsentResponse>("/ai-training/consent");
}

export function updateAITrainingConsent(granted: boolean) {
  return apiFetch<TrainingConsentResponse>("/ai-training/consent", {
    method: "PUT",
    body: JSON.stringify({ granted })
  });
}

export function getAIDatasetReadiness() {
  return apiFetch<AIDatasetReadiness>("/ops/ai/fine-tuning/readiness");
}

export function listAIDatasetProjects() {
  return apiFetch<{ projects: AIDatasetProject[] }>("/ops/ai/datasets/projects");
}

export function listAIDatasetExamples(filters: AIDatasetExampleFilters = {}) {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => {
    if (value != null && String(value).trim()) {
      params.set(key, String(value).trim());
    }
  });
  const query = params.toString();
  return apiFetch<{ examples: AIDatasetExample[] }>(
    `/ops/ai/datasets/examples${query ? `?${query}` : ""}`
  );
}

export function approveAIDatasetExample(exampleId: string, reason: string) {
  return apiFetch<AIDatasetExample>(
    `/ops/ai/datasets/examples/${encodeURIComponent(exampleId)}/approve`,
    {
      method: "POST",
      body: JSON.stringify({ reason })
    }
  );
}

export function rejectAIDatasetExample(exampleId: string, reason: string) {
  return apiFetch<AIDatasetExample>(
    `/ops/ai/datasets/examples/${encodeURIComponent(exampleId)}/reject`,
    {
      method: "POST",
      body: JSON.stringify({ reason })
    }
  );
}
