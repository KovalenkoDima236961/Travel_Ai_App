import type {
  CostAnalyticsCategory,
  CostAnalyticsSource,
  CostInsight,
  ExchangeRateInfo,
  ExpensiveCostItem
} from "@/types/cost-analytics";

export type WorkspaceBudgetStatus = "active" | "archived";

export type WorkspaceBudget = {
  id: string;
  workspaceId: string;
  name: string;
  description?: string | null;
  amount: number;
  currency: string;
  periodStart?: string | null;
  periodEnd?: string | null;
  status: WorkspaceBudgetStatus;
  isPrimary: boolean;
  createdByUserId: string;
  archivedByUserId?: string | null;
  createdAt: string;
  updatedAt: string;
  archivedAt?: string | null;
};

export type WorkspaceBudgetSummaryMetrics = {
  tripCount: number;
  estimatedTotal: number;
  remainingAmount: number;
  overBudgetAmount: number;
  utilizationPercent: number;
  missingEstimateCount: number;
  uncertainEstimateCount: number;
  convertedItemCount: number;
  unconvertedItemCount: number;
};

export type WorkspaceBudgetByTrip = {
  tripId: string;
  title: string;
  destination: string;
  startDate?: string | null;
  estimatedTotal: number;
  percentageOfBudget: number;
  missingEstimateCount: number;
  overTripBudgetAmount?: number | null;
};

export type WorkspaceBudgetByCategory = {
  category?: CostAnalyticsCategory | string;
  amount: number;
  percentageOfBudget?: number;
  percentageOfEstimatedTotal: number;
  itemCount: number;
};

export type WorkspaceBudgetBySource = {
  source?: CostAnalyticsSource | string;
  amount: number;
  percentageOfBudget?: number;
  percentageOfEstimatedTotal: number;
  itemCount: number;
};

export type WorkspaceBudgetSummary = {
  budget: WorkspaceBudget;
  generatedAt: string;
  summary: WorkspaceBudgetSummaryMetrics;
  byTrip: WorkspaceBudgetByTrip[];
  byCategory: WorkspaceBudgetByCategory[];
  bySource: WorkspaceBudgetBySource[];
  expensiveItems: ExpensiveCostItem[];
  insights: CostInsight[];
  warnings: string[];
  exchangeRateInfo?: ExchangeRateInfo | null;
};

export type CreateWorkspaceBudgetInput = {
  name: string;
  description?: string | null;
  amount: number;
  currency: string;
  periodStart?: string | null;
  periodEnd?: string | null;
  isPrimary?: boolean;
};

export type UpdateWorkspaceBudgetInput = Partial<CreateWorkspaceBudgetInput>;

export type WorkspaceBudgetEnvelope = {
  budget: WorkspaceBudget;
};

export type WorkspaceBudgetsEnvelope = {
  budgets: WorkspaceBudget[];
};
