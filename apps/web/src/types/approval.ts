// Types for the Workspace Approval Workflow. They mirror the Trip Service
// approval API (see services/trip-service/internal/application/dto/approval.go).

export type ApprovalStatus =
  | "not_required"
  | "draft"
  | "pending_approval"
  | "changes_requested"
  | "approved"
  | "cancelled";

export type ChecklistItemStatus = "ok" | "warning" | "blocked" | "info";
export type ChecklistSeverity = "blocker" | "warning" | "info";
export type ChecklistOverallStatus = "ok" | "warning" | "blocked";

export interface ApprovalChecklistItem {
  key: string;
  status: ChecklistItemStatus;
  severity: ChecklistSeverity;
  title: string;
  message: string;
}

export interface ApprovalChecklist {
  status: ChecklistOverallStatus;
  items: ApprovalChecklistItem[];
  warningCount: number;
  criticalCount: number;
  blockerCount: number;
}

export interface TripApprovalState {
  tripId: string;
  workspaceId: string | null;
  status: ApprovalStatus;
  submittedAt: string | null;
  submittedByUserId: string | null;
  approvedAt: string | null;
  approvedByUserId: string | null;
  changesRequestedAt: string | null;
  changesRequestedByUserId: string | null;
  cancelledAt: string | null;
  cancelledByUserId: string | null;
  note: string | null;
  decisionNote: string | null;
  lastStatusChangedAt: string | null;
  lastStatusChangedByUserId: string | null;
  checklist?: ApprovalChecklist;
  canSubmit: boolean;
  canApprove: boolean;
  canRequestChanges: boolean;
  canCancel: boolean;
}

export type ApprovalEventType =
  | "submitted"
  | "approved"
  | "changes_requested"
  | "cancelled"
  | "reset_to_draft";

export interface TripApprovalEvent {
  id: string;
  eventType: ApprovalEventType;
  fromStatus: string | null;
  toStatus: string;
  actorUserId: string;
  note: string | null;
  checklistSnapshot?: unknown;
  createdAt: string;
}

export interface TripApprovalEventsResponse {
  events: TripApprovalEvent[];
}

export interface WorkspaceApprovalQueueItem {
  tripId: string;
  title: string;
  destination: string;
  startDate?: string;
  approvalStatus: ApprovalStatus;
  submittedAt: string | null;
  submittedByUserId: string | null;
  submittedByDisplayName?: string;
  estimatedTotal: number;
  budgetAmount?: number;
  budgetCurrency?: string;
  checklistStatus: ChecklistOverallStatus;
  warningCount: number;
  criticalCount: number;
}

export interface WorkspaceApprovalCounts {
  pendingApproval: number;
  changesRequested: number;
  approved: number;
  draft: number;
}

export interface WorkspaceApprovalsResponse {
  approvals: WorkspaceApprovalQueueItem[];
  counts: WorkspaceApprovalCounts;
  nextCursor: string | null;
}

export interface SubmitApprovalInput {
  note?: string;
  acknowledgedWarnings?: string[];
}

export interface ApprovalDecisionInput {
  decisionNote?: string;
}

export interface CancelApprovalInput {
  note?: string;
}

// Status filter used by the workspace approvals queue tabs.
export type WorkspaceApprovalStatusFilter = ApprovalStatus | "all";
