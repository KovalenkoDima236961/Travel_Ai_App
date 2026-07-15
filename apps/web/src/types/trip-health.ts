export type TripHealthLevel = "ready" | "almost_ready" | "needs_attention" | "not_ready";

export type TripHealthIssueSeverity = "info" | "warning" | "high" | "critical";

export type TripHealthCategory =
  | "itinerary"
  | "route"
  | "transport"
  | "budget"
  | "availability"
  | "collaboration"
  | "checklist"
  | "reminders"
  | "accommodation"
  | "expenses"
  | "policy"
  | "approval"
  | "offline"
  | "data_quality"
  | "public_share"
  | "other";

export type TripHealthIssueStatus = "open" | "resolved" | "ignored";

export type TripHealthAction = {
  type: string;
  label: string;
  href: string;
};

export type TripHealthIssue = {
  id: string;
  category: TripHealthCategory;
  severity: TripHealthIssueSeverity;
  status: TripHealthIssueStatus;
  title: string;
  description: string;
  impact?: string;
  recommendation?: string;
  action?: TripHealthAction | null;
  metadata?: Record<string, unknown>;
};

export type TripHealthCategorySummary = {
  category: TripHealthCategory;
  score: number;
  openIssueCount: number;
  highestSeverity: TripHealthIssueSeverity;
};

export type TripHealthTopFix = {
  issueId: string;
  label: string;
  href: string;
};

export type TripHealthComputedFrom = {
  itineraryRevision: number;
  routeUpdatedAt?: string;
  budgetUpdatedAt?: string;
  checklistUpdatedAt?: string;
  remindersUpdatedAt?: string;
};

export type TripHealth = {
  tripId: string;
  score: number;
  level: TripHealthLevel;
  summary: string;
  generatedAt: string;
  categories: TripHealthCategorySummary[];
  issues: TripHealthIssue[];
  topFixes: TripHealthTopFix[];
  computedFrom: TripHealthComputedFrom;
  debug?: Record<string, unknown>;
};
