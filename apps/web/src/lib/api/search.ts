import { apiFetch } from "@/shared/api/client";
import type { SearchParams, SearchResponse } from "@/types/search";

export const searchKeys = {
  all: ["search"] as const,
  global: (params: SearchParams) => [...searchKeys.all, "global", params] as const
};

export function searchGlobal(params: SearchParams) {
  const query = new URLSearchParams();
  query.set("q", params.q);
  if (params.scope) {
    query.set("scope", params.scope);
  }
  if (params.tripId) {
    query.set("tripId", params.tripId);
  }
  if (params.workspaceId) {
    query.set("workspaceId", params.workspaceId);
  }
  if (params.limit != null) {
    query.set("limit", String(params.limit));
  }
  if (params.includeCommands) {
    query.set("includeCommands", "true");
  }
  return apiFetch<SearchResponse>(`/search?${query.toString()}`);
}
