import { apiFetch } from "@/shared/api/client";
import type { TripActivityEvent } from "@/entities/activity/model";
import type { TripHealthCategory, TripHealthIssueSeverity, TripHealthLevel } from "@/types/trip-health";
import type { ReadinessLevel as GroupReadinessLevel } from "@/types/group-readiness";
import type {
  BudgetConfidenceLevel,
  BudgetConfidenceMoney as Money,
  BudgetRiskLevel
} from "@/types/budget-confidence";
import type { RealWorldReadiness } from "@/types/verification";

export type CommandCenterSectionError = {
  section: string;
  code: string;
  message: string;
};

export type CommandCenterSummary = {
  tripId: string;
  trip: {
    destination: string;
    startDate?: string | null;
    days: number;
    tripType: string;
    itineraryRevision: number;
    updatedAt: string;
    workspaceId?: string | null;
    travelers: number;
    budgetCurrency: string;
    accessRole: string;
    canEdit: boolean;
  };
  health?: {
    score: number;
    level: TripHealthLevel;
    summary: string;
    criticalIssueCount: number;
    highIssueCount: number;
    warningIssueCount: number;
    topFixes: Array<{
      id: string;
      title: string;
      description: string;
      recommendation?: string;
      severity: TripHealthIssueSeverity;
      category: TripHealthCategory;
      label: string;
      href: string;
    }>;
  };
  budget?: {
    confidenceScore: number;
    confidenceLevel: BudgetConfidenceLevel;
    riskLevel: BudgetRiskLevel;
    summary: string;
    coverage: number;
    currency: string;
    estimatedTotal: Money;
    actualTotal: Money;
    tripBudget?: Money | null;
    budgetExceeded: boolean;
    missingEstimateCount: number;
  };
  groupReadiness?: {
    score: number;
    level: GroupReadinessLevel;
    summary: string;
    memberCount: number;
    membersNeedingAttention: number;
    topActionLabel?: string;
    topActionHref?: string;
  };
  realWorldReadiness?: Pick<RealWorldReadiness, "score" | "level"> & {
    topIssueCount: number;
    verifiedCount: number;
    staleCount: number;
    missingCount: number;
  };
  route: {
    stopCount: number;
    legCount: number;
    selectedTransportCoverage: number;
    missingTransportCount: number;
  };
  checklist?: {
    completedCount: number;
    totalCount: number;
    overdueCount: number;
    highPriorityCount: number;
  };
  reminders?: { totalCount: number; overdueCount: number; dueSoonCount: number };
  expenses?: {
    expenseCount: number;
    actualTotal: { amount: number; currency: string };
    pendingSettlementCount: number;
  };
  activity?: {
    recentCount: number;
    latestAt?: string | null;
    items: TripActivityEvent[];
  };
  sectionErrors: CommandCenterSectionError[];
  computedAt: string;
};

export function getCommandCenterSummary(tripId: string) {
  return apiFetch<CommandCenterSummary>(`/trips/${tripId}/command-center-summary`);
}
