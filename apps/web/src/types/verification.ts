export type VerificationStatus =
  | "verified"
  | "needs_review"
  | "estimated"
  | "stale"
  | "missing"
  | "unavailable"
  | "failed"
  | "not_applicable";

export type VerificationSource =
  | "provider"
  | "manual"
  | "receipt"
  | "calendar_sync"
  | "ai"
  | "mock"
  | "fallback"
  | "heuristic"
  | "imported"
  | "unknown";

export type VerificationScope =
  | "transport"
  | "place"
  | "opening_hours"
  | "price"
  | "availability"
  | "weather"
  | "route_estimate"
  | "calendar_sync"
  | "accommodation"
  | "itinerary_item"
  | "budget"
  | "public_share"
  | "other";

export type VerificationActionType =
  | "refresh_weather"
  | "recheck_transport"
  | "check_availability"
  | "refresh_place_details"
  | "refresh_price"
  | "review_opening_hours"
  | "update_calendar_sync"
  | "add_accommodation"
  | "attach_place"
  | "open_route"
  | "open_budget"
  | "open_itinerary_item";

export type VerificationAction = {
  type: VerificationActionType;
  label: string;
  href: string;
};

export type VerificationDetail = {
  scope: VerificationScope;
  entityType: string;
  entityId: string;
  status: VerificationStatus;
  source: VerificationSource;
  provider?: string;
  checkedAt?: string;
  expiresAt?: string;
  confidence?: number;
  title: string;
  message: string;
  severity: "info" | "warning" | "high" | "critical";
  action?: VerificationAction | null;
  metadata?: Record<string, unknown>;
};

export type VerificationSection = {
  scope: VerificationScope;
  score: number;
  status: VerificationStatus;
  details: VerificationDetail[];
};

export type RealWorldReadiness = {
  tripId: string;
  score: number;
  level: "ready" | "mostly_ready" | "needs_review" | "not_ready";
  summary: {
    verifiedCount: number;
    needsReviewCount: number;
    estimatedCount: number;
    staleCount: number;
    missingCount: number;
    unavailableCount: number;
    failedCount: number;
  };
  sections: VerificationSection[];
  topIssues: VerificationDetail[];
  recommendedActions: VerificationAction[];
  computedAt: string;
};

export type RunVerificationActionInput = {
  actionType: VerificationActionType;
  scope: VerificationScope;
  entityType?: string;
  entityId?: string;
};

export type VerificationActionResult = {
  status: "completed" | "failed";
  message: string;
  updatedVerification: RealWorldReadiness;
};
