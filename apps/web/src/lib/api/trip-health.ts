import { apiFetch } from "@/shared/api/client";
import type { TripHealth } from "@/types/trip-health";

export const tripHealthKeys = {
  all: ["trip-health"] as const,
  detail: (tripId: string) => [...tripHealthKeys.all, tripId] as const
};

export function getTripHealth(tripId: string) {
  return apiFetch<TripHealth>(`/trips/${tripId}/health`);
}
