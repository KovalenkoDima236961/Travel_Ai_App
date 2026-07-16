import type { TripActivityEvent } from "@/entities/activity/model";
import type { ApprovalRiskResponse } from "@/entities/approval-risk/model";
import type { TripApprovalState } from "@/entities/approval/model";
import type { BudgetSummary } from "@/entities/budget/model";
import type { ChecklistViewResponse } from "@/entities/checklist/model";
import type { ExpenseSummary, SettlementsResponse } from "@/entities/expense/model";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { GroupReadiness } from "@/types/group-readiness";
import type { BudgetConfidence } from "@/types/budget-confidence";
import type { ReminderViewResponse } from "@/entities/trip-reminder/model";
import type { Trip } from "@/entities/trip/model";
import type { TripPoll } from "@/types/trip-decisions";
import type { TripAvailabilityList } from "@/types/trip-availability";
import type { TripHealth, TripHealthCategory, TripHealthIssueSeverity } from "@/types/trip-health";
import type { PolicyEvaluation } from "@/types/workspace-policy";

export type ReadinessCardStatus =
  | "ready"
  | "almost_ready"
  | "needs_attention"
  | "blocked"
  | "empty"
  | "unavailable";

export type TripCommandCenterSection =
  | "overview"
  | "itinerary"
  | "route"
  | "dates"
  | "polls"
  | "budget"
  | "expenses"
  | "settlements"
  | "receipts"
  | "checklist"
  | "reminders"
  | "offline"
  | "collaborators"
  | "activity"
  | "comments"
  | "health"
  | "approval"
  | "policy"
  | "versions";

export type TripCommandCenterCardId =
  | "health"
  | "route_transport"
  | "budget"
  | "group"
  | "checklist_reminders"
  | "expenses_settlements"
  | "approval_policy"
  | "activity"
  | "offline";

export type CommandCenterAction = {
  label: string;
  href: string;
};

export type NextBestAction = {
  id: string;
  title: string;
  description: string;
  reason: string;
  severity: TripHealthIssueSeverity;
  category: TripHealthCategory | "group" | "activity";
  actionLabel: string;
  href: string;
  source:
    | "trip_health"
    | "policy"
    | "approval"
    | "trip"
    | "route"
    | "budget"
    | "checklist"
    | "reminders"
    | "group"
    | "expenses"
    | "offline";
  viewOnly?: boolean;
};

export type ReadinessCardMetric = {
  label: string;
  value: string;
};

export type ReadinessCard = {
  id: TripCommandCenterCardId;
  title: string;
  status: ReadinessCardStatus;
  score?: number | null;
  summary: string;
  detail?: string | null;
  metrics: ReadinessCardMetric[];
  primaryAction?: CommandCenterAction | null;
  secondaryAction?: CommandCenterAction | null;
};

export type NavigationItem = {
  id: TripCommandCenterSection;
  label: string;
  href: string;
  badge?: number | string | null;
};

export type NavigationGroup = {
  id: "plan" | "prepare" | "money" | "team" | "control";
  label: string;
  items: NavigationItem[];
};

export type OfflineCommandCenterStatus = {
  online: boolean;
  availableOffline: boolean;
  pendingCount: number;
  failedCount: number;
  conflictCount: number;
  syncing: boolean;
  cachedAt?: string | null;
};

export type TripCommandCenterAccess = {
  canEdit: boolean;
  canCollaborate: boolean;
  canView: boolean;
  currentUserId?: string | null;
};

export type TripCommandCenterInput = {
  trip: Trip;
  health?: TripHealth | null;
  budgetSummary?: BudgetSummary | null;
  budgetConfidence?: BudgetConfidence | null;
  availability?: TripAvailabilityList | null;
  checklist?: ChecklistViewResponse | null;
  reminders?: ReminderViewResponse | null;
  expenseSummary?: ExpenseSummary | null;
  settlements?: SettlementsResponse | null;
  approval?: TripApprovalState | null;
  policyEvaluation?: PolicyEvaluation | null;
  approvalRisk?: ApprovalRiskResponse | null;
  activity?: TripActivityEvent[] | null;
  polls?: TripPoll[] | null;
  groupReadiness?: GroupReadiness | null;
  generationJobs?: GenerationJob[] | null;
  offlineStatus: OfflineCommandCenterStatus;
  userAccess: TripCommandCenterAccess;
};

export type TripCommandCenterData = {
  nextBestAction: NextBestAction;
  topFixes: NextBestAction[];
  cards: ReadinessCard[];
  navigationGroups: NavigationGroup[];
  recentActivity: TripActivityEvent[];
};
