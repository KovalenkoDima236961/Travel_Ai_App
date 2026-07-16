export type ApprovalRiskLevel =
  | "low"
  | "medium"
  | "high"
  | "critical"
  | "unknown"
  | "not_applicable";

export type ApprovalRiskFactorSeverity = "low" | "medium" | "high" | "critical";

export type ApprovalRiskFactorSource =
  | "workspace_policy"
  | "approval_checklist"
  | "trip_budget"
  | "budget_confidence"
  | "workspace_budget"
  | "cost_analytics"
  | "cost_splitting"
  | "availability"
  | "ai_generation"
  | "template_adaptation"
  | "itinerary_quality"
  | "walking_distance"
  | "schedule"
  | "accommodation"
  | "route";

export type ApprovalRiskActionPriority = "low" | "medium" | "high";

export interface ApprovalRiskAffectedItem {
  dayNumber?: number | null;
  itemIndex?: number | null;
  name?: string;
  category?: string;
  amount?: number | null;
  currency?: string;
}

export interface ApprovalRiskAffectedTarget {
  tripId?: string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  category?: string;
  affectedCount?: number;
  affectedItems?: ApprovalRiskAffectedItem[];
}

export interface ApprovalRiskSuggestedActionTarget {
  tripId?: string;
  workspaceId?: string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  category?: string;
}

export interface ApprovalRiskSuggestedAction {
  type: string;
  label: string;
  priority?: ApprovalRiskActionPriority;
  target?: ApprovalRiskSuggestedActionTarget;
}

export interface ApprovalRiskFactor {
  type: string;
  severity: ApprovalRiskFactorSeverity;
  points: number;
  title: string;
  message: string;
  source: ApprovalRiskFactorSource;
  affected?: ApprovalRiskAffectedTarget | null;
  suggestedActions?: ApprovalRiskSuggestedAction[];
}

export interface ApprovalRiskSummaryCounts {
  factorCount: number;
  criticalFactorCount: number;
  highFactorCount: number;
  mediumFactorCount: number;
  lowFactorCount: number;
  blockingPolicyViolationCount: number;
  suggestedActionCount: number;
}

export interface ApprovalRiskQueueSummary {
  status: ApprovalRiskLevel;
  score: number | null;
  topReasons?: string[];
}

export interface ApprovalRiskResponse {
  tripId: string;
  workspaceId: string | null;
  status: ApprovalRiskLevel;
  score: number | null;
  maxScore: number;
  generatedAt: string;
  summary: ApprovalRiskSummaryCounts;
  factors: ApprovalRiskFactor[];
  topReasons: string[];
  suggestedActions: ApprovalRiskSuggestedAction[];
  warnings: string[];
  notApplicableReason: string | null;
}
