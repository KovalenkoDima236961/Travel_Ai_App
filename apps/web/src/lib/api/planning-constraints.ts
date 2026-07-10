import { apiFetch } from "@/shared/api/client";
import type {
  PlanningConstraintsPreviewRequest,
  PlanningConstraintsPreviewResponse
} from "@/types/planning-constraints";

export const planningConstraintKeys = {
  all: ["planning-constraints"] as const,
  preview: () => [...planningConstraintKeys.all, "preview"] as const
};

export function previewPlanningConstraints(input: PlanningConstraintsPreviewRequest) {
  return apiFetch<PlanningConstraintsPreviewResponse>("/planning-constraints/preview", {
    method: "POST",
    body: JSON.stringify({ ...input, request: input.request ?? {} })
  });
}

