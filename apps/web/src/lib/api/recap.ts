import { apiFetch } from "@/shared/api/client";
import type {
  LearningCandidate,
  TripRecap,
  TripRecapContent,
  TripRecapFeedback,
  TripRecapResponse,
  TripRecapStatusResponse
} from "@/types/recap";

export const tripRecapKeys = {
  all: ["trip-recaps"] as const,
  status: (tripId: string) => [...tripRecapKeys.all, tripId, "status"] as const,
  detail: (tripId: string) => [...tripRecapKeys.all, tripId] as const
};

export function getTripRecapStatus(tripId: string) {
  return apiFetch<TripRecapStatusResponse>(`/trips/${tripId}/recap/status`);
}

export function getTripRecap(tripId: string) {
  return apiFetch<TripRecapResponse>(`/trips/${tripId}/recap`);
}

export function generateTripRecap(tripId: string, input: { forceRegenerate?: boolean; generateEarly?: boolean; language?: string } = {}) {
  return apiFetch<TripRecap>(`/trips/${tripId}/recap/generate`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function updateTripRecap(tripId: string, recap: TripRecapContent) {
  return apiFetch<TripRecap>(`/trips/${tripId}/recap`, {
    method: "PATCH",
    body: JSON.stringify({ recap })
  });
}

export function finalizeTripRecap(tripId: string) {
  return apiFetch<TripRecap>(`/trips/${tripId}/recap/finalize`, { method: "POST" });
}

export function archiveTripRecap(tripId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/recap`, { method: "DELETE" });
}

export function submitTripRecapFeedback(
  tripId: string,
  input: Omit<TripRecapFeedback, "id" | "createdAt"> & { entityType?: string; entityId?: string }
) {
  return apiFetch<TripRecapFeedback>(`/trips/${tripId}/recap/feedback`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function applyTripRecapLearning(tripId: string, learningCandidates: LearningCandidate[]) {
  return apiFetch<{ feedback: TripRecapFeedback[] }>(`/trips/${tripId}/recap/apply-learning`, {
    method: "POST",
    body: JSON.stringify({ learningCandidates })
  });
}

export function createTemplateFromTripRecap(
  tripId: string,
  input: { title: string; description?: string; visibility: "private" | "workspace"; tags?: string[]; useRecapLessons: boolean }
) {
  return apiFetch<unknown>(`/trips/${tripId}/recap/create-template`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}
