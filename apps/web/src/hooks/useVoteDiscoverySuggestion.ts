"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getDiscoverySuggestionVotes,
  tripDecisionKeys,
  voteDiscoverySuggestion
} from "@/lib/api/trip-decisions";
import type { DiscoverySuggestionVoteValue } from "@/types/trip-decisions";

export function useDiscoverySuggestionVotes(sessionId?: string, enabled = true) {
  return useQuery({
    queryKey: sessionId ? tripDecisionKeys.discoveryVotes(sessionId) : ["trip-discovery", "votes"],
    queryFn: () => getDiscoverySuggestionVotes(sessionId ?? ""),
    enabled: enabled && Boolean(sessionId)
  });
}

export function useVoteDiscoverySuggestion(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      suggestionId,
      vote,
      metadata
    }: {
      suggestionId: string;
      vote: DiscoverySuggestionVoteValue;
      metadata?: Record<string, unknown>;
    }) => voteDiscoverySuggestion(sessionId, suggestionId, { vote, metadata }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: tripDecisionKeys.discoveryVotes(sessionId)
      });
    }
  });
}
