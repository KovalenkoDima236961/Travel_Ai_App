export type PolicySeverity = "info" | "warning" | "blocking";

export type PolicyRuleKey =
  | "requireTripBudget"
  | "maxTripBudget"
  | "maxDailyBudget"
  | "maxItemCost"
  | "maxAccommodationTotal"
  | "maxAccommodationPerNight"
  | "requireCostSplitting"
  | "requireAvailabilityForTicketedItems"
  | "maxWalkingKmPerDay"
  | "noLateActivitiesAfter"
  | "requiredRestTimePerDay"
  | "preferredTransportModes"
  | "disallowedActivityTypes";

export interface PolicyRule {
  enabled: boolean;
  severity: PolicySeverity;
}

export interface MoneyPolicyRule extends PolicyRule {
  amount: number;
  currency: string;
}

export interface WorkspacePolicyRules {
  requireTripBudget: PolicyRule;
  maxTripBudget: MoneyPolicyRule;
  maxDailyBudget: MoneyPolicyRule;
  maxItemCost: MoneyPolicyRule & { categories: string[] };
  maxAccommodationTotal: MoneyPolicyRule;
  maxAccommodationPerNight: MoneyPolicyRule;
  requireCostSplitting: PolicyRule;
  requireAvailabilityForTicketedItems: PolicyRule;
  maxWalkingKmPerDay: PolicyRule & { km: number };
  noLateActivitiesAfter: PolicyRule & { time: string };
  requiredRestTimePerDay: PolicyRule & { minutes: number };
  preferredTransportModes: PolicyRule & { modes: string[] };
  disallowedActivityTypes: PolicyRule & { types: string[] };
}

export interface WorkspacePolicyRuleDocument {
  schemaVersion: 1;
  rules: WorkspacePolicyRules;
}

export interface WorkspacePolicy {
  id: string;
  workspaceId: string;
  name: string;
  description: string | null;
  rules: WorkspacePolicyRuleDocument;
  status: "active" | "archived";
  createdByUserId: string;
  updatedByUserId: string | null;
  createdAt: string;
  updatedAt: string;
  archivedAt?: string;
  archivedByUserId?: string;
}

export interface UpsertWorkspacePolicyInput {
  name: string;
  description?: string | null;
  rules: WorkspacePolicyRuleDocument;
}

export interface WorkspacePolicyResponse {
  policy: WorkspacePolicy | null;
  defaults?: WorkspacePolicyRuleDocument;
}

export type PolicyEvaluationStatus =
  | "ok"
  | "info"
  | "warning"
  | "blocking"
  | "not_applicable";

export interface PolicySuggestedAction {
  type: string;
  label: string;
  dayNumber?: number;
  itemIndex?: number;
}

export interface PolicyAffectedItem {
  dayNumber?: number;
  itemIndex?: number;
  name?: string;
  amount?: number;
  currency?: string;
}

export interface PolicyEvaluationResult {
  ruleKey: PolicyRuleKey;
  status: "passed" | "violation" | "warning_unknown" | "info_unknown";
  severity: PolicySeverity;
  title: string;
  message: string;
  actual?: unknown;
  expected?: unknown;
  affectedItems: PolicyAffectedItem[];
  suggestedActions: PolicySuggestedAction[];
}

export interface PolicyEvaluation {
  tripId: string;
  workspaceId: string | null;
  policyId: string | null;
  status: PolicyEvaluationStatus;
  generatedAt: string;
  summary: {
    rulesChecked: number;
    passedCount: number;
    infoCount: number;
    warningCount: number;
    blockingCount: number;
  };
  results: PolicyEvaluationResult[];
  warnings: string[];
  notApplicableReason: string | null;
}
