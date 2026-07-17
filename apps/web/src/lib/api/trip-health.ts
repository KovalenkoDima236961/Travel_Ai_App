import { apiFetch } from "@/shared/api/client";
import type { TripHealth } from "@/types/trip-health";
import { queryKeys } from "@/lib/query-keys";

export const tripHealthKeys = {
  all: ["trip-health"] as const,
  detail: (tripId: string) => queryKeys.trip.health(tripId)
};

export function getTripHealth(tripId: string) {
  return apiFetch<TripHealth>(`/trips/${tripId}/health`);
}
