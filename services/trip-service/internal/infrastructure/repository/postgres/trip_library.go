package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

const maxLibrarySourceTrips = 500

// ArchiveTrip only persists lifecycle organization. Authorization is kept in
// the application service so this method can never widen access on its own.
func (r *Repository) ArchiveTrip(ctx context.Context, tripID, actorUserID uuid.UUID, reason string) (*entity.Trip, error) {
	query, args, err := r.db.Builder.Update("trips").
		Set("archived_at", sq.Expr("NOW()")).
		Set("archived_by_user_id", dto.IDArg(actorUserID)).
		Set("archive_reason", nullableTrimmedText(reason)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(tripID)}).
		Suffix("RETURNING " + dto.Columns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build archive trip: %w", err)
	}
	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) RestoreTrip(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	query, args, err := r.db.Builder.Update("trips").
		Set("archived_at", nil).
		Set("archived_by_user_id", nil).
		Set("archive_reason", nil).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(tripID)}).
		Suffix("RETURNING " + dto.Columns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build restore trip: %w", err)
	}
	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}

// ListAccessibleForLibrary returns only private trips visible to the actor.
// Filtering by lifecycle stays in the service because lifecycle is derived and
// because that avoids persisting a duplicate mutable status.
func (r *Repository) ListAccessibleForLibrary(
	ctx context.Context,
	userID uuid.UUID,
	workspaceIDs []uuid.UUID,
	workspaceID *uuid.UUID,
) ([]entity.Trip, error) {
	builder := r.db.Builder.Select(dto.Columns).From("trips")
	ids := filterWorkspaceIDs(workspaceIDs, workspaceID)
	if workspaceID != nil && len(ids) == 0 {
		return []entity.Trip{}, nil
	}
	if workspaceID != nil {
		builder = builder.Where(sq.Eq{"workspace_id": ids})
	} else {
		conditions := sq.Or{sq.And{sq.Eq{"user_id": dto.IDArg(userID)}, sq.Expr("workspace_id IS NULL")}}
		if len(ids) > 0 {
			conditions = append(conditions, sq.Eq{"workspace_id": ids})
		}
		// Accepted collaborators can browse a private shared trip in the library;
		// removed and pending collaborators are excluded. Workspace trips continue
		// to require workspace membership through the branch above.
		conditions = append(conditions, sq.And{sq.Expr("workspace_id IS NULL"), sq.Expr("EXISTS (SELECT 1 FROM trip_collaborators tc WHERE tc.trip_id = trips.id AND tc.user_id = ? AND tc.status = 'accepted')", dto.IDArg(userID))})
		builder = builder.Where(conditions)
	}
	query, args, err := builder.
		OrderBy("updated_at DESC", "id DESC").Limit(maxLibrarySourceTrips).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build trip library source list: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip library source list: %w", err)
	}
	defer rows.Close()
	items := make([]entity.Trip, 0)
	for rows.Next() {
		trip, scanErr := dto.Scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *trip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip library source list: %w", err)
	}
	return items, nil
}

// GetTripLibrarySummaries reads only compact aggregate data. It never selects
// receipts, notes, comments, calendar data, share controls, or provider data.
func (r *Repository) GetTripLibrarySummaries(ctx context.Context, tripIDs []uuid.UUID) (map[uuid.UUID]appdto.TripLibrarySummary, error) {
	result := make(map[uuid.UUID]appdto.TripLibrarySummary, len(tripIDs))
	if len(tripIDs) == 0 {
		return result, nil
	}
	ids := libraryTripIDArgs(tripIDs)
	if err := r.loadLibraryRecaps(ctx, ids, result); err != nil {
		return nil, err
	}
	if err := r.loadLibraryTemplates(ctx, ids, result); err != nil {
		return nil, err
	}
	if err := r.loadLibraryExpenses(ctx, ids, result); err != nil {
		return nil, err
	}
	if err := r.loadLibraryChecklistCompletion(ctx, ids, result); err != nil {
		return nil, err
	}
	if err := r.loadLibraryMissedItems(ctx, ids, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) loadLibraryRecaps(ctx context.Context, ids []any, out map[uuid.UUID]appdto.TripLibrarySummary) error {
	query, args, err := r.db.Builder.Select("trip_id", "status", "created_at", "recap_json").
		From("trip_recaps").Where(sq.Eq{"trip_id": ids}).Where("archived_at IS NULL").ToSql()
	if err != nil {
		return fmt.Errorf("build library recap summaries: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query library recap summaries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tripID pgtype.UUID
		var status string
		var createdAt pgtype.Timestamp
		var raw []byte
		if err := rows.Scan(&tripID, &status, &createdAt, &raw); err != nil {
			return fmt.Errorf("scan library recap summary: %w", err)
		}
		id := uuid.UUID(tripID.Bytes)
		summary := out[id]
		statusValue := entity.TripRecapStatus(status)
		summary.RecapStatus = &statusValue
		if createdAt.Valid {
			value := createdAt.Time
			summary.RecapCreatedAt = &value
		}
		var payload struct {
			LessonsLearned []string `json:"lessonsLearned"`
		}
		if json.Unmarshal(raw, &payload) == nil {
			summary.Lessons = trimLibraryStrings(payload.LessonsLearned, 5, 240)
		}
		out[id] = summary
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate library recap summaries: %w", err)
	}
	return nil
}

func (r *Repository) loadLibraryTemplates(ctx context.Context, ids []any, out map[uuid.UUID]appdto.TripLibrarySummary) error {
	query, args, err := r.db.Builder.Select("source_trip_id", "id").From("trip_templates").
		Where(sq.Eq{"source_trip_id": ids}).Where("archived_at IS NULL").OrderBy("created_at DESC").ToSql()
	if err != nil {
		return fmt.Errorf("build library template summaries: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query library template summaries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tripID, templateID pgtype.UUID
		if err := rows.Scan(&tripID, &templateID); err != nil {
			return fmt.Errorf("scan library template summary: %w", err)
		}
		id := uuid.UUID(tripID.Bytes)
		summary := out[id]
		if summary.TemplateID == nil {
			value := uuid.UUID(templateID.Bytes)
			summary.TemplateID = &value
			out[id] = summary
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate library template summaries: %w", err)
	}
	return nil
}

func (r *Repository) loadLibraryExpenses(ctx context.Context, ids []any, out map[uuid.UUID]appdto.TripLibrarySummary) error {
	query, args, err := r.db.Builder.Select("trip_id", "currency", "SUM(amount)::text").From("trip_expenses").
		Where(sq.Eq{"trip_id": ids}).Where("deleted_at IS NULL").Where("status = 'active'").
		GroupBy("trip_id", "currency").ToSql()
	if err != nil {
		return fmt.Errorf("build library expense summaries: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query library expense summaries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tripID pgtype.UUID
		var currency, amountRaw string
		if err := rows.Scan(&tripID, &currency, &amountRaw); err != nil {
			return fmt.Errorf("scan library expense summary: %w", err)
		}
		amount, err := strconv.ParseFloat(amountRaw, 64)
		if err != nil {
			return fmt.Errorf("parse library expense amount: %w", err)
		}
		id := uuid.UUID(tripID.Bytes)
		summary := out[id]
		summary.HasExpenses = true
		summary.ExpenseTotals = append(summary.ExpenseTotals, appdto.LibraryMoney{Amount: amount, Currency: currency})
		out[id] = summary
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate library expense summaries: %w", err)
	}
	return nil
}

func (r *Repository) loadLibraryChecklistCompletion(ctx context.Context, ids []any, out map[uuid.UUID]appdto.TripLibrarySummary) error {
	query, args, err := r.db.Builder.Select("trip_id", "COUNT(*)", "COUNT(*) FILTER (WHERE checked)").From("trip_checklist_items").
		Where(sq.Eq{"trip_id": ids}).Where("deleted_at IS NULL").GroupBy("trip_id").ToSql()
	if err != nil {
		return fmt.Errorf("build library checklist summaries: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query library checklist summaries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tripID pgtype.UUID
		var planned, done int
		if err := rows.Scan(&tripID, &planned, &done); err != nil {
			return fmt.Errorf("scan library checklist summary: %w", err)
		}
		id := uuid.UUID(tripID.Bytes)
		summary := out[id]
		summary.PlannedCount = planned
		summary.DoneCount = done
		out[id] = summary
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate library checklist summaries: %w", err)
	}
	return nil
}

func (r *Repository) loadLibraryMissedItems(ctx context.Context, ids []any, out map[uuid.UUID]appdto.TripLibrarySummary) error {
	query, args, err := r.db.Builder.Select("trip_id", "title").From("trip_checklist_items").
		Where(sq.Eq{"trip_id": ids}).Where("deleted_at IS NULL").Where("checked = false").OrderBy("trip_id", "sort_order ASC").ToSql()
	if err != nil {
		return fmt.Errorf("build library missed-item summaries: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query library missed-item summaries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tripID pgtype.UUID
		var title string
		if err := rows.Scan(&tripID, &title); err != nil {
			return fmt.Errorf("scan library missed item: %w", err)
		}
		id := uuid.UUID(tripID.Bytes)
		summary := out[id]
		if len(summary.MissedItems) < 12 {
			summary.MissedItems = append(summary.MissedItems, strings.TrimSpace(title))
			out[id] = summary
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate library missed items: %w", err)
	}
	return nil
}

func libraryTripIDArgs(ids []uuid.UUID) []any {
	values := make([]any, 0, len(ids))
	for _, id := range ids {
		values = append(values, dto.IDArg(id))
	}
	return values
}

func nullableTrimmedText(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func trimLibraryStrings(values []string, limit, maxChars int) []string {
	result := make([]string, 0, min(limit, len(values)))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > maxChars {
			value = value[:maxChars]
		}
		result = append(result, value)
		if len(result) == limit {
			break
		}
	}
	return result
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// Kept referenced so the import remains explicit in generated docs where time
// is used by the compact summary contract rather than exposed raw records.
var _ = time.Time{}
