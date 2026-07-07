import type { Itinerary } from "@/entities/trip/model";

export type ItineraryVersionSource =
  | "GENERATED"
  | "MANUAL_EDIT"
  | "REGENERATE_DAY"
  | "REGENERATE_ITEM"
  | "RESTORED";

export type ItineraryVersionSummary = {
  id: string;
  tripId: string;
  versionNumber: number;
  source: ItineraryVersionSource;
  metadata?: Record<string, unknown> | null;
  createdByUserId?: string | null;
  createdAt: string;
};

export type ItineraryVersionDetail = ItineraryVersionSummary & {
  itinerary: Itinerary;
};

export type ListItineraryVersionsResponse = {
  items: ItineraryVersionSummary[];
  limit: number;
  offset: number;
};
