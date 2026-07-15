import type { Trip } from "@/entities/trip/model";
import { apiFetch } from "@/shared/api/client";
import type {
  AttachRouteLegTransportOptionInput,
  RemoveRouteLegTransportOptionInput,
  SearchRouteLegTransportInput,
  SelectedTransportOption,
  TransportOption,
  TransportSearchResponse
} from "@/types/transport";

export const transportKeys = {
  all: ["transport"] as const,
  routeLeg: (tripId: string, legId: string) =>
    [...transportKeys.all, "trip", tripId, "route-leg", legId] as const
};

export function searchRouteLegTransportOptions(
  tripId: string,
  legId: string,
  input: SearchRouteLegTransportInput = {}
) {
  return apiFetch<TransportSearchResponse>(
    `/trips/${tripId}/route/legs/${encodeURIComponent(legId)}/transport/search`,
    {
      method: "POST",
      body: JSON.stringify(cleanSearchInput(input))
    }
  );
}

export function attachRouteLegTransportOption(
  tripId: string,
  legId: string,
  input: AttachRouteLegTransportOptionInput
) {
  return apiFetch<Trip>(
    `/trips/${tripId}/route/legs/${encodeURIComponent(legId)}/transport-option`,
    {
      method: "PUT",
      body: JSON.stringify({
        expectedItineraryRevision: input.expectedItineraryRevision,
        updateLegMode: input.updateLegMode ?? true,
        option: input.option
      })
    }
  );
}

export function removeRouteLegTransportOption(
  tripId: string,
  legId: string,
  input: RemoveRouteLegTransportOptionInput = {}
) {
  return apiFetch<Trip>(
    `/trips/${tripId}/route/legs/${encodeURIComponent(legId)}/transport-option`,
    {
      method: "DELETE",
      body: JSON.stringify(input)
    }
  );
}

export function selectedOptionFromTransportOption(option: TransportOption): SelectedTransportOption {
  return {
    id: option.id,
    mode: option.mode,
    provider: option.provider,
    operatorName: option.operatorName,
    serviceName: option.serviceName,
    originName: option.originName,
    destinationName: option.destinationName,
    departureDate: option.departureDate,
    departureTime: option.departureTime,
    arrivalDate: option.arrivalDate,
    arrivalTime: option.arrivalTime,
    durationMinutes: option.durationMinutes,
    transfers: option.transfers,
    estimatedPrice: option.estimatedPrice,
    bookingUrl: option.bookingUrl,
    providerUrl: option.providerUrl,
    status: option.status,
    confidence: option.confidence,
    baggageNotes: option.baggageNotes,
    accessibilityNotes: option.accessibilityNotes,
    warnings: option.warnings ?? []
  };
}

function cleanSearchInput(input: SearchRouteLegTransportInput) {
  return {
    ...(input.date ? { date: input.date } : {}),
    ...(input.time ? { time: input.time } : {}),
    ...(input.timePreference ? { timePreference: input.timePreference } : {}),
    ...(input.modes && input.modes.length > 0 ? { modes: input.modes } : {}),
    ...(input.travelers && input.travelers > 0 ? { travelers: input.travelers } : {}),
    ...(input.currency ? { currency: input.currency.trim().toUpperCase() } : {}),
    ...(input.constraints ? { constraints: input.constraints } : {})
  };
}
