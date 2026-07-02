export type QualityIssueSeverity = "info" | "warning" | "critical";

export type QualityIssueScope = "trip" | "day" | "item";

export type QualityIssueType =
  | "walking_distance_high"
  | "place_may_be_closed"
  | "weather_rain_outdoor"
  | "weather_heat_outdoor"
  | "place_match_pending_review"
  | "place_match_low_confidence"
  | "place_no_confident_match"
  | "missing_place_coordinates"
  | "missing_accommodation"
  | "trip_budget_exceeded"
  | "day_budget_high"
  | "expensive_item"
  | "missing_cost_estimate"
  | "missing_ticket_price"
  | "high_ticket_cost"
  | "provider_price_low_confidence"
  | "conversion_unavailable";

export type QualityIssue = {
  id: string;
  type: QualityIssueType;
  severity: QualityIssueSeverity;
  scope: QualityIssueScope;
  dayNumber?: number;
  itemIndex?: number;
  title: string;
  message: string;
  suggestion: string;
  instructionHint: string;
  metadata?: Record<string, unknown>;
};

export type QualitySummary = {
  total: number;
  critical: number;
  warning: number;
  info: number;
  byDay: Record<number, QualityIssue[]>;
  itemIssues: QualityIssue[];
  tripIssues: QualityIssue[];
};
