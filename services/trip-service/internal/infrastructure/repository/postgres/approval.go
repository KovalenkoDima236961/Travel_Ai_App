package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

// GetTripApprovalFields loads the approval columns for a single trip. It returns
// domain errs.ErrNotFound when the trip does not exist.
func (r *Repository) GetTripApprovalFields(ctx context.Context, tripID uuid.UUID) (*entity.TripApprovalFields, error) {
	query, args, err := r.db.Builder.
		Select(dto.ApprovalFieldColumns).
		From("trips").
		Where(sq.Eq{"id": dto.IDArg(tripID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip approval fields: %w", err)
	}
	return dto.ScanTripApprovalFields(r.db.QueryRow(ctx, query, args...))
}

// UpdateTripApprovalStatus writes the full desired approval state for a trip. The
// service constructs the complete TripApprovalFields (mutating only the columns
// relevant to the action) so every write is a single atomic UPDATE.
func (r *Repository) UpdateTripApprovalStatus(ctx context.Context, fields *entity.TripApprovalFields) (*entity.TripApprovalFields, error) {
	builder := r.db.Builder.Update("trips")
	for _, assignment := range dto.ApprovalUpdateAssignments(fields) {
		builder = builder.Set(assignment.Column, assignment.Value)
	}
	// Approval status changes deliberately do not bump updated_at: an approval
	// decision is not a material itinerary edit and must not itself trigger a
	// reset-on-edit or reorder the workspace trip lists.
	query, args, err := builder.
		Where(sq.Eq{"id": dto.IDArg(fields.TripID)}).
		Suffix("RETURNING " + dto.ApprovalFieldColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip approval status: %w", err)
	}
	return dto.ScanTripApprovalFields(r.db.QueryRow(ctx, query, args...))
}

// InsertTripApprovalEvent appends one row to the approval history.
func (r *Repository) InsertTripApprovalEvent(ctx context.Context, event *entity.TripApprovalEvent) (*entity.TripApprovalEvent, error) {
	query, args, err := r.db.Builder.
		Insert("trip_approval_events").
		Columns(dto.ApprovalEventInsertColumns()...).
		Values(dto.ApprovalEventInsertValues(event)...).
		Suffix("RETURNING " + dto.ApprovalEventColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert trip approval event: %w", err)
	}
	return dto.ScanTripApprovalEvent(r.db.QueryRow(ctx, query, args...))
}

// ListTripApprovalEventsByTrip returns a trip's approval history, newest first.
func (r *Repository) ListTripApprovalEventsByTrip(ctx context.Context, tripID uuid.UUID, limit int) ([]entity.TripApprovalEvent, error) {
	builder := r.db.Builder.
		Select(dto.ApprovalEventColumns).
		From("trip_approval_events").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("created_at DESC", "id DESC")
	if limit > 0 {
		builder = builder.Limit(uint64(limit))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip approval events: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip approval events: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripApprovalEventRows(rows)
}

// approvalQueueOrder ranks statuses for the default queue ordering: pending
// first, then changes_requested, draft, approved, cancelled. Within a rank the
// most recently touched row sorts first.
const approvalQueueOrder = `CASE approval_status ` +
	`WHEN 'pending_approval' THEN 0 ` +
	`WHEN 'changes_requested' THEN 1 ` +
	`WHEN 'draft' THEN 2 ` +
	`WHEN 'approved' THEN 3 ` +
	`WHEN 'cancelled' THEN 4 ` +
	`ELSE 5 END`

// ListWorkspaceApprovals returns one page of workspace trips for the approvals
// queue. It fetches Limit+1 rows so the caller can detect a further page without
// a second count query.
func (r *Repository) ListWorkspaceApprovals(ctx context.Context, params entity.ListWorkspaceApprovalsParams) ([]entity.WorkspaceApprovalRow, error) {
	builder := r.db.Builder.
		Select(dto.WorkspaceApprovalRowColumns).
		From("trips").
		Where(sq.Eq{"workspace_id": dto.IDArg(params.WorkspaceID)})
	if len(params.Statuses) > 0 {
		builder = builder.Where(sq.Eq{"approval_status": params.Statuses})
	}
	if params.SubmittedAfter != nil {
		builder = builder.Where(sq.GtOrEq{"approval_submitted_at": *params.SubmittedAfter})
	}
	if params.SubmittedBefore != nil {
		builder = builder.Where(sq.LtOrEq{"approval_submitted_at": *params.SubmittedBefore})
	}
	builder = builder.OrderBy(
		approvalQueueOrder,
		"COALESCE(approval_submitted_at, approval_last_status_changed_at, updated_at) DESC",
		"id DESC",
	)
	if params.Limit > 0 {
		builder = builder.Limit(uint64(params.Limit))
	}
	if params.Offset > 0 {
		builder = builder.Offset(uint64(params.Offset))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list workspace approvals: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query workspace approvals: %w", err)
	}
	defer rows.Close()
	return dto.ScanWorkspaceApprovalRows(rows)
}

// CountWorkspaceApprovalsByStatus tallies workspace trips per approval status.
func (r *Repository) CountWorkspaceApprovalsByStatus(ctx context.Context, workspaceID uuid.UUID) (entity.WorkspaceApprovalCounts, error) {
	query, args, err := r.db.Builder.
		Select("approval_status", "COUNT(*)").
		From("trips").
		Where(sq.Eq{"workspace_id": dto.IDArg(workspaceID)}).
		GroupBy("approval_status").
		ToSql()
	if err != nil {
		return entity.WorkspaceApprovalCounts{}, fmt.Errorf("build count workspace approvals: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return entity.WorkspaceApprovalCounts{}, fmt.Errorf("query count workspace approvals: %w", err)
	}
	defer rows.Close()

	var counts entity.WorkspaceApprovalCounts
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return entity.WorkspaceApprovalCounts{}, fmt.Errorf("scan workspace approval count: %w", err)
		}
		switch status {
		case "pending_approval":
			counts.PendingApproval = count
		case "changes_requested":
			counts.ChangesRequested = count
		case "approved":
			counts.Approved = count
		case "draft":
			counts.Draft = count
		case "cancelled":
			counts.Cancelled = count
		}
	}
	if err := rows.Err(); err != nil {
		return entity.WorkspaceApprovalCounts{}, fmt.Errorf("iterate workspace approval counts: %w", err)
	}
	return counts, nil
}

// ResetApprovalStatusForTripIfActive atomically moves an approved or
// pending_approval workspace trip back to draft after a material edit. It is a
// no-op (Reset=false) for personal trips and for trips in any other status. The
// previous status is returned via a CTE so callers can emit the right event and
// notifications without a prior read.
func (r *Repository) ResetApprovalStatusForTripIfActive(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.ApprovalResetResult, error) {
	const query = `
WITH prev AS (
    SELECT id, workspace_id, approval_status
    FROM trips
    WHERE id = $1
    FOR UPDATE
)
UPDATE trips t
SET approval_status = 'draft',
    approval_last_status_changed_at = NOW(),
    approval_last_status_changed_by_user_id = $2
FROM prev
WHERE t.id = prev.id
  AND prev.workspace_id IS NOT NULL
  AND prev.approval_status IN ('approved', 'pending_approval')
RETURNING prev.approval_status AS from_status, t.workspace_id`

	row := r.db.QueryRow(ctx, query, dto.IDArg(tripID), dto.IDArg(actorUserID))
	result, err := dto.ScanApprovalResetResult(row)
	if err != nil {
		return nil, err
	}
	return result, nil
}
