export type BudgetConfidenceLevel = "very_low" | "low" | "medium" | "high" | "very_high";

export type BudgetRiskLevel = "low" | "medium" | "high" | "critical";

export type BudgetConfidenceIssueSeverity = "info" | "warning" | "high" | "critical";

export type BudgetConfidenceCategory =
  | "transport"
  | "accommodation"
  | "activities"
  | "tickets"
  | "food"
  | "shopping"
  | "fuel"
  | "parking"
  | "tolls"
  | "groceries"
  | "camping"
  | "health_safety"
  | "other";

export type BudgetConfidenceSource =
  | "actual_receipt_expense"
  | "actual_manual_expense"
  | "provider_price"
  | "selected_transport_option_high_confidence"
  | "selected_transport_option_medium_confidence"
  | "selected_transport_option_low_confidence"
  | "manual_estimate"
  | "ai_estimate_high_confidence"
  | "ai_estimate_medium_confidence"
  | "ai_estimate_low_confidence"
  | "mock_estimate"
  | "missing_cost"
  | "unknown_source";

export type BudgetConfidenceMoney = {
  amount: number;
  currency: string;
};

export type BudgetConfidenceCoverage = {
  overall: number;
  transport: number | null;
  accommodation: number | null;
  activities: number | null;
  food: number | null;
  shopping: number | null;
  fuelParkingTolls: number | null;
  other: number | null;
};

export type BudgetConfidenceSourceQuality = {
  source: BudgetConfidenceSource;
  itemCount: number;
  totalAmount: BudgetConfidenceMoney;
  qualityScore: number;
  reason?: string | null;
};

export type BudgetConfidencePlannedVsActualCategory = {
  category: BudgetConfidenceCategory;
  estimated: BudgetConfidenceMoney;
  actual: BudgetConfidenceMoney;
  differencePercent?: number | null;
  status: string;
};

export type BudgetConfidencePlannedVsActual = {
  overallDifference: BudgetConfidenceMoney;
  overallDifferencePercent?: number | null;
  categories: BudgetConfidencePlannedVsActualCategory[];
};

export type BudgetConfidenceAction = {
  label: string;
  href: string;
};

export type BudgetConfidenceIssue = {
  id: string;
  severity: BudgetConfidenceIssueSeverity;
  category: string;
  title: string;
  description: string;
  recommendation: string;
  action?: BudgetConfidenceAction | null;
};

export type BudgetConfidenceRecommendation = {
  id: string;
  label: string;
  description: string;
  href: string;
  priority: "low" | "medium" | "high";
};

export type BudgetConfidence = {
  tripId: string;
  score: number;
  level: BudgetConfidenceLevel;
  riskLevel: BudgetRiskLevel;
  summary: string;
  currency: string;
  estimatedTotal: BudgetConfidenceMoney;
  actualTotal: BudgetConfidenceMoney;
  tripBudget?: BudgetConfidenceMoney | null;
  coverage: BudgetConfidenceCoverage;
  sourceQuality: BudgetConfidenceSourceQuality[];
  plannedVsActual: BudgetConfidencePlannedVsActual;
  issues: BudgetConfidenceIssue[];
  recommendations: BudgetConfidenceRecommendation[];
  warnings: string[];
  computedAt: string;
  debug?: Record<string, unknown>;
};
