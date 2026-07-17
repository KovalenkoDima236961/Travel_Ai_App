import type { GenerationJob } from "@/entities/generation-job/model";
import type { RoutePlace, TransportMode, TripRoute, TripStyle } from "@/entities/route/model";
import type { Trip } from "@/entities/trip/model";
import type { TripPoll } from "@/types/trip-decisions";

export type RouteAlternativeDifficulty = "relaxed" | "balanced" | "intense" | "rushed";

export type RouteAlternativeBudgetEstimate = {
  amount?: number | null;
  currency: string;
  confidence?: string | null;
};

export type RouteAlternativeScores = {
  overallFit: number;
  budgetFit: number;
  timeEfficiency: number;
  relaxation: number;
  nature: number;
  culture: number;
  transportSimplicity: number;
  policyCompliance: number;
};

export type RouteAlternative = {
  id: string;
  title: string;
  summary: string;
  route: TripRoute;
  scores: RouteAlternativeScores;
  estimatedBudget?: RouteAlternativeBudgetEstimate | null;
  estimatedTransferMinutes?: number | null;
  estimatedTransferCost?: RouteAlternativeBudgetEstimate | null;
  difficulty: RouteAlternativeDifficulty;
  bestFor: string[];
  pros: string[];
  cons: string[];
  warnings: string[];
  suggestedItineraryPrompt?: string;
  personalizationFit?: { score: number; reasons: string[]; concerns: string[] } | null;
};

export type RouteAlternativeComparisonSummary = {
  cheapestAlternativeId?: string;
  mostRelaxedAlternativeId?: string;
  bestNatureAlternativeId?: string;
  bestOverallAlternativeId?: string;
};

export type RouteAlternativeSession = {
  id: string;
  userId: string;
  tripId?: string | null;
  workspaceId?: string | null;
  source: "pre_trip" | "existing_trip" | "discovery_refinement" | "route_refinement" | string;
  prompt?: string;
  outputLanguage: string;
  status: "completed" | "failed" | "created_trip" | "applied" | "archived" | string;
  selectedAlternativeId?: string;
  createdTripId?: string | null;
  appliedToTripId?: string | null;
  parentSessionId?: string | null;
  sessionTitle: string;
  alternatives: RouteAlternative[];
  comparisonSummary: RouteAlternativeComparisonSummary;
  followUpQuestions: string[];
  warnings: string[];
  createdAt: string;
  updatedAt: string;
};

export type RouteAlternativeSessionsResponse = {
  items: RouteAlternativeSession[];
  limit: number;
};

export type SuggestRouteAlternativesInput = {
  origin?: RoutePlace | null;
  prompt: string;
  durationDays: number;
  startDate?: string;
  budget?: RouteAlternativeBudgetEstimate | null;
  travelers?: number;
  workspaceId?: string | null;
  transport?: {
    preferredModes?: TransportMode[];
    avoidModes?: TransportMode[];
    carAvailable?: boolean;
    maxTransferHoursPerDay?: number | null;
  };
  tripStyles?: TripStyle[];
  outputLanguage?: string;
  suggestionCount?: number;
};

export type SuggestTripRouteAlternativesInput = {
  prompt?: string;
  suggestionCount?: number;
  useCurrentRouteAsBaseline?: boolean;
  outputLanguage?: string;
};

export type RefineRouteAlternativesInput = {
  instruction: string;
  selectedAlternativeId?: string;
};

export type CreateTripFromRouteAlternativeInput = {
  title: string;
  startDate?: string;
  budget?: RouteAlternativeBudgetEstimate | null;
  travelers?: number;
  workspaceId?: string | null;
  autoGenerateItinerary?: boolean;
};

export type CreateTripFromRouteAlternativeResult = {
  trip: Trip;
  generationJob?: GenerationJob;
};

export type ApplyRouteAlternativeInput = {
  expectedItineraryRevision?: number;
  regenerateItinerary?: boolean;
};

export type CreateRouteAlternativesPollInput = {
  title?: string;
  alternativeIds?: string[];
};

export type RouteAlternativeVote = {
  vote: "like" | "dislike" | "favorite" | "not_interested";
  metadata?: Record<string, unknown>;
};

export type CreateRouteAlternativesPollResult = TripPoll;
