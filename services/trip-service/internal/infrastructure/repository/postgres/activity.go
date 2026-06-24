package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

// CreateTripActivityEvent inserts one activity event and returns the stored row.
func (r *Repository) CreateTripActivityEvent(ctx context.Context, event *entity.TripActivityEvent) (*entity.TripActivityEvent, error) {
	values, err := dto.TripActivityEventInsertValues(event)
	if err != nil {
		return nil, err
	}

	query, args, err := r.db.Builder.
		Insert("trip_activity_events").
		Columns(dto.TripActivityEventInsertColumns()...).
		Values(values...).
		Suffix("RETURNING " + dto.TripActivityEventColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert trip activity event: %w", err)
	}

	return dto.ScanTripActivityEvent(r.db.QueryRow(ctx, query, args...))
}

// ListTripActivityEvents returns a trip's activity newest first (created_at DESC,
// id DESC). When a cursor is supplied it returns only rows strictly older than
// the cursor position, giving stable keyset pagination over (created_at, id).
func (r *Repository) ListTripActivityEvents(
	ctx context.Context,
	tripID uuid.UUID,
	limit int,
	cursorCreatedAt *time.Time,
	cursorID *uuid.UUID,
) ([]entity.TripActivityEvent, error) {
	builder := r.db.Builder.
		Select(dto.TripActivityEventColumns).
		From("trip_activity_events").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)})

	if cursorCreatedAt != nil && cursorID != nil {
		builder = builder.Where(sq.Or{
			sq.Lt{"created_at": *cursorCreatedAt},
			sq.And{
				sq.Eq{"created_at": *cursorCreatedAt},
				sq.Lt{"id": dto.IDArg(*cursorID)},
			},
		})
	}

	query, args, err := builder.
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip activity events: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip activity events: %w", err)
	}
	defer rows.Close()

	return dto.ScanTripActivityEventRows(rows)
}
