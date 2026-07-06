import type { TripAccommodation } from "@/types/accommodation";
import type { Budget, EstimatedCost } from "@/types/budget";
import type { Place } from "@/types/place";

export type TripStatus = "DRAFT" | "PROCESSING" | "COMPLETED" | "FAILED";

export type Pace = "relaxed" | "balanced" | "packed" | "intensive" | string;

export type TripScope = "personal" | "workspace";

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

export type ItineraryItem = {
  time: string;
  type: "place" | "food" | "activity" | "transport" | "rest" | string;
  name: string;
  note?: string | null;
  estimatedCost?: EstimatedCost | null;
  place?: Place | null;
  placeEnrichment?: PlaceEnrichment | null;
  priceEnrichment?: PriceEnrichment | null;
  availabilityCheck?: AvailabilityCheckMeta | null;
};

export type ItineraryDay = {
  day: number;
  title: string;
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
  destination: string;
  startDate?: string | null;
  days: number;
  budgetAmount?: number | null;
  budgetCurrency: string;
  budget?: Budget | null;
  accommodation?: TripAccommodation | null;
  travelers: number;
  interests: string[];
  pace: Pace;
  status: TripStatus;
  itinerary?: Itinerary | null;
  itineraryRevision: number;
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
  workspaceId?: string | null;
  startDate?: string;
  days: number;
  budgetAmount?: number;
  budgetCurrency: string;
  travelers: number;
  interests: string[];
  pace: "relaxed" | "balanced" | "packed";
};
