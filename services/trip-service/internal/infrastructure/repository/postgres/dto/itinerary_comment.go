package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

// ItineraryCommentColumns is the canonical column projection for comment rows.
const ItineraryCommentColumns = "id, trip_id, day_number, item_index, author_user_id, body, status, created_at, updated_at, deleted_at"

// ScanItineraryComment maps a single comment row to its domain entity.
func ScanItineraryComment(row pgx.Row) (*entity.ItineraryComment, error) {
	var (
		id, tripID, authorUserID pgtype.UUID
		dayNumber, itemIndex     int32
		body, status             string
		createdAt, updatedAt     pgtype.Timestamp
		deletedAt                pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&dayNumber,
		&itemIndex,
		&authorUserID,
		&body,
		&status,
		&createdAt,
		&updatedAt,
		&deletedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan itinerary comment: %w", err)
	}

	return &entity.ItineraryComment{
		ID:           uuid.UUID(id.Bytes),
		TripID:       uuid.UUID(tripID.Bytes),
		DayNumber:    int(dayNumber),
		ItemIndex:    int(itemIndex),
		AuthorUserID: uuid.UUID(authorUserID.Bytes),
		Body:         body,
		Status:       entity.CommentStatus(status),
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
		DeletedAt:    timestampPtr(deletedAt),
	}, nil
}

// ScanItineraryCommentRows maps a set of comment rows to domain entities.
func ScanItineraryCommentRows(rows pgx.Rows) ([]entity.ItineraryComment, error) {
	comments := make([]entity.ItineraryComment, 0)
	for rows.Next() {
		comment, err := ScanItineraryComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate itinerary comments: %w", err)
	}
	return comments, nil
}

// ScanItineraryCommentCounts maps grouped (day_number, item_index, count) rows.
func ScanItineraryCommentCounts(rows pgx.Rows) ([]entity.ItineraryCommentCount, error) {
	counts := make([]entity.ItineraryCommentCount, 0)
	for rows.Next() {
		var dayNumber, itemIndex int32
		var count int64
		if err := rows.Scan(&dayNumber, &itemIndex, &count); err != nil {
			return nil, fmt.Errorf("scan itinerary comment count: %w", err)
		}
		counts = append(counts, entity.ItineraryCommentCount{
			DayNumber: int(dayNumber),
			ItemIndex: int(itemIndex),
			Count:     int(count),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate itinerary comment counts: %w", err)
	}
	return counts, nil
}
