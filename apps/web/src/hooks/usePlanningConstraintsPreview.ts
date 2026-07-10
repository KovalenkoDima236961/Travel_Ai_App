"use client";

import { useMutation } from "@tanstack/react-query";
import { previewPlanningConstraints } from "@/lib/api/planning-constraints";
import type {
  PlanningConstraintsPreviewRequest,
  PlanningConstraintsPreviewResponse
} from "@/types/planning-constraints";

export function usePlanningConstraintsPreview() {
  return useMutation<PlanningConstraintsPreviewResponse, Error, PlanningConstraintsPreviewRequest>({
    mutationFn: previewPlanningConstraints
  });
}

