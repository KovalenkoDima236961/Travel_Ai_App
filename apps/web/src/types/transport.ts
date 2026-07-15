export type TransportModeValue =
  | "train"
  | "bus"
  | "flight"
  | "ferry"
  | "car"
  | "rental_car"
  | "public_transport"
  | "walk"
  | "bike"
  | "hiking"
  | "boat"
  | "other";

export type TransportProvider =
  | "mock"
  | "route_estimate"
  | "gtfs_static"
  | "amadeus"
  | "skyscanner"
  | "rome2rio"
  | "national_rail"
  | "ferry_provider"
  | "manual"
  | string;

export type TransportStatus = "available" | "limited" | "unknown" | "unavailable";
export type TransportConfidence = "low" | "medium" | "high";
export type TransportTimePreference = "depart_after" | "arrive_before" | "flexible";

export type TransportLocation = {
  name: string;
  lat?: number | null;
  lng?: number | null;
  country?: string | null;
  stopId?: string | null;
};

export type TransportMoney = {
  amount: number;
  currency: string;
};

export type TransportMoneyRange = {
  min: TransportMoney;
  max: TransportMoney;
};

export type TransportSearchConstraints = {
  maxDurationMinutes?: number | null;
  maxPriceAmount?: number | null;
  avoidFlights?: boolean;
  preferredModes?: TransportModeValue[];
  accessibilityNotes?: string | null;
};

export type SearchRouteLegTransportInput = {
  date?: string;
  time?: string;
  timePreference?: TransportTimePreference | "";
  modes?: TransportModeValue[];
  travelers?: number;
  currency?: string;
  constraints?: TransportSearchConstraints;
};

export type TransportOption = {
  id: string;
  mode: TransportModeValue;
  provider: TransportProvider;
  operatorName?: string | null;
  serviceName?: string | null;
  originName?: string | null;
  destinationName?: string | null;
  departureDate?: string | null;
  departureTime?: string | null;
  arrivalDate?: string | null;
  arrivalTime?: string | null;
  durationMinutes: number;
  transfers: number;
  estimatedPrice?: TransportMoney | null;
  priceRange?: TransportMoneyRange | null;
  bookingUrl?: string | null;
  providerUrl?: string | null;
  status: TransportStatus;
  confidence: TransportConfidence;
  emissionsEstimateKg?: number | null;
  baggageNotes?: string | null;
  accessibilityNotes?: string | null;
  warnings?: string[];
  metadata?: Record<string, unknown> | null;
};

export type SelectedTransportOption = {
  id: string;
  mode: TransportModeValue;
  provider: TransportProvider;
  operatorName?: string | null;
  serviceName?: string | null;
  originName?: string | null;
  destinationName?: string | null;
  departureDate?: string | null;
  departureTime?: string | null;
  arrivalDate?: string | null;
  arrivalTime?: string | null;
  durationMinutes?: number | null;
  transfers?: number | null;
  estimatedPrice?: TransportMoney | null;
  bookingUrl?: string | null;
  providerUrl?: string | null;
  status?: TransportStatus | null;
  confidence?: TransportConfidence | null;
  baggageNotes?: string | null;
  accessibilityNotes?: string | null;
  warnings?: string[];
  selectedAt?: string | null;
  selectedByUserId?: string | null;
};

export type TransportSearchSummary = {
  origin: string;
  destination: string;
  date: string;
  searchedModes: TransportModeValue[];
  provider: TransportProvider;
  fallbackUsed?: boolean;
  cached?: boolean;
  warnings?: string[];
};

export type TransportSearchResponse = {
  options: TransportOption[];
  summary: TransportSearchSummary;
};

export type AttachRouteLegTransportOptionInput = {
  expectedItineraryRevision?: number;
  option: SelectedTransportOption;
  updateLegMode?: boolean;
};

export type RemoveRouteLegTransportOptionInput = {
  expectedItineraryRevision?: number;
  resetLegMode?: boolean;
};
