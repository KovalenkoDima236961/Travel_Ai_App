import type { Itinerary } from "@/entities/trip/model";
import type { GenerationQuality } from "@/types/generation-quality";

export type ItineraryVersionSource =
  | "GENERATED"
  | "MANUAL_EDIT"
  | "REGENERATE_DAY"
  | "REGENERATE_ITEM"
  | "BUDGET_OPTIMIZATION_APPLIED"
  | "AI_POLICY_REPAIR"
  | "COST_SPLIT_UPDATED"
  | "CREATED_FROM_TEMPLATE"
  | "CREATED_FROM_TEMPLATE_AI"
  | "RESTORED";

export type ItineraryVersionSummary = {
  id: string;
  tripId: string;
  versionNumber: number;
  source: ItineraryVersionSource;
  metadata?: ({ generationQuality?: GenerationQuality | null } & Record<string, unknown>) | null;
  generationQuality?: GenerationQuality | null;
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
