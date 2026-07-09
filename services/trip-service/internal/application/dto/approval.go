package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
)

// TripApprovalState is the GET /trips/{id}/approval response. For personal trips
// Status is "not_required", Checklist is nil, and every can* flag is false.
type TripApprovalState struct {
	TripID                    uuid.UUID            `json:"tripId"`
	WorkspaceID               *uuid.UUID           `json:"workspaceId"`
	Status                    string               `json:"status"`
	SubmittedAt               *time.Time           `json:"submittedAt"`
	SubmittedByUserID         *uuid.UUID           `json:"submittedByUserId"`
	ApprovedAt                *time.Time           `json:"approvedAt"`
	ApprovedByUserID          *uuid.UUID           `json:"approvedByUserId"`
	ChangesRequestedAt        *time.Time           `json:"changesRequestedAt"`
	ChangesRequestedByUserID  *uuid.UUID           `json:"changesRequestedByUserId"`
	CancelledAt               *time.Time           `json:"cancelledAt"`
	CancelledByUserID         *uuid.UUID           `json:"cancelledByUserId"`
	Note                      *string              `json:"note"`
	DecisionNote              *string              `json:"decisionNote"`
	LastStatusChangedAt       *time.Time           `json:"lastStatusChangedAt"`
	LastStatusChangedByUserID *uuid.UUID           `json:"lastStatusChangedByUserId"`
	Checklist                 *approvals.Checklist `json:"checklist,omitempty"`
	CanSubmit                 bool                 `json:"canSubmit"`
	CanApprove                bool                 `json:"canApprove"`
	CanRequestChanges         bool                 `json:"canRequestChanges"`
	CanCancel                 bool                 `json:"canCancel"`
}

// TripApprovalEventDTO is one approval-history entry in the API.
type TripApprovalEventDTO struct {
	ID                uuid.UUID       `json:"id"`
	EventType         string          `json:"eventType"`
	FromStatus        *string         `json:"fromStatus"`
	ToStatus          string          `json:"toStatus"`
	ActorUserID       uuid.UUID       `json:"actorUserId"`
	Note              *string         `json:"note"`
	ChecklistSnapshot json.RawMessage `json:"checklistSnapshot,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
}

// TripApprovalEventsResponse envelopes the approval history.
type TripApprovalEventsResponse struct {
	Events []TripApprovalEventDTO `json:"events"`
}

// WorkspaceApprovalQueueItem is one row of the workspace approvals queue.
type WorkspaceApprovalQueueItem struct {
	TripID                 uuid.UUID                 `json:"tripId"`
	Title                  string                    `json:"title"`
	Destination            string                    `json:"destination"`
	StartDate              *string                   `json:"startDate,omitempty"`
	ApprovalStatus         string                    `json:"approvalStatus"`
	SubmittedAt            *time.Time                `json:"submittedAt"`
	SubmittedByUserID      *uuid.UUID                `json:"submittedByUserId"`
	SubmittedByDisplayName *string                   `json:"submittedByDisplayName,omitempty"`
	EstimatedTotal         float64                   `json:"estimatedTotal"`
	BudgetAmount           *float64                  `json:"budgetAmount,omitempty"`
	BudgetCurrency         string                    `json:"budgetCurrency,omitempty"`
	ChecklistStatus        string                    `json:"checklistStatus"`
	WarningCount           int                       `json:"warningCount"`
	CriticalCount          int                       `json:"criticalCount"`
	Risk                   approvalrisk.QueueSummary `json:"risk"`
}

// WorkspaceApprovalCountsDTO is the per-status tally above the queue.
type WorkspaceApprovalCountsDTO struct {
	PendingApproval  int `json:"pendingApproval"`
	ChangesRequested int `json:"changesRequested"`
	Approved         int `json:"approved"`
	Draft            int `json:"draft"`
}

// WorkspaceApprovalsResponse is the GET /workspaces/{id}/approvals response.
type WorkspaceApprovalsResponse struct {
	Approvals  []WorkspaceApprovalQueueItem `json:"approvals"`
	Counts     WorkspaceApprovalCountsDTO   `json:"counts"`
	NextCursor *string                      `json:"nextCursor"`
}

// SubmitApprovalInput is the use-case input for a submit action.
type SubmitApprovalInput struct {
	Note                 string
	AcknowledgedWarnings []string
}

// ApprovalDecisionInput is the use-case input for approve/request-changes.
type ApprovalDecisionInput struct {
	DecisionNote string
}

// CancelApprovalInput is the use-case input for cancelling a pending submission.
type CancelApprovalInput struct {
	Note string
}

// ListWorkspaceApprovalsInput filters and paginates the queue request.
type ListWorkspaceApprovalsInput struct {
	Status string
	Limit  int
	Offset int
}
