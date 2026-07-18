import { apiFetch } from "@/shared/api/client";
import type { ArchiveTripInput, ArchiveTripResponse, TripLibraryFilters, TripLibraryInsights, TripLibraryResponse } from "@/types/library";

export const tripLibraryKeys = {
  all: ["trip-library"] as const,
  list: (filters: TripLibraryFilters) => [...tripLibraryKeys.all, "list", filters] as const,
  insights: (params: { workspaceId?: string; year?: number }) => [...tripLibraryKeys.all, "insights", params] as const
};

export function getTripLibrary(filters: TripLibraryFilters = {}) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== null && value !== "") query.set(key, String(value));
  }
  const suffix = query.toString();
  return apiFetch<TripLibraryResponse>(`/trips/library${suffix ? `?${suffix}` : ""}`);
}

export function getTripLibraryInsights(params: { workspaceId?: string; year?: number } = {}) {
  const query = new URLSearchParams();
  if (params.workspaceId) query.set("workspaceId", params.workspaceId);
  if (params.year) query.set("year", String(params.year));
  const suffix = query.toString();
  return apiFetch<TripLibraryInsights>(`/trips/library/insights${suffix ? `?${suffix}` : ""}`);
}

export function archiveTrip(tripId: string, input: ArchiveTripInput = {}) {
  return apiFetch<ArchiveTripResponse>(`/trips/${tripId}/archive`, { method: "POST", body: JSON.stringify(input) });
}

export function restoreTrip(tripId: string) {
  return apiFetch<ArchiveTripResponse>(`/trips/${tripId}/restore`, { method: "POST" });
}
