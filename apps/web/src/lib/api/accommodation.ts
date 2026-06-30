import { apiFetch } from "@/lib/api/client";
import type { TripAccommodation } from "@/types/accommodation";

export const accommodationKeys = {
  detail: (tripId: string) => ["trips", "detail", tripId, "accommodation"] as const
};

type AccommodationEnvelope = {
  accommodation: TripAccommodation | null;
};

export async function getTripAccommodation(tripId: string): Promise<TripAccommodation | null> {
  const response = await apiFetch<AccommodationEnvelope>(`/trips/${tripId}/accommodation`);
  return response.accommodation;
}

export async function updateTripAccommodation(
  tripId: string,
  accommodation: TripAccommodation
): Promise<TripAccommodation> {
  const response = await apiFetch<AccommodationEnvelope>(`/trips/${tripId}/accommodation`, {
    method: "PUT",
    body: JSON.stringify({ accommodation: cleanAccommodation(accommodation) })
  });
  if (!response.accommodation) {
    throw new Error("Accommodation update did not return an accommodation.");
  }
  return response.accommodation;
}

export function deleteTripAccommodation(tripId: string): Promise<{ success: boolean }> {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/accommodation`, {
    method: "DELETE"
  });
}

function cleanAccommodation(accommodation: TripAccommodation): TripAccommodation {
  const cost = accommodation.estimatedCost;
  const normalizedCost =
    cost && cost.amount != null
      ? {
          amount: cost.amount,
          currency: cost.currency?.trim().toUpperCase() || null,
          category: "accommodation" as const,
          confidence: cost.confidence ?? null,
          source: "manual" as const,
          note: cost.note?.trim() || null
        }
      : null;

  return {
    name: accommodation.name.trim(),
    type: accommodation.type || "other",
    address: accommodation.address?.trim() || null,
    place: accommodation.place ?? null,
    checkInDate: accommodation.checkInDate || null,
    checkOutDate: accommodation.checkOutDate || null,
    estimatedCost: normalizedCost,
    notes: accommodation.notes?.trim() || null
  };
}
