"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { tripKeys } from "@/lib/api/trips";
import {
  createTripFromSuggestion,
  getTripDiscoverySuggestions,
  getTripDiscoverySession,
  listTripDiscoverySessions,
  refineTripDiscovery,
  surpriseMe,
  tripDiscoveryKeys
} from "@/lib/api/trip-discovery";
import type {
  CreateTripFromSuggestionRequest,
  RefineDiscoveryRequest
} from "@/types/trip-discovery";

export function useTripDiscoverySuggestions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: getTripDiscoverySuggestions,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: tripDiscoveryKeys.sessions() })
  });
}

export function useSurpriseMe() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: surpriseMe,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: tripDiscoveryKeys.sessions() })
  });
}

export function useRefineTripDiscovery(sessionId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: RefineDiscoveryRequest) => {
      if (!sessionId) {
        throw new Error("Discovery session is required.");
      }
      return refineTripDiscovery(sessionId, input);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: tripDiscoveryKeys.sessions() })
  });
}

export function useCreateTripFromSuggestion(sessionId?: string, suggestionId?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTripFromSuggestionRequest) => {
      if (!sessionId || !suggestionId) {
        throw new Error("Select a destination before creating a trip.");
      }
      return createTripFromSuggestion(sessionId, suggestionId, input);
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() }),
        queryClient.invalidateQueries({ queryKey: tripDiscoveryKeys.sessions() })
      ]);
    }
  });
}

export function useTripDiscoverySessions(enabled = true) {
  return useQuery({
    queryKey: tripDiscoveryKeys.sessions(),
    queryFn: () => listTripDiscoverySessions(),
    enabled
  });
}

export function useTripDiscoverySession(sessionId?: string) {
  return useQuery({
    queryKey: sessionId ? tripDiscoveryKeys.session(sessionId) : tripDiscoveryKeys.sessions(),
    queryFn: () => getTripDiscoverySession(sessionId ?? ""),
    enabled: Boolean(sessionId)
  });
}
