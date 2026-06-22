import { apiFetch } from "@/lib/api/client";
import type { CreateTripInput, Itinerary, Trip, TripsListResponse } from "@/types/trip";

type ListTripsParams = {
  limit?: number;
  offset?: number;
};

export const tripKeys = {
  all: ["trips"] as const,
  lists: () => [...tripKeys.all, "list"] as const,
  list: (params: ListTripsParams) => [...tripKeys.lists(), params] as const,
  details: () => [...tripKeys.all, "detail"] as const,
  detail: (id: string) => [...tripKeys.details(), id] as const
};

export function listTrips(params: ListTripsParams = {}) {
  const searchParams = new URLSearchParams();

  if (params.limit != null) {
    searchParams.set("limit", String(params.limit));
  }

  if (params.offset != null) {
    searchParams.set("offset", String(params.offset));
  }

  const query = searchParams.toString();
  return apiFetch<TripsListResponse>(`/trips${query ? `?${query}` : ""}`);
}

export function getTrip(id: string) {
  return apiFetch<Trip>(`/trips/${id}`);
}

export function createTrip(input: CreateTripInput) {
  return apiFetch<Trip>("/trips", {
    method: "POST",
    body: JSON.stringify(cleanCreateTripPayload(input))
  });
}

export function generateItinerary(id: string) {
  return apiFetch<Trip | Itinerary>(`/trips/${id}/generate`, {
    method: "POST"
  });
}

function cleanCreateTripPayload(input: CreateTripInput) {
  return {
    destination: input.destination.trim(),
    ...(input.startDate ? { startDate: input.startDate } : {}),
    days: input.days,
    ...(input.budgetAmount != null ? { budgetAmount: input.budgetAmount } : {}),
    budgetCurrency: input.budgetCurrency.trim().toUpperCase(),
    travelers: input.travelers,
    interests: input.interests,
    pace: input.pace
  };
}
