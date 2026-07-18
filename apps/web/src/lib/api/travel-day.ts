import { apiFetch } from "@/shared/api/client";
import { queryKeys } from "@/lib/query-keys";
import type {
  TravelDaySummary,
  UpdateTravelItemStatusInput,
  UpdateTravelItemStatusResponse
} from "@/types/travel-day";

export const travelDayKeys = {
  all: ["travel-day"] as const,
  detail: (tripId: string, date: string) => queryKeys.trip.travelDay(tripId, date)
};

export function getTravelDay(tripId: string, date: string) {
  const query = date ? `?date=${encodeURIComponent(date)}` : "";
  return apiFetch<TravelDaySummary>(`/trips/${tripId}/travel-day${query}`);
}

export function updateTravelItemStatus(
  tripId: string,
  dayNumber: number,
  itemIndex: number,
  input: UpdateTravelItemStatusInput
) {
  return apiFetch<UpdateTravelItemStatusResponse>(
    `/trips/${tripId}/itinerary/days/${dayNumber}/items/${itemIndex}/travel-status`,
    { method: "PATCH", body: JSON.stringify(input) }
  );
}
