"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getBudgetSuggestion,
  getPersonalizationContext,
  getPersonalizationSummary,
  getPreferenceCompleteness,
  getRecommendedTemplates,
  personalizationKeys,
  submitPersonalizationFeedback
} from "@/lib/api/personalization";
import type { PersonalizationFeedbackInput } from "@/types/personalization";

export function usePreferenceCompleteness() { return useQuery({ queryKey: personalizationKeys.completeness(), queryFn: getPreferenceCompleteness }); }
export function usePersonalizationSummary() { return useQuery({ queryKey: personalizationKeys.summary(), queryFn: getPersonalizationSummary }); }
export function usePersonalizationContext(tripId?: string) { return useQuery({ queryKey: personalizationKeys.context(tripId), queryFn: () => getPersonalizationContext(tripId) }); }
export function useBudgetSuggestion(tripId?: string) { return useQuery({ queryKey: personalizationKeys.budget(tripId ?? ""), queryFn: () => getBudgetSuggestion(tripId ?? ""), enabled: Boolean(tripId) }); }
export function useRecommendedTemplates(workspaceId?: string) { return useQuery({ queryKey: personalizationKeys.templates(workspaceId), queryFn: () => getRecommendedTemplates(workspaceId) }); }
export function useSubmitPersonalizationFeedback() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: PersonalizationFeedbackInput) => submitPersonalizationFeedback(input),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: personalizationKeys.all })
  });
}
