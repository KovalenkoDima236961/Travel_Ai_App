import type { ItineraryDay, Trip } from "@/types/trip";

export type BudgetOptimizationScope = "day";

export type BudgetOptimizationChangeType =
  | "replace_item"
  | "remove_item"
  | "add_item"
  | "modify_item_cost"
  | "reorder_item"
  | "keep_item";

export type BudgetOptimizationProposalStatus =
  | "pending"
  | "applied"
  | "discarded"
  | "expired"
  | "failed";

export type BudgetOptimizationConstraints = {
  preserveMustSeeItems?: boolean;
  maxWalkingIncreaseKm?: number | null;
  keepMealCount?: boolean;
  avoidReplacingManualCosts?: boolean;
};

export type BudgetOptimizationJobRequest = {
  scope: BudgetOptimizationScope;
  dayNumber: number;
  targetReductionAmount?: number | null;
  currency?: string | null;
  expectedItineraryRevision: number;
  constraints?: BudgetOptimizationConstraints | null;
  instruction?: string | null;
};

export type BudgetOptimizationChange = {
  type: BudgetOptimizationChangeType;
  oldItemIndex?: number | null;
  oldItemName?: string | null;
  newItemIndex?: number | null;
  newItemName?: string | null;
  reason?: string | null;
  estimatedSavingsAmount?: number | null;
  currency?: string | null;
};

export type BudgetOptimizationPreservedItem = {
  itemIndex: number;
  itemName: string;
  reason?: string | null;
};

export type BudgetOptimizationProposalContent = {
  summary: string;
  scope: BudgetOptimizationScope;
  dayNumber: number;
  currency: string;
  baseDayEstimatedTotal: number;
  proposedDayEstimatedTotal: number;
  estimatedSavingsAmount: number;
  confidence: "low" | "medium" | "high";
  changes: BudgetOptimizationChange[];
  preservedItems?: BudgetOptimizationPreservedItem[];
  tradeoffs?: string[];
  warnings?: string[];
  proposedDay: ItineraryDay;
};

export type BudgetOptimizationProposal = {
  id: string;
  tripId: string;
  jobId?: string | null;
  createdByUserId: string;
  scope: BudgetOptimizationScope;
  dayNumber?: number | null;
  status: BudgetOptimizationProposalStatus;
  expectedItineraryRevision: number;
  baseItineraryRevision: number;
  currency: string;
  targetReductionAmount?: number | null;
  estimatedSavingsAmount?: number | null;
  proposal: BudgetOptimizationProposalContent;
  appliedItineraryRevision?: number | null;
  createdAt: string;
  appliedAt?: string | null;
  discardedAt?: string | null;
  expiredAt?: string | null;
  updatedAt: string;
};

export type BudgetOptimizationProposalListResponse = {
  proposals: BudgetOptimizationProposal[];
  limit: number;
};

export type BudgetOptimizationProposalEnvelope = {
  proposal: BudgetOptimizationProposal;
};

export type ApplyBudgetOptimizationProposalResponse = {
  trip: Trip;
  proposal: BudgetOptimizationProposal;
};
