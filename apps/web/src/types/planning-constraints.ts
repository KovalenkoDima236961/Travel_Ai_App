import type { TripRoute, TransportMode, TripStyle } from "@/entities/route/model";

export type PlanningSource =
  | "trip_discovery"
  | "trip_generation"
  | "day_regeneration"
  | "item_regeneration"
  | "template_adaptation"
  | "policy_repair"
  | "budget_optimization"
  | "route_generation"
  | "route_update_preview";

export type ConstraintSeverity = "info" | "warning" | "blocking";

export type ConstraintSuggestedAction = {
  type:
    | "change_transport_mode"
    | "remove_disallowed_mode"
    | "increase_budget"
    | "reduce_stops"
    | "increase_duration"
    | "lower_pace"
    | "disable_hiking"
    | "enable_car_available"
    | "change_accommodation_type"
    | "adjust_workspace_policy"
    | "open_route_builder"
    | "open_budget_settings"
    | "open_preferences"
    | "repair_with_ai"
    | string;
  label: string;
};

export type PlanningConstraintIssue = {
  type: string;
  severity: ConstraintSeverity;
  message: string;
  source: string;
  affected?: Record<string, unknown>;
  suggestedActions: ConstraintSuggestedAction[];
};

export type PlanningConstraints = {
  schemaVersion: 1;
  language: "en" | "es" | "uk" | "fr" | string;
  scope: "personal" | "workspace";
  workspaceId?: string | null;
  source: PlanningSource;
  profile: {
    homeCity?: string;
    homeCountry?: string;
    preferredCurrency?: string;
  };
  budget?: {
    amount?: number | null;
    currency: string;
    strictness: "loose" | "target" | "strict";
  } | null;
  dates: {
    startDate?: string;
    endDate?: string;
    durationDays?: number;
    flexibility: "fixed" | "flexible" | "weekend" | "month" | "unknown";
  };
  travelers: {
    count: number;
    type?: string;
  };
  pace: "relaxed" | "balanced" | "packed" | string;
  walking: {
    maxKmPerDay?: number | null;
    allowLongHikes: boolean;
  };
  transport: {
    preferredModes: TransportMode[];
    allowedModes: TransportMode[];
    avoidModes: TransportMode[];
    disallowedModes: TransportMode[];
    carAvailable: boolean;
    maxTransferHoursPerDay?: number | null;
  };
  tripStyles: TripStyle[];
  accommodation: {
    preferredTypes: string[];
    avoidTypes: string[];
    campingAllowed: boolean;
  };
  interests: string[];
  avoid: string[];
  mustHave: string[];
  accessibility: {
    lowWalkingRequired: boolean;
    stepFreePreferred: boolean;
    notes?: string;
  };
  food: {
    preferences: string[];
    dietaryRestrictions: string[];
  };
  route?: Record<string, unknown> | null;
  workspacePolicy?: {
    policyId?: string;
    summary?: string;
    blockingRules: string[];
    warningRules: string[];
    rules?: Record<string, unknown>;
  } | null;
  previousTripSignals?: {
    visitedDestinations: string[];
    likedStyles: string[];
    typicalDurationDays?: number;
    typicalBudget?: {
      amount?: number | null;
      currency: string;
      strictness: "loose" | "target" | "strict";
    } | null;
  } | null;
  prompt?: {
    userPrompt?: string;
    quickChips: string[];
    refinementInstruction?: string;
  } | null;
  warnings: PlanningConstraintIssue[];
  blockers: PlanningConstraintIssue[];
};

export type PlanningConstraintSummary = {
  language: string;
  budget: string;
  pace: string;
  transport: string;
  tripStyles: string[];
  workspacePolicyRules: number;
  warningCount: number;
  blockerCount: number;
};

export type PlanningConstraintsPreviewRequest = {
  source: PlanningSource;
  tripId?: string;
  workspaceId?: string | null;
  request?: {
    tripType?: "single_destination" | "multi_destination";
    destination?: string;
    outputLanguage?: "en" | "es" | "uk" | "fr" | string;
    startDate?: string;
    endDate?: string;
    durationDays?: number;
    dateFlexibility?: "fixed" | "flexible" | "weekend" | "month" | "unknown";
    budget?: {
      amount?: number | null;
      currency?: string;
      strictness?: "loose" | "target" | "strict";
    } | null;
    travelers?: {
      count?: number;
      type?: string;
    };
    pace?: "relaxed" | "balanced" | "packed" | string;
    walking?: {
      maxKmPerDay?: number | null;
      allowLongHikes?: boolean;
    };
    transport?: {
      preferredModes?: TransportMode[];
      allowedModes?: TransportMode[];
      avoidModes?: TransportMode[];
      disallowedModes?: TransportMode[];
      carAvailable?: boolean;
      maxTransferHoursPerDay?: number | null;
    };
    route?: TripRoute | null;
    tripStyles?: TripStyle[];
    accommodation?: {
      preferredTypes?: string[];
      avoidTypes?: string[];
      campingAllowed?: boolean;
    };
    interests?: string[];
    avoid?: string[];
    mustHave?: string[];
    prompt?: {
      userPrompt?: string;
      quickChips?: string[];
      refinementInstruction?: string;
    };
  };
  includePreviousTripSignals?: boolean;
  includeWorkspacePolicy?: boolean;
  includeRoute?: boolean;
  includeTripState?: boolean;
};

export type PlanningConstraintsPreviewResponse = {
  constraints: PlanningConstraints;
  summary: PlanningConstraintSummary;
  warnings: PlanningConstraintIssue[];
  blockers: PlanningConstraintIssue[];
};

