import { apiFetch } from "@/shared/api/client";
import type { TripActivityResponse } from "@/entities/activity/model";

// React Query keys for the trip activity feed. Activity is a private,
// authenticated feature and is never fetched from the public share page.
export const activityKeys = {
  all: (tripId: string) => ["trips", "detail", tripId, "activity"] as const
};

type ListTripActivityParams = {
  limit?: number;
  cursor?: string;
};

/**
 * Fetches one newest-first page of a trip's activity feed. The Authorization
 * header is attached by apiFetch. This must only be called from authenticated
 * private trip views — never from the public share page.
 */
export async function listTripActivity(
  tripId: string,
  params: ListTripActivityParams = {}
): Promise<TripActivityResponse> {
  const query = new URLSearchParams();
  if (params.limit != null) {
    query.set("limit", String(params.limit));
  }
  if (params.cursor) {
    query.set("cursor", params.cursor);
  }
  const suffix = query.toString() ? `?${query.toString()}` : "";

  const response = await apiFetch<TripActivityResponse>(`/trips/${tripId}/activity${suffix}`);
  return {
    items: response?.items ?? [],
    nextCursor: response?.nextCursor ?? null
  };
}
