package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

// ApprovalFieldColumns is the canonical column order for reading a trip's
// approval fields. It intentionally starts with id (as trip id) so callers get a
// self-describing TripApprovalFields without touching the main trip scan path.
const ApprovalFieldColumns = "id, workspace_id, approval_status, " +
	"approval_submitted_at, approval_submitted_by_user_id, " +
	"approval_approved_at, approval_approved_by_user_id, " +
	"approval_changes_requested_at, approval_changes_requested_by_user_id, " +
	"approval_cancelled_at, approval_cancelled_by_user_id, " +
	"approval_note, approval_decision_note, " +
	"approval_last_status_changed_at, approval_last_status_changed_by_user_id"

// ScanTripApprovalFields reads a single row (in ApprovalFieldColumns order) into
// TripApprovalFields, mapping domain errs.ErrNotFound when the trip is absent.
func ScanTripApprovalFields(row pgx.Row) (*entity.TripApprovalFields, error) {
	var (
		tripID                    pgtype.UUID
		workspaceID               pgtype.UUID
		status                    string
		submittedAt               pgtype.Timestamp
		submittedByUserID         pgtype.UUID
		approvedAt                pgtype.Timestamp
		approvedByUserID          pgtype.UUID
		changesRequestedAt        pgtype.Timestamp
		changesRequestedByUserID  pgtype.UUID
		cancelledAt               pgtype.Timestamp
		cancelledByUserID         pgtype.UUID
		note                      pgtype.Text
		decisionNote              pgtype.Text
		lastStatusChangedAt       pgtype.Timestamp
		lastStatusChangedByUserID pgtype.UUID
	)
	err := row.Scan(
		&tripID, &workspaceID, &status,
		&submittedAt, &submittedByUserID,
		&approvedAt, &approvedByUserID,
		&changesRequestedAt, &changesRequestedByUserID,
		&cancelledAt, &cancelledByUserID,
		&note, &decisionNote,
		&lastStatusChangedAt, &lastStatusChangedByUserID,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip approval fields: %w", err)
	}
	return &entity.TripApprovalFields{
		TripID:                    uuid.UUID(tripID.Bytes),
		WorkspaceID:               fromPgUUID(workspaceID),
		Status:                    status,
		SubmittedAt:               fromPgTimestamp(submittedAt),
		SubmittedByUserID:         fromPgUUID(submittedByUserID),
		ApprovedAt:                fromPgTimestamp(approvedAt),
		ApprovedByUserID:          fromPgUUID(approvedByUserID),
		ChangesRequestedAt:        fromPgTimestamp(changesRequestedAt),
		ChangesRequestedByUserID:  fromPgUUID(changesRequestedByUserID),
		CancelledAt:               fromPgTimestamp(cancelledAt),
		CancelledByUserID:         fromPgUUID(cancelledByUserID),
		Note:                      fromPgText(note),
		DecisionNote:              fromPgText(decisionNote),
		LastStatusChangedAt:       fromPgTimestamp(lastStatusChangedAt),
		LastStatusChangedByUserID: fromPgUUID(lastStatusChangedByUserID),
	}, nil
}

// ApprovalUpdateAssignments returns the column->value pairs written by an
// approval status update, in a stable order. The service builds the full desired
// TripApprovalFields state and this writes every approval column atomically.
func ApprovalUpdateAssignments(f *entity.TripApprovalFields) []struct {
	Column string
	Value  any
} {
	return []struct {
		Column string
		Value  any
	}{
		{"approval_status", f.Status},
		{"approval_submitted_at", toPgTimestampPtr(f.SubmittedAt)},
		{"approval_submitted_by_user_id", toPgUUIDPtr(f.SubmittedByUserID)},
		{"approval_approved_at", toPgTimestampPtr(f.ApprovedAt)},
		{"approval_approved_by_user_id", toPgUUIDPtr(f.ApprovedByUserID)},
		{"approval_changes_requested_at", toPgTimestampPtr(f.ChangesRequestedAt)},
		{"approval_changes_requested_by_user_id", toPgUUIDPtr(f.ChangesRequestedByUserID)},
		{"approval_cancelled_at", toPgTimestampPtr(f.CancelledAt)},
		{"approval_cancelled_by_user_id", toPgUUIDPtr(f.CancelledByUserID)},
		{"approval_note", toPgTextPtr(f.Note)},
		{"approval_decision_note", toPgTextPtr(f.DecisionNote)},
		{"approval_last_status_changed_at", toPgTimestampPtr(f.LastStatusChangedAt)},
		{"approval_last_status_changed_by_user_id", toPgUUIDPtr(f.LastStatusChangedByUserID)},
	}
}

// ApprovalEventColumns is the canonical column order for trip_approval_events.
const ApprovalEventColumns = "id, trip_id, workspace_id, actor_user_id, event_type, " +
	"from_status, to_status, note, checklist_snapshot, created_at"

// ApprovalEventInsertColumns lists the columns set on INSERT (id and created_at
// are DB-defaulted), in the same order as ApprovalEventInsertValues.
func ApprovalEventInsertColumns() []string {
	return []string{
		"trip_id", "workspace_id", "actor_user_id", "event_type",
		"from_status", "to_status", "note", "checklist_snapshot",
	}
}

// ApprovalEventInsertValues returns the values for ApprovalEventInsertColumns.
func ApprovalEventInsertValues(e *entity.TripApprovalEvent) []any {
	var snapshot any
	if len(e.ChecklistSnapshot) > 0 {
		snapshot = []byte(e.ChecklistSnapshot)
	}
	return []any{
		toPgUUID(e.TripID), toPgUUID(e.WorkspaceID), toPgUUID(e.ActorUserID), e.EventType,
		toPgTextPtr(e.FromStatus), e.ToStatus, toPgTextPtr(e.Note), snapshot,
	}
}

// ScanTripApprovalEvent reads a single trip_approval_events row.
func ScanTripApprovalEvent(row pgx.Row) (*entity.TripApprovalEvent, error) {
	var (
		id, tripID, workspaceID, actorUserID pgtype.UUID
		eventType                            string
		fromStatus                           pgtype.Text
		toStatus                             string
		note                                 pgtype.Text
		snapshot                             []byte
		createdAt                            pgtype.Timestamp
	)
	err := row.Scan(
		&id, &tripID, &workspaceID, &actorUserID, &eventType,
		&fromStatus, &toStatus, &note, &snapshot, &createdAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip approval event: %w", err)
	}
	event := &entity.TripApprovalEvent{
		ID:          uuid.UUID(id.Bytes),
		TripID:      uuid.UUID(tripID.Bytes),
		WorkspaceID: uuid.UUID(workspaceID.Bytes),
		ActorUserID: uuid.UUID(actorUserID.Bytes),
		EventType:   eventType,
		FromStatus:  fromPgText(fromStatus),
		ToStatus:    toStatus,
		Note:        fromPgText(note),
		CreatedAt:   createdAt.Time,
	}
	if len(snapshot) > 0 {
		event.ChecklistSnapshot = json.RawMessage(snapshot)
	}
	return event, nil
}

// ScanTripApprovalEventRows reads a set of trip_approval_events rows.
func ScanTripApprovalEventRows(rows pgx.Rows) ([]entity.TripApprovalEvent, error) {
	events := make([]entity.TripApprovalEvent, 0)
	for rows.Next() {
		event, err := ScanTripApprovalEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip approval events: %w", err)
	}
	return events, nil
}

// WorkspaceApprovalRowColumns is the projection used by the approvals queue.
const WorkspaceApprovalRowColumns = "id, workspace_id, destination, start_date, " +
	"budget_amount, budget_currency, approval_status, approval_submitted_at, " +
	"approval_submitted_by_user_id, approval_approved_at, updated_at, itinerary"

// ScanWorkspaceApprovalRows reads workspace approval queue rows.
func ScanWorkspaceApprovalRows(rows pgx.Rows) ([]entity.WorkspaceApprovalRow, error) {
	out := make([]entity.WorkspaceApprovalRow, 0)
	for rows.Next() {
		var (
			tripID, workspaceID pgtype.UUID
			destination         string
			startDate           pgtype.Date
			budgetAmount        pgtype.Numeric
			budgetCurrency      pgtype.Text
			approvalStatus      string
			submittedAt         pgtype.Timestamp
			submittedByUserID   pgtype.UUID
			approvedAt          pgtype.Timestamp
			updatedAt           pgtype.Timestamp
			itineraryRaw        []byte
		)
		if err := rows.Scan(
			&tripID, &workspaceID, &destination, &startDate,
			&budgetAmount, &budgetCurrency, &approvalStatus, &submittedAt,
			&submittedByUserID, &approvedAt, &updatedAt, &itineraryRaw,
		); err != nil {
			return nil, fmt.Errorf("scan workspace approval row: %w", err)
		}
		row := entity.WorkspaceApprovalRow{
			TripID:            uuid.UUID(tripID.Bytes),
			WorkspaceID:       uuid.UUID(workspaceID.Bytes),
			Destination:       destination,
			StartDate:         fromPgDate(startDate),
			BudgetAmount:      fromPgNumeric(budgetAmount),
			BudgetCurrency:    budgetCurrency.String,
			ApprovalStatus:    approvalStatus,
			SubmittedAt:       fromPgTimestamp(submittedAt),
			SubmittedByUserID: fromPgUUID(submittedByUserID),
			ApprovedAt:        fromPgTimestamp(approvedAt),
			UpdatedAt:         updatedAt.Time,
		}
		if len(itineraryRaw) > 0 {
			row.Itinerary = append([]byte(nil), itineraryRaw...)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace approval rows: %w", err)
	}
	return out, nil
}

// ScanApprovalResetResult reads the CTE result of an atomic reset-on-edit. When
// no row was reset (personal trip, or not approved/pending) the query returns no
// rows, which maps to a Reset=false result rather than an error.
func ScanApprovalResetResult(row pgx.Row) (*entity.ApprovalResetResult, error) {
	var (
		fromStatus  string
		workspaceID pgtype.UUID
	)
	if err := row.Scan(&fromStatus, &workspaceID); err != nil {
		if postgres.NoRowsFound(err) {
			return &entity.ApprovalResetResult{Reset: false}, nil
		}
		return nil, fmt.Errorf("scan approval reset result: %w", err)
	}
	result := &entity.ApprovalResetResult{
		Reset:      true,
		FromStatus: fromStatus,
		ToStatus:   "draft",
	}
	if workspaceID.Valid {
		result.WorkspaceID = uuid.UUID(workspaceID.Bytes)
	}
	return result, nil
}

// --- nullable pgtype helpers used by the approval mappers ---

func toPgTimestampPtr(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}
