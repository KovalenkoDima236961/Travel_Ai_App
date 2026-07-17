export type PreferenceMissingField = { field: string; label: string; reason: string };
export type PreferenceCompleteness = {
  score: number;
  level: "excellent" | "good" | "partial" | "poor" | string;
  missingFields: PreferenceMissingField[];
  recommendedActions: Array<{ label: string; href: string }>;
};

export type FeedbackType =
  | "like"
  | "dislike"
  | "too_expensive"
  | "too_much_walking"
  | "too_packed"
  | "not_my_vibe"
  | "more_nature"
  | "more_food"
  | "less_museums"
  | "prefer_trains"
  | "avoid_nightlife"
  | "prefer_relaxed"
  | "prefer_fast_paced"
  | "too_far"
  | "too_many_transfers"
  | "other";

export type PersonalizationFeedbackInput = {
  workspaceId?: string | null;
  tripId?: string | null;
  entityType: "destination_suggestion" | "route_alternative" | "itinerary_item" | "template" | "budget_suggestion" | "checklist_item" | "general";
  entityId?: string;
  feedbackType: FeedbackType;
  feedbackValue?: string;
  metadata?: Record<string, string | string[]>;
};

export type FeedbackSummary = {
  likedDestinations: string[];
  dislikedDestinations: string[];
  likedStyles: string[];
  dislikedStyles: string[];
  tooExpensiveCount: number;
  tooMuchWalkingCount: number;
  preferTrainCount: number;
  budgetSensitivity: string;
  walkingSensitivity: string;
  recentFeedbackCount: number;
};

export type WhyThisFitsYou = {
  score: number;
  reasons: string[];
  concerns?: string[];
  signalsUsed?: string[];
};

export type PersonalizationContext = {
  schemaVersion: "personalization_v2" | string;
  completeness: PreferenceCompleteness;
  feedbackSignals: FeedbackSummary;
  explanationInputs: string[];
  warnings: string[];
};

export type BudgetSuggestion = {
  suggestedRange: { min: { amount: number; currency: string }; max: { amount: number; currency: string } };
  confidence: string;
  reasons: string[];
  categorySuggestions: Array<{ category: string; amountPerDay: { amount: number; currency: string }; reason: string }>;
};

export type RecommendedTemplate = {
  template: TripTemplate;
  fitScore: number;
  whyThisFitsYou: WhyThisFitsYou;
  fitTags: string[];
};
import type { TripTemplate } from "@/entities/trip-template/model";
