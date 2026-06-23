import { apiFetch } from "@/lib/api/client";
import type {
  ItineraryVersionDetail,
  ListItineraryVersionsResponse
} from "@/types/itinerary-version";
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
  detail: (id: string) => [...tripKeys.details(), id] as const,
  itineraryVersions: (tripId: string) => [...tripKeys.detail(tripId), "itinerary-versions"] as const,
  itineraryVersion: (tripId: string, versionId: string) =>
    [...tripKeys.itineraryVersions(tripId), versionId] as const
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

export function updateTripItinerary(tripId: string, itinerary: Itinerary) {
  return apiFetch<Trip>(`/trips/${tripId}/itinerary`, {
    method: "PUT",
    body: JSON.stringify({ itinerary })
  });
}

export function regenerateItineraryDay(
  tripId: string,
  dayNumber: number,
  instruction?: string
) {
  return apiFetch<Trip>(`/trips/${tripId}/itinerary/days/${dayNumber}/regenerate`, {
    method: "POST",
    body: JSON.stringify(cleanRegenerationPayload(instruction))
  });
}

export function regenerateItineraryItem(
  tripId: string,
  dayNumber: number,
  itemIndex: number,
  instruction?: string
) {
  return apiFetch<Trip>(
    `/trips/${tripId}/itinerary/days/${dayNumber}/items/${itemIndex}/regenerate`,
    {
      method: "POST",
      body: JSON.stringify(cleanRegenerationPayload(instruction))
    }
  );
}

export function listItineraryVersions(tripId: string) {
  return apiFetch<ListItineraryVersionsResponse>(
    `/trips/${tripId}/itinerary/versions`
  );
}

export function getItineraryVersion(tripId: string, versionId: string) {
  return apiFetch<ItineraryVersionDetail>(
    `/trips/${tripId}/itinerary/versions/${versionId}`
  );
}

export function restoreItineraryVersion(tripId: string, versionId: string) {
  return apiFetch<Trip>(
    `/trips/${tripId}/itinerary/versions/${versionId}/restore`,
    {
      method: "POST"
    }
  );
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

function cleanRegenerationPayload(instruction?: string) {
  const trimmed = instruction?.trim() ?? "";
  return trimmed ? { instruction: trimmed } : {};
}
