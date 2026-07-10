"use client";

import { useQuery } from "@tanstack/react-query";
import {
  getRouteAlternativeSession,
  getRouteAlternativeSessions,
  routeAlternativeKeys
} from "@/lib/api/route-alternatives";

export function useRouteAlternativeSessions(params: { tripId?: string; limit?: number } = {}, enabled = true) {
  return useQuery({
    queryKey: routeAlternativeKeys.sessions(params),
    queryFn: () => getRouteAlternativeSessions(params),
    enabled
  });
}

export function useRouteAlternativeSession(sessionId?: string) {
  return useQuery({
    queryKey: sessionId ? routeAlternativeKeys.session(sessionId) : routeAlternativeKeys.all,
    queryFn: () => getRouteAlternativeSession(sessionId ?? ""),
    enabled: Boolean(sessionId)
  });
}
