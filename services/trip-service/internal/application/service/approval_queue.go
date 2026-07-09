package service

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const (
	defaultApprovalQueueLimit = 50
	maxApprovalQueueLimit     = 100
)

// ListWorkspaceApprovals returns one page of the workspace approvals queue plus
// per-status counts. Any active workspace member (owner/admin/member/viewer) may
// view it; non-members are denied. Each row is enriched with a lightweight
// checklist status and estimated total.
func (s *Service) ListWorkspaceApprovals(ctx context.Context, workspaceID uuid.UUID, input appdto.ListWorkspaceApprovalsInput) (appdto.WorkspaceApprovalsResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.WorkspaceApprovalsResponse{}, err
	}
	if err := s.requireWorkspaceApprovalViewAccess(ctx, user.ID, workspaceID); err != nil {
		return appdto.WorkspaceApprovalsResponse{}, err
	}

	limit := input.Limit
	if limit == 0 {
		limit = defaultApprovalQueueLimit
	}
	if limit < 1 || limit > maxApprovalQueueLimit {
		return appdto.WorkspaceApprovalsResponse{}, apperrs.NewInvalidInput("limit must be between 1 and %d", maxApprovalQueueLimit)
	}
	if input.Offset < 0 {
		return appdto.WorkspaceApprovalsResponse{}, apperrs.NewInvalidInput("offset must be >= 0")
	}
	statuses, err := approvalQueueStatuses(input.Status)
	if err != nil {
		return appdto.WorkspaceApprovalsResponse{}, err
	}

	rows, err := s.repo.ListWorkspaceApprovals(ctx, entity.ListWorkspaceApprovalsParams{
		WorkspaceID: workspaceID,
		Statuses:    statuses,
		Limit:       limit + 1, // fetch one extra to detect a further page
		Offset:      input.Offset,
	})
	if err != nil {
		return appdto.WorkspaceApprovalsResponse{}, err
	}
	counts, err := s.repo.CountWorkspaceApprovalsByStatus(ctx, workspaceID)
	if err != nil {
		return appdto.WorkspaceApprovalsResponse{}, err
	}

	var nextCursor *string
	if len(rows) > limit {
		rows = rows[:limit]
		next := strconv.Itoa(input.Offset + limit)
		nextCursor = &next
	}

	hasWorkspaceBudget := s.workspaceHasPrimaryBudget(ctx, &workspaceID)
	items := make([]appdto.WorkspaceApprovalQueueItem, 0, len(rows))
	for i := range rows {
		items = append(items, s.workspaceApprovalItem(ctx, rows[i], hasWorkspaceBudget))
	}

	return appdto.WorkspaceApprovalsResponse{
		Approvals: items,
		Counts: appdto.WorkspaceApprovalCountsDTO{
			PendingApproval:  counts.PendingApproval,
			ChangesRequested: counts.ChangesRequested,
			Approved:         counts.Approved,
			Draft:            counts.Draft,
		},
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) workspaceApprovalItem(ctx context.Context, row entity.WorkspaceApprovalRow, hasWorkspaceBudget bool) appdto.WorkspaceApprovalQueueItem {
	item := appdto.WorkspaceApprovalQueueItem{
		TripID:            row.TripID,
		Title:             row.Destination,
		Destination:       row.Destination,
		StartDate:         formatDatePtr(row.StartDate),
		ApprovalStatus:    row.ApprovalStatus,
		SubmittedAt:       row.SubmittedAt,
		SubmittedByUserID: row.SubmittedByUserID,
		BudgetAmount:      row.BudgetAmount,
		BudgetCurrency:    row.BudgetCurrency,
		ChecklistStatus:   string(approvals.ChecklistStatusOK),
		Risk:              approvalrisk.UnknownSummary(),
	}

	workspaceID := row.WorkspaceID
	trip := &entity.Trip{
		ID:             row.TripID,
		WorkspaceID:    &workspaceID,
		Destination:    row.Destination,
		StartDate:      row.StartDate,
		BudgetAmount:   row.BudgetAmount,
		BudgetCurrency: row.BudgetCurrency,
		Itinerary:      row.Itinerary,
	}
	checklist, in, err := s.computeChecklistWithInput(ctx, trip, hasWorkspaceBudget)
	if err != nil {
		// A single row's checklist failure must not fail the whole queue; leave
		// the row with a neutral checklist status.
		s.log.Warn("failed to compute checklist for approvals queue row",
			zap.String("trip_id", row.TripID.String()), zap.Error(err))
		return item
	}
	item.ChecklistStatus = string(checklist.Status)
	item.WarningCount = checklist.WarningCount
	item.CriticalCount = checklist.CriticalCount
	item.EstimatedTotal = in.EstimatedTotal
	item.Risk = approvalrisk.QueueSummaryFromResponse(approvalrisk.Score(approvalrisk.Input{
		TripID:      row.TripID,
		WorkspaceID: &workspaceID,
		GeneratedAt: time.Now().UTC(),
		Trip: approvalrisk.TripContext{
			BudgetAmount:   row.BudgetAmount,
			BudgetCurrency: row.BudgetCurrency,
		},
		ChecklistInput: in,
		Itinerary:      parseItineraryLenient(row.Itinerary),
	}))
	return item
}

// requireWorkspaceApprovalViewAccess allows any active workspace member to view
// the approvals queue; non-members are denied.
func (s *Service) requireWorkspaceApprovalViewAccess(ctx context.Context, userID, workspaceID uuid.UUID) error {
	access, err := s.workspaceAccess(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	switch access.Role {
	case workspaces.RoleOwner, workspaces.RoleAdmin, workspaces.RoleMember, workspaces.RoleViewer:
		return nil
	default:
		return apperrs.ErrForbidden
	}
}

// approvalQueueStatuses maps the status query parameter to a repository filter.
// An empty value focuses the queue on the active review set (pending, changes
// requested, draft); "all" removes the filter; a specific status filters to it.
func approvalQueueStatuses(status string) ([]string, error) {
	switch approvals.Status(status) {
	case "":
		return []string{
			string(approvals.StatusPendingApproval),
			string(approvals.StatusChangesRequested),
			string(approvals.StatusDraft),
		}, nil
	case approvals.StatusPendingApproval, approvals.StatusChangesRequested,
		approvals.StatusApproved, approvals.StatusDraft, approvals.StatusCancelled:
		return []string{status}, nil
	default:
		if status == "all" {
			return nil, nil
		}
		return nil, apperrs.NewInvalidInput("invalid status filter")
	}
}

func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format("2006-01-02")
	return &formatted
}
