import type { EstimatedCost } from "@/types/budget";
import type { ItineraryItem, Trip } from "@/types/trip";

export type AvailabilityStatus = "available" | "limited" | "unavailable" | "unknown";

export type AvailabilityProviderResult =
  | "success"
  | "no_match"
  | "unavailable"
  | "provider_error"
  | "rate_limited"
  | "quota_exceeded"
  | "fallback";

export type AvailabilityPriceType = "per_person" | "per_group" | "total" | "unknown";

export type AvailabilityPrice = {
  amount: number;
  currency: string;
};

export type AvailabilityOption = {
  id: string;
  title: string;
  description?: string;
  availability: AvailabilityStatus;
  price?: AvailabilityPrice | null;
  priceType: AvailabilityPriceType;
  startTimes?: string[];
  durationMinutes?: number | null;
  bookingUrl?: string;
  providerName: string;
  cancellationPolicy?: string;
  instantConfirmation?: boolean | null;
  metadata?: Record<string, unknown>;
};

export type AvailabilityMatch = {
  matched: boolean;
  confidence: number;
  matchedName?: string;
};

export type AvailabilitySearchRequest = {
  destination: string;
  date: string;
  currency?: string;
  item: {
    name: string;
    type?: string;
    description?: string | null;
    startTime?: string;
    place?: {
      name?: string;
      address?: string;
      lat?: number | null;
      lng?: number | null;
      provider?: string;
      providerPlaceId?: string;
    } | null;
    estimatedCost?: EstimatedCost | null;
  };
  travelers?: {
    adults?: number;
    children?: number;
  };
};

export type AvailabilitySearchResponse = {
  status: AvailabilityStatus;
  result: AvailabilityProviderResult;
  provider: string;
  providerDisplayName: string;
  fallbackUsed: boolean;
  cached: boolean;
  checkedAt: string;
  cacheExpiresAt?: string | null;
  match: AvailabilityMatch;
  options: AvailabilityOption[];
  warnings?: string[];
  metadata?: Record<string, unknown>;
};

export type AvailabilityResultByItem = Record<string, AvailabilitySearchResponse>;

export type AvailabilityCardPriceApply = {
  trip: Trip;
  dayNumber: number;
  itemIndex: number;
  item: ItineraryItem;
  option: AvailabilityOption;
  result: AvailabilitySearchResponse;
};
