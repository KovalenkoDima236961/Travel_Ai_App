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

const TripInvitationColumns = "id, trip_id, inviter_user_id, invited_user_id, email, role, status, message, expires_at, accepted_at, declined_at, revoked_at, created_at, updated_at"

const TripSuggestionColumns = "id, trip_id, author_user_id, suggestion_type, target_type, target_id, status, before_json, after_json, comment, metadata, applied_itinerary_revision, created_at, updated_at, resolved_at, resolved_by_user_id"

const TripVoteColumns = "id, trip_id, target_type, target_id, user_id, vote_type, metadata, created_at, updated_at"

func ScanTripInvitation(row pgx.Row) (*entity.TripInvitation, error) {
	var (
		id, tripID, inviterUserID, invitedUserID pgtype.UUID
		email, message                           pgtype.Text
		role, status                             string
		expiresAt, acceptedAt, declinedAt        pgtype.Timestamp
		revokedAt, createdAt, updatedAt          pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&inviterUserID,
		&invitedUserID,
		&email,
		&role,
		&status,
		&message,
		&expiresAt,
		&acceptedAt,
		&declinedAt,
		&revokedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip invitation: %w", err)
	}

	return &entity.TripInvitation{
		ID:            uuid.UUID(id.Bytes),
		TripID:        uuid.UUID(tripID.Bytes),
		InviterUserID: uuid.UUID(inviterUserID.Bytes),
		InvitedUserID: fromPgUUID(invitedUserID),
		Email:         textValue(email),
		Role:          entity.CollaboratorRole(role),
		Status:        entity.TripInvitationStatus(status),
		Message:       textValue(message),
		ExpiresAt:     expiresAt.Time,
		AcceptedAt:    timestampPtr(acceptedAt),
		DeclinedAt:    timestampPtr(declinedAt),
		RevokedAt:     timestampPtr(revokedAt),
		CreatedAt:     createdAt.Time,
		UpdatedAt:     updatedAt.Time,
	}, nil
}

func ScanTripInvitationRows(rows pgx.Rows) ([]entity.TripInvitation, error) {
	invitations := make([]entity.TripInvitation, 0)
	for rows.Next() {
		invitation, err := ScanTripInvitation(rows)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, *invitation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip invitations: %w", err)
	}
	return invitations, nil
}

func ScanTripSuggestion(row pgx.Row) (*entity.TripSuggestion, error) {
	var (
		id, tripID, authorUserID, resolvedByUserID pgtype.UUID
		suggestionType, targetType, status         string
		targetID, comment                          pgtype.Text
		beforeRaw, afterRaw, metadataRaw           []byte
		appliedRevision                            pgtype.Int4
		createdAt, updatedAt, resolvedAt           pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&authorUserID,
		&suggestionType,
		&targetType,
		&targetID,
		&status,
		&beforeRaw,
		&afterRaw,
		&comment,
		&metadataRaw,
		&appliedRevision,
		&createdAt,
		&updatedAt,
		&resolvedAt,
		&resolvedByUserID,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip suggestion: %w", err)
	}

	var applied *int
	if appliedRevision.Valid {
		value := int(appliedRevision.Int32)
		applied = &value
	}

	return &entity.TripSuggestion{
		ID:                       uuid.UUID(id.Bytes),
		TripID:                   uuid.UUID(tripID.Bytes),
		AuthorUserID:             uuid.UUID(authorUserID.Bytes),
		SuggestionType:           entity.TripSuggestionType(suggestionType),
		TargetType:               entity.TripSuggestionTargetType(targetType),
		TargetID:                 textValue(targetID),
		Status:                   entity.TripSuggestionStatus(status),
		Before:                   rawJSONCopy(beforeRaw),
		After:                    rawJSONCopy(afterRaw),
		Comment:                  textValue(comment),
		Metadata:                 metadataMap(metadataRaw),
		AppliedItineraryRevision: applied,
		CreatedAt:                createdAt.Time,
		UpdatedAt:                updatedAt.Time,
		ResolvedAt:               timestampPtr(resolvedAt),
		ResolvedByUserID:         fromPgUUID(resolvedByUserID),
	}, nil
}

func ScanTripSuggestionRows(rows pgx.Rows) ([]entity.TripSuggestion, error) {
	suggestions := make([]entity.TripSuggestion, 0)
	for rows.Next() {
		suggestion, err := ScanTripSuggestion(rows)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, *suggestion)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip suggestions: %w", err)
	}
	return suggestions, nil
}

func ScanTripVote(row pgx.Row) (*entity.TripVote, error) {
	var (
		id, tripID, userID             pgtype.UUID
		targetType, targetID, voteType string
		metadataRaw                    []byte
		createdAt, updatedAt           pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&targetType,
		&targetID,
		&userID,
		&voteType,
		&metadataRaw,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip vote: %w", err)
	}

	return &entity.TripVote{
		ID:         uuid.UUID(id.Bytes),
		TripID:     uuid.UUID(tripID.Bytes),
		TargetType: entity.TripVoteTargetType(targetType),
		TargetID:   targetID,
		UserID:     uuid.UUID(userID.Bytes),
		VoteType:   entity.TripVoteType(voteType),
		Metadata:   metadataMap(metadataRaw),
		CreatedAt:  createdAt.Time,
		UpdatedAt:  updatedAt.Time,
	}, nil
}

func ScanTripVoteRows(rows pgx.Rows) ([]entity.TripVote, error) {
	votes := make([]entity.TripVote, 0)
	for rows.Next() {
		vote, err := ScanTripVote(rows)
		if err != nil {
			return nil, err
		}
		votes = append(votes, *vote)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip votes: %w", err)
	}
	return votes, nil
}

func RawJSONArg(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return []byte(raw)
}

func rawJSONCopy(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return json.RawMessage(out)
}
