import { apiFetch } from "@/shared/api/client";
import type {
  CreateTripFromSuggestionRequest,
  CreateTripFromSuggestionResponse,
  RefineDiscoveryRequest,
  SurpriseMeRequest,
  TripDiscoveryRequest,
  TripDiscoverySession,
  TripDiscoverySessionsResponse
} from "@/types/trip-discovery";

export const tripDiscoveryKeys = {
  all: ["trip-discovery"] as const,
  sessions: () => [...tripDiscoveryKeys.all, "sessions"] as const,
  session: (sessionId: string) => [...tripDiscoveryKeys.sessions(), sessionId] as const
};

export function getTripDiscoverySuggestions(input: TripDiscoveryRequest) {
  return apiFetch<TripDiscoverySession>("/trip-discovery/suggestions", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function surpriseMe(input: SurpriseMeRequest) {
  return apiFetch<TripDiscoverySession>("/trip-discovery/surprise-me", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function refineTripDiscovery(sessionId: string, input: RefineDiscoveryRequest) {
  return apiFetch<TripDiscoverySession>(
    `/trip-discovery/${encodeURIComponent(sessionId)}/refine`,
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );
}

export function createTripFromSuggestion(
  sessionId: string,
  suggestionId: string,
  input: CreateTripFromSuggestionRequest
) {
  return apiFetch<CreateTripFromSuggestionResponse>(
    `/trip-discovery/${encodeURIComponent(sessionId)}/suggestions/${encodeURIComponent(suggestionId)}/create-trip`,
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );
}

export function listTripDiscoverySessions(limit = 20) {
  return apiFetch<TripDiscoverySessionsResponse>(
    `/trip-discovery/sessions?limit=${encodeURIComponent(limit)}`
  );
}

export function getTripDiscoverySession(sessionId: string) {
  return apiFetch<TripDiscoverySession>(
    `/trip-discovery/sessions/${encodeURIComponent(sessionId)}`
  );
}
