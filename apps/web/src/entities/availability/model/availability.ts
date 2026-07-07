import type { EstimatedCost } from "@/entities/budget/model";
import type { ItineraryItem, Trip } from "@/entities/trip/model";

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

// Qualifier describes how precise a price is, orthogonal to priceType (the unit).
export type AvailabilityPriceQualifier = "exact" | "from" | "estimate" | "unknown";

export type AvailabilityPrice = {
  amount: number;
  currency: string;
  qualifier?: AvailabilityPriceQualifier;
};

export type AvailabilityLocation = {
  name?: string;
  address?: string;
  lat?: number | null;
  lng?: number | null;
};

export type AvailabilityOption = {
  id: string;
  title: string;
  description?: string;
  availability: AvailabilityStatus;
  price?: AvailabilityPrice | null;
  priceType: AvailabilityPriceType;
  startTimes?: string[];
  date?: string;
  durationMinutes?: number | null;
  bookingUrl?: string;
  providerName: string;
  providerEntityId?: string;
  location?: AvailabilityLocation | null;
  matchConfidence?: number;
  cancellationPolicy?: string;
  instantConfirmation?: boolean | null;
  warnings?: string[];
  metadata?: Record<string, unknown>;
};

export type AvailabilityMatch = {
  matched: boolean;
  confidence: number;
  matchedName?: string;
  providerEntityId?: string;
  providerUrl?: string;
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
