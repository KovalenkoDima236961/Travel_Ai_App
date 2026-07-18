import type { TripAccommodation } from "@/entities/accommodation/model";
import type { Budget, EstimatedCost } from "@/entities/budget/model";
import type { Place } from "@/entities/place/model";
import type { TransportMode, TripRoute } from "@/entities/route/model";

export type TripStatus = "DRAFT" | "PROCESSING" | "COMPLETED" | "FAILED";

export type Pace = "relaxed" | "balanced" | "packed" | "intensive" | string;

export type TripScope = "personal" | "workspace";

export type TripType = "single_destination" | "multi_destination";
export type TripLifecycle = "draft" | "planning" | "ready" | "active" | "completed" | "archived";

export type PlaceEnrichmentReviewStatus = "pending" | "accepted" | "changed" | "removed";

export type PlaceEnrichmentMeta = {
  status: "matched" | "no_match" | "skipped" | "failed";
  reviewStatus?: PlaceEnrichmentReviewStatus | null;
  confidence?: number | null;
  query?: string | null;
  provider?: string | null;
  matchedAt?: string | null;
  reason?: string | null;
};

export type PlaceEnrichment = PlaceEnrichmentMeta;

export type PriceEnrichmentReviewStatus = "pending" | "accepted" | "changed" | "removed";

export type PriceEnrichmentMeta = {
  status: "matched" | "no_match" | "skipped" | "failed";
  provider?: string | null;
  matchConfidence?: number | null;
  priceType?: string | null;
  reviewStatus?: PriceEnrichmentReviewStatus | null;
  updatedAt?: string | null;
  reason?: string | null;
};

export type PriceEnrichment = PriceEnrichmentMeta;

// AvailabilityCheckMeta is a lightweight snapshot persisted on an item when a
// user applies a provider availability result. Mirrors the Trip Service
// aggregate.AvailabilityCheckMeta and feeds the approval checklist signals.
export type AvailabilityCheckMeta = {
  provider?: string;
  status?: "available" | "limited" | "unavailable" | "unknown" | string;
  checkedAt?: string;
  matchConfidence?: number;
  selectedOptionId?: string;
  fallbackUsed?: boolean;
  priceChanged?: boolean;
};

export type TravelStatus = "planned" | "done" | "skipped" | "delayed";

export type ItineraryTravelStatus = {
  status: TravelStatus;
  updatedAt?: string;
  updatedByUserId?: string;
  note?: string;
};

export type ItineraryItem = {
  time: string;
  endTime?: string | null;
  type: "place" | "food" | "activity" | "transport" | "transfer" | "rest" | string;
  category?: string | null;
  transportMode?: TransportMode | string | null;
  durationMinutes?: number | null;
  walkingDistanceKm?: number | null;
  name: string;
  description?: string | null;
  note?: string | null;
  transfer?: {
    legId?: string | null;
    from: string;
    to: string;
    mode: TransportMode | string;
    estimatedDurationMinutes?: number | null;
    estimatedDistanceKm?: number | null;
    estimatedCost?: EstimatedCost | null;
    bookingRequired?: boolean;
    notes?: string | null;
    warnings?: string[];
  } | null;
  estimatedCost?: EstimatedCost | null;
  place?: Place | null;
  placeEnrichment?: PlaceEnrichment | null;
  priceEnrichment?: PriceEnrichment | null;
  availabilityCheck?: AvailabilityCheckMeta | null;
  travelStatus?: ItineraryTravelStatus | null;
};

export type ItineraryDay = {
  day: number;
  date?: string | null;
  title: string;
  primaryStopId?: string | null;
  locationName?: string | null;
  transferDay?: boolean;
  items: ItineraryItem[];
};

export type Itinerary = {
  destination?: string;
  summary?: string;
  travelers?: number;
  pace?: string;
  currency?: string;
  totalBudget?: number | null;
  generatedAt?: string;
  source?: string;
  days: ItineraryDay[];
};

export type Trip = {
  id: string;
  userId?: string | null;
  workspaceId?: string | null;
  scope?: TripScope;
  tripType?: TripType;
  route?: TripRoute | null;
  destination: string;
  startDate?: string | null;
  days: number;
  budgetAmount?: number | null;
  budgetCurrency: string;
  budget?: Budget | null;
  accommodation?: TripAccommodation | null;
  creationMetadata?: Record<string, unknown>;
  travelers: number;
  interests: string[];
  pace: Pace;
  status: TripStatus;
  itinerary?: Itinerary | null;
  itineraryRevision: number;
  lifecycle?: TripLifecycle;
  archivedAt?: string | null;
  archivedByUserId?: string | null;
  access?: TripAccess | null;
  createdAt: string;
  updatedAt: string;
};

export type TripAccess = {
  role: "owner" | "editor" | "viewer";
  source?: "owner" | "workspace" | "collaborator" | "public";
  canEdit: boolean;
  canManageCollaborators: boolean;
  canManageShare: boolean;
  canRestoreVersion: boolean;
  canDelete: boolean;
};

export type TripsListResponse = {
  items: Trip[];
  limit: number;
  offset: number;
};

export type CreateTripInput = {
  destination: string;
  tripType?: TripType;
  route?: TripRoute | null;
  workspaceId?: string | null;
  startDate?: string;
  days: number;
  budgetAmount?: number;
  budgetCurrency: string;
  travelers: number;
  interests: string[];
  pace: "relaxed" | "balanced" | "packed";
};
