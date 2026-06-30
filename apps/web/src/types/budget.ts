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
  missingEstimateCount: number;
  estimatedItemCount: number;
  unsupportedCurrencyCount?: number;
  byDay: BudgetDaySummary[];
  byCategory: BudgetCategorySummary[];
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
