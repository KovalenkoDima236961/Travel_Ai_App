import { apiFetch } from "@/shared/api/client";
import type {
  ApplyRouteAlternativeInput,
  CreateRouteAlternativesPollInput,
  CreateRouteAlternativesPollResult,
  CreateTripFromRouteAlternativeInput,
  CreateTripFromRouteAlternativeResult,
  RefineRouteAlternativesInput,
  RouteAlternativeSession,
  RouteAlternativeSessionsResponse,
  RouteAlternativeVote,
  SuggestRouteAlternativesInput,
  SuggestTripRouteAlternativesInput
} from "@/types/route-alternatives";
import type { Trip } from "@/entities/trip/model";

export const routeAlternativeKeys = {
  all: ["route-alternatives"] as const,
  sessions: (params?: { tripId?: string; limit?: number }) =>
    [...routeAlternativeKeys.all, "sessions", params ?? {}] as const,
  session: (sessionId: string) => [...routeAlternativeKeys.all, "session", sessionId] as const,
  tripSessions: (tripId: string) => [...routeAlternativeKeys.all, "trip", tripId] as const
};

export function suggestRouteAlternatives(input: SuggestRouteAlternativesInput) {
  return apiFetch<RouteAlternativeSession>("/route-alternatives/suggest", {
    method: "POST",
    body: JSON.stringify(cleanSuggestPayload(input))
  });
}

export function getRouteAlternativeSessions(params: { tripId?: string; limit?: number } = {}) {
  const searchParams = new URLSearchParams();
  if (params.tripId) {
    searchParams.set("tripId", params.tripId);
  }
  if (params.limit != null) {
    searchParams.set("limit", String(params.limit));
  }
  const query = searchParams.toString();
  return apiFetch<RouteAlternativeSessionsResponse>(
    `/route-alternatives/sessions${query ? `?${query}` : ""}`
  );
}

export function getRouteAlternativeSession(sessionId: string) {
  return apiFetch<RouteAlternativeSession>(`/route-alternatives/sessions/${sessionId}`);
}

export function refineRouteAlternatives(
  sessionId: string,
  input: RefineRouteAlternativesInput
) {
  return apiFetch<RouteAlternativeSession>(`/route-alternatives/sessions/${sessionId}/refine`, {
    method: "POST",
    body: JSON.stringify({
      instruction: input.instruction.trim(),
      ...(input.selectedAlternativeId ? { selectedAlternativeId: input.selectedAlternativeId } : {})
    })
  });
}

export function createTripFromRouteAlternative(
  sessionId: string,
  alternativeId: string,
  input: CreateTripFromRouteAlternativeInput
) {
  return apiFetch<CreateTripFromRouteAlternativeResult>(
    `/route-alternatives/sessions/${sessionId}/alternatives/${alternativeId}/create-trip`,
    {
      method: "POST",
      body: JSON.stringify(cleanCreateTripPayload(input))
    }
  );
}

export function suggestTripRouteAlternatives(
  tripId: string,
  input: SuggestTripRouteAlternativesInput
) {
  return apiFetch<RouteAlternativeSession>(`/trips/${tripId}/route-alternatives`, {
    method: "POST",
    body: JSON.stringify({
      prompt: input.prompt?.trim() ?? "",
      suggestionCount: input.suggestionCount ?? 3,
      useCurrentRouteAsBaseline: input.useCurrentRouteAsBaseline ?? true,
      ...(input.outputLanguage ? { outputLanguage: input.outputLanguage } : {})
    })
  });
}

export function applyRouteAlternative(
  tripId: string,
  sessionId: string,
  alternativeId: string,
  input: ApplyRouteAlternativeInput
) {
  return apiFetch<Trip>(
    `/trips/${tripId}/route-alternatives/${sessionId}/alternatives/${alternativeId}/apply`,
    {
      method: "POST",
      body: JSON.stringify({
        ...(input.expectedItineraryRevision != null
          ? { expectedItineraryRevision: input.expectedItineraryRevision }
          : {}),
        regenerateItinerary: input.regenerateItinerary ?? false
      })
    }
  );
}

export function voteRouteAlternative(
  tripId: string,
  sessionId: string,
  alternativeId: string,
  input: RouteAlternativeVote
) {
  return apiFetch<RouteAlternativeSession>(
    `/trips/${tripId}/route-alternatives/${sessionId}/alternatives/${alternativeId}/vote`,
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );
}

export function createRouteAlternativesPoll(
  tripId: string,
  sessionId: string,
  input: CreateRouteAlternativesPollInput
) {
  return apiFetch<CreateRouteAlternativesPollResult>(
    `/trips/${tripId}/route-alternatives/${sessionId}/create-poll`,
    {
      method: "POST",
      body: JSON.stringify({
        ...(input.title?.trim() ? { title: input.title.trim() } : {}),
        ...(input.alternativeIds?.length ? { alternativeIds: input.alternativeIds } : {})
      })
    }
  );
}

function cleanSuggestPayload(input: SuggestRouteAlternativesInput) {
  return {
    ...input,
    prompt: input.prompt.trim(),
    travelers: input.travelers ?? 1,
    outputLanguage: input.outputLanguage ?? "en",
    suggestionCount: input.suggestionCount ?? 3,
    ...(input.startDate ? { startDate: input.startDate } : {}),
    ...(input.workspaceId ? { workspaceId: input.workspaceId } : {}),
    ...(input.budget ? { budget: cleanBudget(input.budget) } : {})
  };
}

function cleanCreateTripPayload(input: CreateTripFromRouteAlternativeInput) {
  return {
    title: input.title.trim(),
    ...(input.startDate ? { startDate: input.startDate } : {}),
    ...(input.budget ? { budget: cleanBudget(input.budget) } : {}),
    ...(input.travelers != null ? { travelers: input.travelers } : {}),
    ...(input.workspaceId ? { workspaceId: input.workspaceId } : {}),
    autoGenerateItinerary: input.autoGenerateItinerary ?? false
  };
}

function cleanBudget(input: NonNullable<SuggestRouteAlternativesInput["budget"]>) {
  return {
    ...(input.amount != null ? { amount: input.amount } : {}),
    currency: input.currency.toUpperCase(),
    ...(input.confidence ? { confidence: input.confidence } : {})
  };
}
