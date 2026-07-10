import type { GenerationJob } from "@/entities/generation-job/model";
import type { TripRoute } from "@/entities/route/model";
import type { Trip } from "@/entities/trip/model";

export type TripDiscoveryMode = "prompt" | "surprise" | "refine";
export type TripDiscoveryScope = "personal" | "workspace";
export type TripDiscoveryLanguage = "en" | "es" | "uk" | "fr";

export type TripDiscoveryBudget = {
  amount: number;
  currency: string;
};

export type TripDiscoverySuggestion = {
  id: string;
  suggestionType?: "single_destination" | "route";
  destination: string;
  city: string;
  country: string;
  region?: string | null;
  matchScore: number;
  recommendedDurationDays: number;
  bestFor: string[];
  estimatedBudget: TripDiscoveryBudget & {
    confidence: "low" | "medium" | "high";
  };
  bestTimeToGo: string;
  whyItFits: string;
  possibleDownsides: string[];
  tripPreview: {
    title: string;
    summary: string;
    sampleDay: string[];
  };
  tags: string[];
  suggestedPromptForItinerary: string;
  route?: TripRoute | null;
  concerns: Array<{ type: string; message: string }>;
};

export type TripDiscoveryResponse = {
  sessionTitle: string;
  suggestions: TripDiscoverySuggestion[];
  followUpQuestions: string[];
  warnings: string[];
};

export type TripDiscoverySession = {
  id: string;
  workspaceId?: string | null;
  parentSessionId?: string | null;
  mode: TripDiscoveryMode;
  prompt?: string;
  outputLanguage: TripDiscoveryLanguage;
  status: "completed" | "failed" | "created_trip";
  response: TripDiscoveryResponse;
  createdTripId?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type TripDiscoveryRequest = {
  prompt: string;
  scope: TripDiscoveryScope;
  workspaceId?: string | null;
  durationDays?: number;
  startDate?: string;
  dateFlexibility?: string;
  budget?: TripDiscoveryBudget;
  travelers?: number;
  origin?: string;
  quickChips?: string[];
  outputLanguage?: TripDiscoveryLanguage;
  avoidPreviouslyVisited?: boolean;
  preferNovelty?: boolean;
};

export type SurpriseMeRequest = Omit<TripDiscoveryRequest, "prompt" | "quickChips"> & {
  noveltyLevel?: "safe" | "balanced" | "adventurous";
};

export type RefineDiscoveryRequest = {
  instruction: string;
  selectedSuggestionId?: string;
  feedbackType?: string;
  outputLanguage?: TripDiscoveryLanguage;
};

export type CreateTripFromSuggestionRequest = {
  title?: string;
  startDate?: string;
  durationDays: number;
  budget?: TripDiscoveryBudget;
  travelers: number;
  workspaceId?: string | null;
  tripType?: "single_destination" | "multi_destination";
  route?: TripRoute | null;
  autoGenerateItinerary: boolean;
};

export type CreateTripFromSuggestionResponse = {
  trip: Trip;
  generationJob?: GenerationJob | null;
};

export type TripDiscoverySessionsResponse = {
  items: TripDiscoverySession[];
  limit: number;
};
