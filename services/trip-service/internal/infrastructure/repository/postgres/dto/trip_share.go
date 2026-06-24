package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

// TripShareColumns is the canonical column order for trip_shares.
const TripShareColumns = "id, trip_id, user_id, share_token, enabled, created_at, disabled_at"

// TripShareInsertColumns returns the non-default columns set on INSERT.
func TripShareInsertColumns() []string {
	return []string{"trip_id", "user_id", "share_token", "enabled"}
}

// TripShareInsertValues returns values matching TripShareInsertColumns.
func TripShareInsertValues(s *entity.TripShare) []any {
	return []any{
		toPgUUID(s.TripID),
		toPgUUID(s.UserID),
		s.ShareToken,
		s.Enabled,
	}
}

// ScanTripShare reads one trip_shares row.
func ScanTripShare(row pgx.Row) (*entity.TripShare, error) {
	var (
		id, tripID, userID pgtype.UUID
		shareToken         string
		enabled            bool
		createdAt          pgtype.Timestamp
		disabledAt         pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&userID,
		&shareToken,
		&enabled,
		&createdAt,
		&disabledAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip share: %w", err)
	}

	return &entity.TripShare{
		ID:         uuid.UUID(id.Bytes),
		TripID:     uuid.UUID(tripID.Bytes),
		UserID:     uuid.UUID(userID.Bytes),
		ShareToken: shareToken,
		Enabled:    enabled,
		CreatedAt:  createdAt.Time,
		DisabledAt: fromPgTimestamp(disabledAt),
	}, nil
}

func fromPgTimestamp(ts pgtype.Timestamp) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
