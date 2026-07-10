import { apiFetch } from "@/shared/api/client";
import type {
  CreateTripPollInput,
  DiscoverySuggestionVoteValue,
  DiscoverySuggestionVotesResponse,
  GroupPreferencesSummary,
  ItineraryItemReactionSummary,
  SetItineraryItemReactionInput,
  TripPoll,
  VoteTripPollInput
} from "@/types/trip-decisions";

type ListTripPollsResponse = {
  items: TripPoll[];
};

type ListItineraryReactionsResponse = {
  items: ItineraryItemReactionSummary[];
};

export const tripDecisionKeys = {
  all: (tripId: string) => ["trips", "detail", tripId, "decisions"] as const,
  polls: (tripId: string) => [...tripDecisionKeys.all(tripId), "polls"] as const,
  poll: (tripId: string, pollId: string) =>
    [...tripDecisionKeys.polls(tripId), pollId] as const,
  reactions: (tripId: string) => [...tripDecisionKeys.all(tripId), "reactions"] as const,
  itemReactions: (tripId: string, dayNumber: number, itemIndex: number) =>
    [...tripDecisionKeys.reactions(tripId), dayNumber, itemIndex] as const,
  groupPreferences: (tripId: string) =>
    [...tripDecisionKeys.all(tripId), "group-preferences"] as const,
  discoveryVotes: (sessionId: string) =>
    ["trip-discovery", "sessions", sessionId, "votes"] as const
};

export async function getTripPolls(tripId: string): Promise<TripPoll[]> {
  const response = await apiFetch<ListTripPollsResponse>(`/trips/${tripId}/polls`);
  return response?.items ?? [];
}

export function getTripPoll(tripId: string, pollId: string): Promise<TripPoll> {
  return apiFetch<TripPoll>(`/trips/${tripId}/polls/${pollId}`);
}

export function createTripPoll(
  tripId: string,
  input: CreateTripPollInput
): Promise<TripPoll> {
  return apiFetch<TripPoll>(`/trips/${tripId}/polls`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function voteTripPoll(
  tripId: string,
  pollId: string,
  input: VoteTripPollInput
): Promise<TripPoll> {
  return apiFetch<TripPoll>(`/trips/${tripId}/polls/${pollId}/vote`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function closeTripPoll(tripId: string, pollId: string): Promise<TripPoll> {
  return apiFetch<TripPoll>(`/trips/${tripId}/polls/${pollId}/close`, {
    method: "POST"
  });
}

export function archiveTripPoll(tripId: string, pollId: string): Promise<TripPoll> {
  return apiFetch<TripPoll>(`/trips/${tripId}/polls/${pollId}/archive`, {
    method: "POST"
  });
}

export function setItineraryItemReaction(
  tripId: string,
  input: SetItineraryItemReactionInput
): Promise<ItineraryItemReactionSummary> {
  return apiFetch<ItineraryItemReactionSummary>(`/trips/${tripId}/itinerary/reactions`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function getItineraryReactions(
  tripId: string
): Promise<ItineraryItemReactionSummary[]> {
  const response = await apiFetch<ListItineraryReactionsResponse>(
    `/trips/${tripId}/itinerary/reactions`
  );
  return response?.items ?? [];
}

export function getItemReactions(
  tripId: string,
  dayNumber: number,
  itemIndex: number
): Promise<ItineraryItemReactionSummary> {
  return apiFetch<ItineraryItemReactionSummary>(
    `/trips/${tripId}/itinerary/days/${dayNumber}/items/${itemIndex}/reactions`
  );
}

export function deleteMyItineraryReaction(
  tripId: string,
  dayNumber: number,
  itemIndex: number
): Promise<{ success: boolean }> {
  return apiFetch<{ success: boolean }>(
    `/trips/${tripId}/itinerary/days/${dayNumber}/items/${itemIndex}/reactions/me`,
    { method: "DELETE" }
  );
}

export function voteDiscoverySuggestion(
  sessionId: string,
  suggestionId: string,
  input: { vote: DiscoverySuggestionVoteValue; metadata?: Record<string, unknown> }
): Promise<DiscoverySuggestionVotesResponse> {
  return apiFetch<DiscoverySuggestionVotesResponse>(
    `/trip-discovery/sessions/${sessionId}/suggestions/${suggestionId}/vote`,
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );
}

export function getDiscoverySuggestionVotes(
  sessionId: string
): Promise<DiscoverySuggestionVotesResponse> {
  return apiFetch<DiscoverySuggestionVotesResponse>(
    `/trip-discovery/sessions/${sessionId}/votes`
  );
}

export function getGroupPreferences(tripId: string): Promise<GroupPreferencesSummary> {
  return apiFetch<GroupPreferencesSummary>(`/trips/${tripId}/group-preferences`);
}
