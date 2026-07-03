export type CostAnalyticsCategory =
  | "food"
  | "transport"
  | "ticket"
  | "activity"
  | "accommodation"
  | "shopping"
  | "other"
  | "unknown";

export type CostAnalyticsSource =
  | "ai"
  | "manual"
  | "provider"
  | "availability"
  | "unknown";

export type CostAnalyticsConfidence = "low" | "medium" | "high" | "unknown";

export type InsightSeverity = "info" | "warning" | "critical";

export type CostInsightActionType =
  | "optimize_budget"
  | "check_availability"
  | "update_price"
  | "open_item"
  | "open_trip"
  | "export_report";

export type CostAnalyticsSummary = {
  budgetAmount?: number | null;
  estimatedTotal: number;
  remainingAmount?: number | null;
  overBudgetAmount?: number | null;
  budgetUtilizationPercent?: number | null;
  itemEstimatedTotal: number;
  accommodationTotal?: number | null;
  missingEstimateCount: number;
  uncertainEstimateCount: number;
  convertedItemCount: number;
  unconvertedItemCount: number;
  incompleteBudgetDataCount?: number;
};

export type WorkspaceAnalyticsSummary = {
  tripCount: number;
  estimatedTotal: number;
  budgetTotal?: number | null;
  overBudgetTripCount: number;
  missingEstimateCount: number;
  uncertainEstimateCount: number;
  convertedItemCount: number;
  unconvertedItemCount: number;
  incompleteBudgetTripCount: number;
};

export type CostByDay = {
  dayNumber: number;
  date?: string | null;
  estimatedTotal: number;
  budgetShare?: number | null;
  overBudgetAmount?: number | null;
  missingEstimateCount: number;
  topItems: ExpensiveCostItem[];
};

export type CostAmountBreakdown = {
  name?: string;
  category?: CostAnalyticsCategory;
  source?: CostAnalyticsSource;
  confidence?: CostAnalyticsConfidence;
  amount: number;
  percentage: number;
  itemCount: number;
};

export type OriginalCurrencyTotal = {
  currency: string;
  amount: number;
  convertedAmount?: number | null;
};

export type ExpensiveCostItem = {
  tripId?: string | null;
  tripTitle?: string;
  destination?: string;
  dayNumber?: number;
  itemIndex?: number;
  name: string;
  type: string;
  category: CostAnalyticsCategory;
  amount: number;
  currency: string;
  convertedAmount?: number | null;
  source: CostAnalyticsSource;
  confidence: CostAnalyticsConfidence;
  percentageOfTrip: number;
};

export type CostInsight = {
  type: string;
  severity: InsightSeverity;
  title: string;
  message: string;
  action?: CostInsightAction | null;
};

export type CostInsightAction = {
  type: CostInsightActionType;
  tripId?: string | null;
  dayNumber?: number | null;
  itemIndex?: number | null;
};

export type ExchangeRateInfo = {
  provider?: string | null;
  asOf?: string | null;
  fallbackUsed?: boolean;
};

export type TripCostAnalytics = {
  tripId: string;
  workspaceId?: string | null;
  currency: string;
  generatedAt: string;
  summary: CostAnalyticsSummary;
  byDay: CostByDay[];
  byCategory: CostAmountBreakdown[];
  bySource: CostAmountBreakdown[];
  byConfidence: CostAmountBreakdown[];
  originalCurrencyTotals: OriginalCurrencyTotal[];
  expensiveItems: ExpensiveCostItem[];
  insights: CostInsight[];
  warnings: string[];
  exchangeRateInfo?: ExchangeRateInfo | null;
};

export type TripCostSummary = {
  tripId: string;
  title: string;
  destination: string;
  startDate?: string | null;
  endDate?: string | null;
  budgetAmount?: number | null;
  estimatedTotal: number;
  overBudgetAmount?: number | null;
  missingEstimateCount: number;
  workspaceId: string;
};

export type CostByMonth = {
  month: string;
  estimatedTotal: number;
  tripCount: number;
};

export type WorkspaceCostAnalytics = {
  workspaceId: string;
  currency: string;
  generatedAt: string;
  dateRange: {
    from?: string | null;
    to?: string | null;
  };
  summary: WorkspaceAnalyticsSummary;
  byTrip: TripCostSummary[];
  byCategory: CostAmountBreakdown[];
  bySource: CostAmountBreakdown[];
  byMonth: CostByMonth[];
  expensiveTrips: TripCostSummary[];
  expensiveItems: ExpensiveCostItem[];
  insights: CostInsight[];
  warnings: string[];
};

export type WorkspaceCostAnalyticsParams = {
  currency?: string;
  from?: string | null;
  to?: string | null;
  includeArchived?: boolean;
};
