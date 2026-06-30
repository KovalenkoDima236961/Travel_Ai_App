export type Budget = {
  amount: number;
  currency: string;
};

export type CostCategory =
  | "food"
  | "transport"
  | "ticket"
  | "activity"
  | "accommodation"
  | "shopping"
  | "other";

export type CostConfidence = "low" | "medium" | "high";

export type CostSource = "ai" | "manual" | "provider";

export type EstimatedCost = {
  amount?: number | null;
  currency?: string | null;
  category?: CostCategory | null;
  confidence?: CostConfidence | null;
  source?: CostSource | null;
  note?: string | null;
};

export type BudgetDaySummary = {
  dayNumber: number;
  estimatedTotal: number;
  missingEstimateCount: number;
  originalCurrencyTotals?: OriginalCurrencyTotal[];
  dailyBudgetShare?: number | null;
  overDailyBudgetBy?: number | null;
};

export type BudgetCategorySummary = {
  category: CostCategory;
  estimatedTotal: number;
  itemCount: number;
};

export type BudgetSummary = {
  currency: string;
  tripBudget?: number | null;
  estimatedTotal: number;
  remaining?: number | null;
  overBudgetBy?: number | null;
  accommodationTotal?: number | null;
  missingEstimateCount: number;
  estimatedItemCount: number;
  convertedItemCount?: number;
  unconvertedItemCount?: number;
  unsupportedCurrencyCount?: number;
  originalCurrencyTotals?: OriginalCurrencyTotal[];
  conversionWarnings?: ConversionWarning[];
  exchangeRateInfo?: ExchangeRateInfo | null;
  byDay: BudgetDaySummary[];
  byCategory: BudgetCategorySummary[];
};

export type OriginalCurrencyTotal = {
  currency: string;
  amount: number;
};

export type ConversionWarning = {
  currency: string;
  amount?: number | null;
  reason: string;
};

export type ExchangeRateInfo = {
  provider?: string | null;
  asOf?: string | null;
  fallbackUsed?: boolean;
};

export const COST_CATEGORIES: CostCategory[] = [
  "food",
  "transport",
  "ticket",
  "activity",
  "accommodation",
  "shopping",
  "other"
];

export const COST_CONFIDENCES: CostConfidence[] = ["low", "medium", "high"];
