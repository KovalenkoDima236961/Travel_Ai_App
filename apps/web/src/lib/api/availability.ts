import { apiFetch } from "@/lib/api/client";
import { getExternalIntegrationsApiBaseUrl } from "@/lib/config";
import type {
  AvailabilitySearchRequest,
  AvailabilitySearchResponse
} from "@/types/availability";

export const availabilityKeys = {
  all: ["availability"] as const,
  search: (parts: {
    tripId: string;
    dayNumber: number;
    itemIndex: number;
    date: string;
    itemName: string;
  }) =>
    [
      ...availabilityKeys.all,
      parts.tripId,
      parts.dayNumber,
      parts.itemIndex,
      parts.date,
      parts.itemName
    ] as const
};

const externalIntegrationsOptions = {
  baseUrl: getExternalIntegrationsApiBaseUrl(),
  serviceName: "External Integrations Service"
};

export async function searchAvailability(
  input: AvailabilitySearchRequest
): Promise<AvailabilitySearchResponse> {
  return apiFetch<AvailabilitySearchResponse>(
    "/availability/search",
    {
      method: "POST",
      body: JSON.stringify(input)
    },
    externalIntegrationsOptions
  );
}
