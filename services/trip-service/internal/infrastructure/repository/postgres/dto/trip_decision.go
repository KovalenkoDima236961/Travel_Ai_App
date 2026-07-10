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

const TripPollColumns = "id, trip_id, created_by_user_id, title, description, poll_type, status, allow_multiple_votes, closes_at, metadata, created_at, updated_at, closed_at, closed_by_user_id"

const TripPollOptionColumns = "id, poll_id, option_key, label, description, sort_order, metadata, created_at"

const TripPollVoteColumns = "id, poll_id, option_id, user_id, vote_value, rating_value, metadata, created_at, updated_at"

const ItineraryItemReactionColumns = "id, trip_id, day_number, item_index, item_id, user_id, reaction, metadata, created_at, updated_at"

const DiscoverySuggestionVoteColumns = "id, session_id, suggestion_id, trip_id, user_id, vote, metadata, created_at, updated_at"

func ScanTripPoll(row pgx.Row) (*entity.TripPoll, error) {
	var (
		id, tripID, createdByUserID, closedByUserID pgtype.UUID
		title, pollType, status                     string
		description                                 pgtype.Text
		allowMultipleVotes                          bool
		closesAt, createdAt, updatedAt, closedAt    pgtype.Timestamp
		metadata                                    []byte
	)
	err := row.Scan(
		&id,
		&tripID,
		&createdByUserID,
		&title,
		&description,
		&pollType,
		&status,
		&allowMultipleVotes,
		&closesAt,
		&metadata,
		&createdAt,
		&updatedAt,
		&closedAt,
		&closedByUserID,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip poll: %w", err)
	}
	return &entity.TripPoll{
		ID:                 uuid.UUID(id.Bytes),
		TripID:             uuid.UUID(tripID.Bytes),
		CreatedByUserID:    uuid.UUID(createdByUserID.Bytes),
		Title:              title,
		Description:        textValue(description),
		PollType:           entity.PollType(pollType),
		Status:             entity.PollStatus(status),
		AllowMultipleVotes: allowMultipleVotes,
		ClosesAt:           timestampPtr(closesAt),
		Metadata:           metadataMap(metadata),
		CreatedAt:          createdAt.Time,
		UpdatedAt:          updatedAt.Time,
		ClosedAt:           timestampPtr(closedAt),
		ClosedByUserID:     uuidPtr(closedByUserID),
	}, nil
}

func ScanTripPollRows(rows pgx.Rows) ([]entity.TripPoll, error) {
	items := make([]entity.TripPoll, 0)
	for rows.Next() {
		item, err := ScanTripPoll(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip polls: %w", err)
	}
	return items, nil
}

func ScanTripPollOption(row pgx.Row) (*entity.TripPollOption, error) {
	var (
		id, pollID       pgtype.UUID
		optionKey, label string
		description      pgtype.Text
		sortOrder        int32
		metadata         []byte
		createdAt        pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&pollID,
		&optionKey,
		&label,
		&description,
		&sortOrder,
		&metadata,
		&createdAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip poll option: %w", err)
	}
	return &entity.TripPollOption{
		ID:          uuid.UUID(id.Bytes),
		PollID:      uuid.UUID(pollID.Bytes),
		OptionKey:   optionKey,
		Label:       label,
		Description: textValue(description),
		SortOrder:   int(sortOrder),
		Metadata:    metadataMap(metadata),
		CreatedAt:   createdAt.Time,
	}, nil
}

func ScanTripPollOptionRows(rows pgx.Rows) ([]entity.TripPollOption, error) {
	items := make([]entity.TripPollOption, 0)
	for rows.Next() {
		item, err := ScanTripPollOption(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip poll options: %w", err)
	}
	return items, nil
}

func ScanTripPollVote(row pgx.Row) (*entity.TripPollVote, error) {
	var (
		id, pollID, optionID, userID pgtype.UUID
		voteValue                    pgtype.Text
		ratingValue                  pgtype.Int4
		metadata                     []byte
		createdAt, updatedAt         pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&pollID,
		&optionID,
		&userID,
		&voteValue,
		&ratingValue,
		&metadata,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip poll vote: %w", err)
	}
	return &entity.TripPollVote{
		ID:          uuid.UUID(id.Bytes),
		PollID:      uuid.UUID(pollID.Bytes),
		OptionID:    uuidPtr(optionID),
		UserID:      uuid.UUID(userID.Bytes),
		VoteValue:   textValue(voteValue),
		RatingValue: intPtr(ratingValue),
		Metadata:    metadataMap(metadata),
		CreatedAt:   createdAt.Time,
		UpdatedAt:   updatedAt.Time,
	}, nil
}

func ScanTripPollVoteRows(rows pgx.Rows) ([]entity.TripPollVote, error) {
	items := make([]entity.TripPollVote, 0)
	for rows.Next() {
		item, err := ScanTripPollVote(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip poll votes: %w", err)
	}
	return items, nil
}

func ScanItineraryItemReaction(row pgx.Row) (*entity.ItineraryItemReaction, error) {
	var (
		id, tripID, userID   pgtype.UUID
		dayNumber, itemIndex int32
		itemID               pgtype.Text
		reaction             string
		metadata             []byte
		createdAt, updatedAt pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&tripID,
		&dayNumber,
		&itemIndex,
		&itemID,
		&userID,
		&reaction,
		&metadata,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan itinerary item reaction: %w", err)
	}
	return &entity.ItineraryItemReaction{
		ID:        uuid.UUID(id.Bytes),
		TripID:    uuid.UUID(tripID.Bytes),
		DayNumber: int(dayNumber),
		ItemIndex: int(itemIndex),
		ItemID:    textValue(itemID),
		UserID:    uuid.UUID(userID.Bytes),
		Reaction:  entity.ItineraryReaction(reaction),
		Metadata:  metadataMap(metadata),
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}, nil
}

func ScanItineraryItemReactionRows(rows pgx.Rows) ([]entity.ItineraryItemReaction, error) {
	items := make([]entity.ItineraryItemReaction, 0)
	for rows.Next() {
		item, err := ScanItineraryItemReaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate itinerary item reactions: %w", err)
	}
	return items, nil
}

func ScanDiscoverySuggestionVote(row pgx.Row) (*entity.DiscoverySuggestionVote, error) {
	var (
		id, sessionID, tripID, userID pgtype.UUID
		suggestionID, vote            string
		metadata                      []byte
		createdAt, updatedAt          pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&sessionID,
		&suggestionID,
		&tripID,
		&userID,
		&vote,
		&metadata,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan discovery suggestion vote: %w", err)
	}
	return &entity.DiscoverySuggestionVote{
		ID:           uuid.UUID(id.Bytes),
		SessionID:    uuid.UUID(sessionID.Bytes),
		SuggestionID: suggestionID,
		TripID:       uuidPtr(tripID),
		UserID:       uuid.UUID(userID.Bytes),
		Vote:         entity.DiscoverySuggestionVoteValue(vote),
		Metadata:     metadataMap(metadata),
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}, nil
}

func ScanDiscoverySuggestionVoteRows(rows pgx.Rows) ([]entity.DiscoverySuggestionVote, error) {
	items := make([]entity.DiscoverySuggestionVote, 0)
	for rows.Next() {
		item, err := ScanDiscoverySuggestionVote(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discovery suggestion votes: %w", err)
	}
	return items, nil
}

func JSONBArg(metadata map[string]any) ([]byte, error) {
	if len(metadata) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return raw, nil
}

func TextNullableArg(value string) pgtype.Text {
	return toPgText(value)
}

func UUIDNullableArg(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return toPgUUID(*id)
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func uuidPtr(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func intPtr(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int32)
	return &result
}

func metadataMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
