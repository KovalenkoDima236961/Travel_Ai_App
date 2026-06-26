package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

func (r *Repository) UpsertTripCalendarSync(ctx context.Context, sync *entity.TripCalendarSync) (*entity.TripCalendarSync, error) {
	query := `
INSERT INTO trip_calendar_syncs (
    id, trip_id, user_id, provider, external_calendar_id, external_event_id,
    external_event_link, day_number, item_index, itinerary_revision, sync_key,
    status, last_synced_at, created_at, updated_at, deleted_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'active', NOW(), NOW(), NOW(), NULL)
ON CONFLICT (trip_id, user_id, provider, sync_key) DO UPDATE SET
    external_calendar_id = EXCLUDED.external_calendar_id,
    external_event_id = EXCLUDED.external_event_id,
    external_event_link = EXCLUDED.external_event_link,
    day_number = EXCLUDED.day_number,
    item_index = EXCLUDED.item_index,
    itinerary_revision = EXCLUDED.itinerary_revision,
    status = 'active',
    last_synced_at = NOW(),
    updated_at = NOW(),
    deleted_at = NULL
RETURNING id, trip_id, user_id, provider, external_calendar_id, external_event_id,
          external_event_link, day_number, item_index, itinerary_revision, sync_key,
          status, last_synced_at, created_at, updated_at, deleted_at`
	return scanTripCalendarSync(r.db.QueryRow(
		ctx,
		query,
		sync.ID,
		sync.TripID,
		sync.UserID,
		sync.Provider,
		sync.ExternalCalendarID,
		sync.ExternalEventID,
		sync.ExternalEventLink,
		sync.DayNumber,
		sync.ItemIndex,
		sync.ItineraryRevision,
		sync.SyncKey,
	))
}

func (r *Repository) ListTripCalendarSyncsByTripUserProvider(ctx context.Context, tripID, userID uuid.UUID, provider string) ([]entity.TripCalendarSync, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, trip_id, user_id, provider, external_calendar_id, external_event_id,
		        external_event_link, day_number, item_index, itinerary_revision, sync_key,
		        status, last_synced_at, created_at, updated_at, deleted_at
		 FROM trip_calendar_syncs
		 WHERE trip_id = $1 AND user_id = $2 AND provider = $3 AND status = 'active' AND deleted_at IS NULL
		 ORDER BY day_number ASC, item_index ASC`,
		tripID,
		userID,
		provider,
	)
	if err != nil {
		return nil, fmt.Errorf("query trip calendar syncs: %w", err)
	}
	defer rows.Close()

	out := make([]entity.TripCalendarSync, 0)
	for rows.Next() {
		sync, err := scanTripCalendarSync(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sync)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip calendar syncs: %w", err)
	}
	return out, nil
}

func (r *Repository) GetTripCalendarSyncStatus(ctx context.Context, tripID, userID uuid.UUID, provider string) (int, *time.Time, int, error) {
	row := r.db.QueryRow(
		ctx,
		`SELECT COUNT(*), MAX(last_synced_at), MAX(itinerary_revision)
		 FROM trip_calendar_syncs
		 WHERE trip_id = $1 AND user_id = $2 AND provider = $3 AND status = 'active' AND deleted_at IS NULL`,
		tripID,
		userID,
		provider,
	)
	var count int
	var last sql.NullTime
	var revision sql.NullInt64
	if err := row.Scan(&count, &last, &revision); err != nil {
		return 0, nil, 0, fmt.Errorf("scan trip calendar sync status: %w", err)
	}
	var lastPtr *time.Time
	if last.Valid {
		v := last.Time
		lastPtr = &v
	}
	return count, lastPtr, int(revision.Int64), nil
}

func (r *Repository) GetActiveTripCalendarSyncByKey(ctx context.Context, tripID, userID uuid.UUID, provider, syncKey string) (*entity.TripCalendarSync, error) {
	query := `
SELECT id, trip_id, user_id, provider, external_calendar_id, external_event_id,
       external_event_link, day_number, item_index, itinerary_revision, sync_key,
       status, last_synced_at, created_at, updated_at, deleted_at
FROM trip_calendar_syncs
WHERE trip_id = $1 AND user_id = $2 AND provider = $3 AND sync_key = $4
  AND status = 'active' AND deleted_at IS NULL`
	sync, err := scanTripCalendarSync(r.db.QueryRow(ctx, query, tripID, userID, provider, syncKey))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, err
	}
	return sync, nil
}

func (r *Repository) MarkTripCalendarSyncDeleted(ctx context.Context, tripID, userID uuid.UUID, provider, syncKey string) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE trip_calendar_syncs
		 SET status = 'deleted', deleted_at = NOW(), updated_at = NOW()
		 WHERE trip_id = $1 AND user_id = $2 AND provider = $3 AND sync_key = $4`,
		tripID,
		userID,
		provider,
		syncKey,
	)
	if err != nil {
		return fmt.Errorf("mark trip calendar sync deleted: %w", err)
	}
	return nil
}

func (r *Repository) MarkAllTripCalendarSyncsDeleted(ctx context.Context, tripID, userID uuid.UUID, provider string) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE trip_calendar_syncs
		 SET status = 'deleted', deleted_at = NOW(), updated_at = NOW()
		 WHERE trip_id = $1 AND user_id = $2 AND provider = $3 AND status = 'active'`,
		tripID,
		userID,
		provider,
	)
	if err != nil {
		return fmt.Errorf("mark all trip calendar syncs deleted: %w", err)
	}
	return nil
}

func scanTripCalendarSync(row pgx.Row) (*entity.TripCalendarSync, error) {
	var out entity.TripCalendarSync
	var link sql.NullString
	var deletedAt sql.NullTime
	if err := row.Scan(
		&out.ID,
		&out.TripID,
		&out.UserID,
		&out.Provider,
		&out.ExternalCalendarID,
		&out.ExternalEventID,
		&link,
		&out.DayNumber,
		&out.ItemIndex,
		&out.ItineraryRevision,
		&out.SyncKey,
		&out.Status,
		&out.LastSyncedAt,
		&out.CreatedAt,
		&out.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, fmt.Errorf("scan trip calendar sync: %w", err)
	}
	if link.Valid {
		v := link.String
		out.ExternalEventLink = &v
	}
	if deletedAt.Valid {
		v := deletedAt.Time
		out.DeletedAt = &v
	}
	return &out, nil
}
