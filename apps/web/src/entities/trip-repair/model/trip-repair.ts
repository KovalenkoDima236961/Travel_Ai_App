import type { Itinerary, Trip } from "@/entities/trip/model";

export type RepairMode =
  | "policy_compliance"
  | "reduce_budget_risk"
  | "fix_schedule_risk"
  | "reduce_walking"
  | "add_rest_time"
  | "replace_disallowed_items"
  | "selected_issues";

export type RepairProposalStatus = "pending" | "applied" | "discarded" | "expired" | "failed";

export type RepairConstraints = {
  preserveConfirmedItems?: boolean;
  minimizeChanges?: boolean;
  preserveUserEditedItems?: boolean;
  doNotChangeAccommodation?: boolean;
  doNotChangeDates?: boolean;
  maxChangedItems?: number | null;
};

export type CreateRepairJobInput = {
  expectedItineraryRevision: number;
  repairMode: RepairMode;
  selectedIssueTypes?: string[];
  selectedRiskFactorTypes?: string[];
  constraints?: RepairConstraints;
  specialInstructions?: string | null;
};

export type RepairMoney = {
  amount: number;
  currency: string;
};

export type RepairIssue = {
  type: string;
  severity?: string | null;
  message: string;
  affected?: {
    dayNumber?: number | null;
    itemIndex?: number | null;
    name?: string | null;
    amount?: number | null;
    currency?: string | null;
  } | null;
};

export type RepairSummary = {
  repairMode: RepairMode;
  changedItemCount: number;
  addedItemCount: number;
  removedItemCount: number;
  movedItemCount: number;
  estimatedCostBefore?: RepairMoney | null;
  estimatedCostAfter?: RepairMoney | null;
  majorChanges: string[];
  issuesAddressed: string[];
  issuesRemaining: string[];
  warnings: string[];
};

export type RepairFieldChange = {
  field: string;
  before?: unknown;
  after?: unknown;
};

export type RepairChange = {
  type: string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  before?: Record<string, unknown> | null;
  after?: Record<string, unknown> | null;
  fieldChanges?: RepairFieldChange[];
  reason?: string | null;
};

export type RepairDiff = {
  daysChanged: RepairChange[];
  itemsAdded: RepairChange[];
  itemsRemoved: RepairChange[];
  itemsModified: RepairChange[];
  itemsMoved: RepairChange[];
  warnings?: string[];
};

export type RepairValidation = {
  valid: boolean;
  warnings: string[];
};

export type RepairProposalContent = {
  repairedItinerary: Itinerary;
  repairSummary: RepairSummary;
  changes: RepairChange[];
  diff: RepairDiff;
  validation: RepairValidation;
};

export type RepairProposal = {
  id: string;
  tripId: string;
  jobId?: string | null;
  createdByUserId: string;
  status: RepairProposalStatus;
  repairMode: RepairMode;
  baseItineraryRevision: number;
  baseRiskScore?: number | null;
  proposedRiskScore?: number | null;
  basePolicyStatus?: string | null;
  proposedPolicyStatus?: string | null;
  summary: RepairSummary;
  createdAt: string;
  updatedAt: string;
  appliedAt?: string | null;
  appliedByUserId?: string | null;
  discardedAt?: string | null;
  discardedByUserId?: string | null;
  expiredAt?: string | null;
};

export type RepairProposalDetail = RepairProposal & {
  issues: RepairIssue[];
  proposal: RepairProposalContent;
};

export type RepairProposalListResponse = {
  proposals: RepairProposal[];
  limit: number;
};

export type RepairProposalEnvelope = {
  proposal: RepairProposalDetail;
};

export type ApplyRepairProposalResponse = {
  trip: Trip;
  proposal: RepairProposalDetail;
};
