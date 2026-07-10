import type { EstimatedCost } from "@/entities/budget/model";

export type TransportMode =
  | "walk"
  | "walking"
  | "car"
  | "driving"
  | "rental_car"
  | "train"
  | "bus"
  | "flight"
  | "boat"
  | "ferry"
  | "bike"
  | "cycling"
  | "public_transport"
  | "hiking"
  | "other";

export type RouteMode = TransportMode;

export type TripStyle =
  | "city_break"
  | "road_trip"
  | "train_trip"
  | "backpacking"
  | "camping"
  | "hiking"
  | "island_hopping"
  | "nature"
  | "beach"
  | "food"
  | "culture"
  | "adventure"
  | "family"
  | "romantic"
  | "low_budget"
  | "luxury"
  | "hidden_gem";

export type AccommodationHint =
  | "hotel"
  | "hostel"
  | "apartment"
  | "guesthouse"
  | "campsite"
  | "cabin"
  | "campervan"
  | "home"
  | "other"
  | "unknown";

export type Coordinates = {
  lat: number;
  lng: number;
};

export type RoutePlace = {
  name?: string | null;
  country?: string | null;
  coordinates?: Coordinates | null;
};

export type RouteStop = {
  name: string;
  latitude: number;
  longitude: number;
};

export type RouteSegment = {
  fromName: string;
  toName: string;
  distanceKm: number;
  estimatedDistanceKm?: number;
  durationMinutes: number;
  estimatedDurationMinutes?: number;
  estimatedCost?: EstimatedCost | null;
};

export type RouteEstimate = {
  mode: RouteMode;
  provider: string;
  distanceKm: number;
  estimatedDistanceKm?: number;
  durationMinutes: number;
  estimatedDurationMinutes?: number;
  estimatedCost?: EstimatedCost | null;
  segments: RouteSegment[];
  fallbackUsed?: boolean;
  warnings?: string[];
};

export type RouteEstimateRequest = {
  mode: RouteMode;
  stops?: RouteStop[];
  from?: RouteEstimatePoint;
  to?: RouteEstimatePoint;
  date?: string;
  currency?: string;
};

export type RouteEstimatePoint = {
  name: string;
  latitude?: number;
  longitude?: number;
  lat?: number;
  lng?: number;
};

export type TripRouteStop = {
  id: string;
  destination: string;
  city?: string | null;
  country?: string | null;
  arrivalDate?: string | null;
  departureDate?: string | null;
  nights?: number | null;
  coordinates?: Coordinates | null;
  accommodationHint?: AccommodationHint | null;
  notes?: string | null;
};

export type TripRouteLeg = {
  id: string;
  fromStopId: string;
  toStopId: string;
  fromName?: string | null;
  toName?: string | null;
  mode: TransportMode;
  departureDate?: string | null;
  estimatedDurationMinutes?: number | null;
  estimatedDistanceKm?: number | null;
  estimatedCost?: EstimatedCost | null;
  notes?: string | null;
  providerMetadata?: Record<string, unknown> | null;
};

export type TripRoutePreferences = {
  preferredModes?: TransportMode[];
  avoidModes?: TransportMode[];
  carAvailable?: boolean;
  maxTransferHoursPerDay?: number | null;
  tripStyles?: TripStyle[];
};

export type TripRoute = {
  origin?: RoutePlace | null;
  returnToOrigin?: boolean;
  stops: TripRouteStop[];
  legs?: TripRouteLeg[];
  preferences?: TripRoutePreferences;
};

export type RouteValidationWarning = {
  code: string;
  message: string;
  severity?: "info" | "warning" | "error";
};
