import type { TripAccommodation } from "@/entities/accommodation/model";
import type { BudgetSummary } from "@/entities/budget/model";
import type { Itinerary, Trip } from "@/entities/trip/model";

export type OfflineMutationStatus =
  | "pending"
  | "syncing"
  | "conflict"
  | "failed"
  | "synced"
  | "discarded";

export type OfflineMutationType = "update_itinerary";

export type CachedTripRecord = {
  tripId: string;
  userId: string;
  trip: Trip;
  budgetSummary?: BudgetSummary | null;
  accommodation?: TripAccommodation | null;
  itineraryRevision: number;
  cachedAt: string;
};

export type PendingItineraryMutation = {
  mutationId: string;
  type: "update_itinerary";
  tripId: string;
  userId: string;
  baseRevision: number;
  baseItinerary: Itinerary;
  draftItinerary: Itinerary;
  status: OfflineMutationStatus;
  createdAt: string;
  updatedAt: string;
  lastAttemptAt?: string | null;
  errorCode?: string | null;
  errorMessage?: string | null;
};

export type SyncMetadataRecord = {
  key: string;
  userId?: string | null;
  value: unknown;
  updatedAt: string;
};

export type EnqueueItineraryUpdateInput = {
  tripId: string;
  userId: string;
  baseRevision: number;
  baseItinerary: Itinerary;
  draftItinerary: Itinerary;
};

export type SyncResult =
  | {
      status: "synced";
      mutation: PendingItineraryMutation;
      trip: Trip;
    }
  | {
      status: "conflict";
      mutation: PendingItineraryMutation;
      currentItineraryRevision?: number | null;
      latestTrip?: Trip | null;
      errorMessage?: string | null;
    }
  | {
      status: "failed";
      mutation: PendingItineraryMutation;
      retryable: boolean;
      errorCode?: string | null;
      errorMessage?: string | null;
    };
