import { apiFetch } from "@/shared/api/client";
import type {
  CostSplitRule,
  CostSplittingSummary,
  CreateTripTravelerInput,
  TripTraveler,
  TripTravelersResponse,
  UpdateTripTravelerInput
} from "@/entities/cost-splitting/model";
import type { Trip } from "@/entities/trip/model";

export const costSplittingKeys = {
  all: ["cost-splitting"] as const,
  travelers: (tripId: string) => [...costSplittingKeys.all, "travelers", tripId] as const,
  summary: (tripId: string, currency?: string | null) =>
    [...costSplittingKeys.all, "summary", tripId, currency ?? null] as const
};

export function listTripTravelers(tripId: string) {
  return apiFetch<TripTravelersResponse>(`/trips/${tripId}/travelers`);
}

export function createTripTraveler(tripId: string, input: CreateTripTravelerInput) {
  return apiFetch<TripTraveler>(`/trips/${tripId}/travelers`, {
    method: "POST",
    body: JSON.stringify(cleanTravelerPayload(input))
  });
}

export function updateTripTraveler(
  tripId: string,
  travelerId: string,
  input: UpdateTripTravelerInput
) {
  return apiFetch<TripTraveler>(`/trips/${tripId}/travelers/${travelerId}`, {
    method: "PATCH",
    body: JSON.stringify(cleanTravelerPayload(input))
  });
}

export function removeTripTraveler(tripId: string, travelerId: string) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/travelers/${travelerId}`, {
    method: "DELETE"
  });
}

export function updateItemCostSplit(
  tripId: string,
  dayNumber: number,
  itemIndex: number,
  expectedItineraryRevision: number,
  split: CostSplitRule
) {
  return apiFetch<{ trip: Trip }>(
    `/trips/${tripId}/itinerary/days/${dayNumber}/items/${itemIndex}/cost-split`,
    {
      method: "PATCH",
      body: JSON.stringify({ expectedItineraryRevision, split })
    }
  );
}

export function updateAccommodationCostSplit(tripId: string, split: CostSplitRule) {
  return apiFetch<Trip>(`/trips/${tripId}/accommodation/cost-split`, {
    method: "PATCH",
    body: JSON.stringify({ split })
  });
}

export function getCostSplittingSummary(tripId: string, currency?: string | null) {
  const params = new URLSearchParams();
  if (currency) {
    params.set("currency", currency.trim().toUpperCase());
  }
  const query = params.toString();
  return apiFetch<CostSplittingSummary>(
    `/trips/${tripId}/cost-splitting/summary${query ? `?${query}` : ""}`
  );
}

function cleanTravelerPayload(input: CreateTripTravelerInput | UpdateTripTravelerInput) {
  return {
    ...(input.name != null ? { name: input.name.trim() } : {}),
    ...(input.email != null ? { email: input.email.trim() } : {}),
    ...("linkedUserId" in input && input.linkedUserId ? { linkedUserId: input.linkedUserId } : {}),
    ...(input.role ? { role: input.role } : {})
  };
}
