import { apiFetch } from "@/shared/api/client";
import { getUserApiBaseUrl } from "@/shared/config";
import type {
  BudgetSuggestion,
  FeedbackSummary,
  PersonalizationContext,
  PersonalizationFeedbackInput,
  PreferenceCompleteness,
  RecommendedTemplate
} from "@/types/personalization";

export const personalizationKeys = {
  all: ["personalization"] as const,
  completeness: () => [...personalizationKeys.all, "completeness"] as const,
  summary: () => [...personalizationKeys.all, "summary"] as const,
  context: (tripId?: string) => [...personalizationKeys.all, "context", tripId ?? "me"] as const,
  templates: (workspaceId?: string) => [...personalizationKeys.all, "templates", workspaceId ?? "personal"] as const,
  budget: (tripId: string) => [...personalizationKeys.all, "budget", tripId] as const
};

export function getPreferenceCompleteness() {
  return apiFetch<PreferenceCompleteness>("/users/me/preferences/completeness", {}, { baseUrl: getUserApiBaseUrl(), serviceName: "User Service" });
}

export function getPersonalizationSummary() { return apiFetch<FeedbackSummary>("/personalization/feedback/summary"); }
export function getPersonalizationContext(tripId?: string) {
  const query = tripId ? `?tripId=${encodeURIComponent(tripId)}` : "";
  return apiFetch<PersonalizationContext>(`/personalization/context${query}`);
}
export function submitPersonalizationFeedback(input: PersonalizationFeedbackInput) {
  return apiFetch("/personalization/feedback", { method: "POST", body: JSON.stringify(input) });
}
export function clearPersonalizationFeedback() { return apiFetch<void>("/personalization/feedback", { method: "DELETE" }); }
export function getRecommendedTemplates(workspaceId?: string) {
  const query = new URLSearchParams({ limit: "10" }); if (workspaceId) query.set("workspaceId", workspaceId);
  return apiFetch<{ items: RecommendedTemplate[] }>(`/trip-templates/recommended?${query.toString()}`);
}
export function getBudgetSuggestion(tripId: string) { return apiFetch<BudgetSuggestion>(`/trips/${encodeURIComponent(tripId)}/budget-suggestion`); }
