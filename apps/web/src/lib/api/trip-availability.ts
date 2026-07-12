import { apiFetch } from "@/shared/api/client";
import type {
  ApplyDateOptionInput,
  ApplyDateOptionResponse,
  CreateDateOptionsPollInput,
  CreateDateOptionsPollResult,
  DateOptionsInput,
  DateOptionsResult,
  RequestTripAvailabilityInput,
  TripAvailabilityList,
  TripAvailabilityResponseInfo,
  UpsertTripAvailabilityInput
} from "@/types/trip-availability";

export const tripAvailabilityKeys = {
  all: (tripId: string) => ["trips", "detail", tripId, "availability"] as const,
  list: (tripId: string) => [...tripAvailabilityKeys.all(tripId), "responses"] as const,
  dateOptions: (tripId: string, input: DateOptionsInput = {}) =>
    [...tripAvailabilityKeys.all(tripId), "date-options", cleanDateOptionsInput(input)] as const
};

export function getTripAvailability(tripId: string): Promise<TripAvailabilityList> {
  return apiFetch<TripAvailabilityList>(`/trips/${tripId}/availability`);
}

export function upsertMyTripAvailability(
  tripId: string,
  input: UpsertTripAvailabilityInput
): Promise<TripAvailabilityResponseInfo> {
  return apiFetch<TripAvailabilityResponseInfo>(`/trips/${tripId}/availability/me`, {
    method: "PUT",
    body: JSON.stringify(cleanAvailabilityInput(input))
  });
}

export function deleteMyTripAvailability(tripId: string): Promise<{ success: boolean }> {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/availability/me`, {
    method: "DELETE"
  });
}

export function requestTripAvailability(
  tripId: string,
  input: RequestTripAvailabilityInput = {}
): Promise<TripAvailabilityList["summary"]> {
  return apiFetch<TripAvailabilityList["summary"]>(`/trips/${tripId}/availability/request`, {
    method: "POST",
    body: JSON.stringify({ message: input.message?.trim() ?? "" })
  });
}

export function getTripDateOptions(
  tripId: string,
  input: DateOptionsInput = {}
): Promise<DateOptionsResult> {
  const query = buildDateOptionsQuery(input);
  return apiFetch<DateOptionsResult>(`/trips/${tripId}/date-options${query ? `?${query}` : ""}`);
}

export function generateTripDateOptions(
  tripId: string,
  input: DateOptionsInput = {}
): Promise<DateOptionsResult> {
  return apiFetch<DateOptionsResult>(`/trips/${tripId}/date-options/generate`, {
    method: "POST",
    body: JSON.stringify(cleanDateOptionsInput(input))
  });
}

export function applyTripDateOption(
  tripId: string,
  optionId: string,
  input: ApplyDateOptionInput
): Promise<ApplyDateOptionResponse> {
  return apiFetch<ApplyDateOptionResponse>(`/trips/${tripId}/date-options/${optionId}/apply`, {
    method: "POST",
    body: JSON.stringify({
      expectedItineraryRevision: input.expectedItineraryRevision,
      regenerateItinerary: Boolean(input.regenerateItinerary)
    })
  });
}

export function createDateOptionsPoll(
  tripId: string,
  input: CreateDateOptionsPollInput
): Promise<CreateDateOptionsPollResult> {
  return apiFetch<CreateDateOptionsPollResult>(`/trips/${tripId}/date-options/create-poll`, {
    method: "POST",
    body: JSON.stringify({
      title: input.title?.trim() ?? "",
      optionIds: input.optionIds
    })
  });
}

function buildDateOptionsQuery(input: DateOptionsInput) {
  const params = new URLSearchParams();
  const cleaned = cleanDateOptionsInput(input);
  if (cleaned.minDays != null) {
    params.set("minDays", String(cleaned.minDays));
  }
  if (cleaned.maxDays != null) {
    params.set("maxDays", String(cleaned.maxDays));
  }
  if (cleaned.searchStartDate) {
    params.set("searchStartDate", cleaned.searchStartDate);
  }
  if (cleaned.searchEndDate) {
    params.set("searchEndDate", cleaned.searchEndDate);
  }
  if (cleaned.preferWeekends != null) {
    params.set("preferWeekends", String(cleaned.preferWeekends));
  }
  if (cleaned.limit != null) {
    params.set("limit", String(cleaned.limit));
  }
  return params.toString();
}

function cleanDateOptionsInput(input: DateOptionsInput): DateOptionsInput {
  return {
    minDays: cleanOptionalNumber(input.minDays),
    maxDays: cleanOptionalNumber(input.maxDays),
    searchStartDate: input.searchStartDate?.trim() || undefined,
    searchEndDate: input.searchEndDate?.trim() || undefined,
    preferWeekends: input.preferWeekends ?? undefined,
    limit: cleanOptionalNumber(input.limit)
  };
}

function cleanAvailabilityInput(input: UpsertTripAvailabilityInput): UpsertTripAvailabilityInput {
  return {
    availableRanges: input.availableRanges,
    unavailableRanges: input.unavailableRanges ?? [],
    preferredRanges: input.preferredRanges ?? [],
    minTripDays: cleanOptionalNumber(input.minTripDays),
    maxTripDays: cleanOptionalNumber(input.maxTripDays),
    timezone: input.timezone?.trim() ?? "",
    notes: input.notes?.trim() ?? ""
  };
}

function cleanOptionalNumber(value: number | null | undefined) {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}
