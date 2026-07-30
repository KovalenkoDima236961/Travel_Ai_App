package dto

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

// ItineraryCommentColumns is the canonical column projection for comment rows.
const ItineraryCommentColumns = "id, trip_id, day_number, item_index, target_type, target_id, parent_comment_id, author_user_id, body, status, mentions, attachments, resolved_at, resolved_by_user_id, created_at, updated_at, edited_at, deleted_at"

// ScanItineraryComment maps a single comment row to its domain entity.
func ScanItineraryComment(row pgx.Row) (*entity.ItineraryComment, error) {
	var (
		id, tripID, parentCommentID, authorUserID, resolvedByUserID pgtype.UUID
		dayNumber, itemIndex                                        int32
		targetID                                                    pgtype.Text
		targetType, body, status                                    string
		mentionsRaw, attachmentsRaw                                 []byte
		resolvedAt, createdAt, updatedAt, editedAt, deletedAt       pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&dayNumber,
		&itemIndex,
		&targetType,
		&targetID,
		&parentCommentID,
		&authorUserID,
		&body,
		&status,
		&mentionsRaw,
		&attachmentsRaw,
		&resolvedAt,
		&resolvedByUserID,
		&createdAt,
		&updatedAt,
		&editedAt,
		&deletedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan itinerary comment: %w", err)
	}

	mentions, err := unmarshalUUIDSlice(mentionsRaw, "itinerary comment mentions")
	if err != nil {
		return nil, err
	}
	attachments, err := unmarshalStringSlice(attachmentsRaw, "itinerary comment attachments")
	if err != nil {
		return nil, err
	}

	return &entity.ItineraryComment{
		ID:           uuid.UUID(id.Bytes),
		TripID:       uuid.UUID(tripID.Bytes),
		DayNumber:    int(dayNumber),
		ItemIndex:    int(itemIndex),
		TargetType:   entity.CommentTargetType(targetType),
		TargetID:     commentTextValue(targetID),
		ParentID:     fromPgUUID(parentCommentID),
		AuthorUserID: uuid.UUID(authorUserID.Bytes),
		Body:         body,
		Status:       entity.CommentStatus(status),
		Mentions:     mentions,
		Attachments:  attachments,
		ResolvedAt:   timestampPtr(resolvedAt),
		ResolvedBy:   fromPgUUID(resolvedByUserID),
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
		EditedAt:     timestampPtr(editedAt),
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

func unmarshalUUIDSlice(raw []byte, label string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return []uuid.UUID{}, nil
	}
	var out []uuid.UUID
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", label, err)
	}
	if out == nil {
		out = []uuid.UUID{}
	}
	return out, nil
}

func commentTextValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
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
