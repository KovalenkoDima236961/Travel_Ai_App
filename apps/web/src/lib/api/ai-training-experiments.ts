import { apiFetch } from "@/shared/api/client";

export type AITrainingExperimentStatus =
  | "draft"
  | "validated"
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | "rejected"
  | "promoted_staging"
  | "promoted_production";

export type AIModelVariant = "base" | "grounded_baseline" | "fine_tuned_candidate";

export type AITrainingExperiment = {
  id: string;
  key: string;
  name: string;
  taskType: "grounded_itinerary_generation";
  method: "lora" | "qlora";
  status: AITrainingExperimentStatus;
  baseModelName: string;
  datasetVersion: string;
  trainCount: number;
  validationCount: number;
  testCount: number;
  holdoutCount: number;
  adapterKey?: string | null;
  adapterChecksum?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type AITrainingEvaluationSummary = {
  variant: AIModelVariant;
  split: "validation" | "test" | "holdout";
  status: string;
  metrics: {
    schemaValidRate?: number;
    factualPrecision?: number;
    groundingCitationRate?: number;
    noPiiRate?: number;
    costRegressionPct?: number;
    latencyRegressionPct?: number;
  };
  reportPath?: string | null;
  createdAt?: string | null;
};

export type AIPromotionGateSummary = {
  approved: boolean;
  failedGates: string[];
  metricsSnapshot: Record<string, number | boolean | string>;
};

export const aiTrainingExperimentKeys = {
  all: ["ai-training-experiments"] as const,
  detail: (experimentId: string | null) => ["ai-training-experiments", experimentId] as const
};

export function listAITrainingExperiments() {
  return apiFetch<{ experiments: AITrainingExperiment[] }>(
    "/ops/ai/fine-tuning/experiments"
  );
}

export function getAITrainingExperiment(experimentId: string) {
  return apiFetch<{
    experiment: AITrainingExperiment;
    evaluations: AITrainingEvaluationSummary[];
    promotionGates?: AIPromotionGateSummary | null;
  }>(`/ops/ai/fine-tuning/experiments/${encodeURIComponent(experimentId)}`);
}

export function validateAITrainingExperiment(experimentId: string) {
  return apiFetch<AITrainingExperiment>(
    `/ops/ai/fine-tuning/experiments/${encodeURIComponent(experimentId)}/validate`,
    { method: "POST" }
  );
}

export function startAITrainingExperiment(experimentId: string) {
  return apiFetch<AITrainingExperiment>(
    `/ops/ai/fine-tuning/experiments/${encodeURIComponent(experimentId)}/start`,
    { method: "POST" }
  );
}

export function cancelAITrainingExperiment(experimentId: string, reason: string) {
  return apiFetch<AITrainingExperiment>(
    `/ops/ai/fine-tuning/experiments/${encodeURIComponent(experimentId)}/cancel`,
    { method: "POST", body: JSON.stringify({ reason }) }
  );
}

export function evaluateAITrainingExperiment(experimentId: string, split = "validation") {
  return apiFetch<{ evaluations: AITrainingEvaluationSummary[] }>(
    `/ops/ai/fine-tuning/experiments/${encodeURIComponent(experimentId)}/evaluate`,
    { method: "POST", body: JSON.stringify({ split }) }
  );
}

export function decideAITrainingExperimentPromotion(
  experimentId: string,
  decision: "approve_staging" | "approve_production" | "reject" | "needs_more_data" | "needs_retraining",
  reason: string
) {
  return apiFetch<AITrainingExperiment>(
    `/ops/ai/fine-tuning/experiments/${encodeURIComponent(experimentId)}/promotion-decisions`,
    { method: "POST", body: JSON.stringify({ decision, reason }) }
  );
}
