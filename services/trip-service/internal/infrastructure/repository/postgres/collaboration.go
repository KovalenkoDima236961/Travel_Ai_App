package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

func (r *Repository) CreateTripInvitation(ctx context.Context, invitation *entity.TripInvitation) (*entity.TripInvitation, error) {
	query, args, err := r.db.Builder.
		Insert("trip_invitations").
		Columns(
			"id", "trip_id", "inviter_user_id", "invited_user_id", "email",
			"role", "status", "message", "expires_at",
		).
		Values(
			dto.IDArg(invitation.ID),
			dto.IDArg(invitation.TripID),
			dto.IDArg(invitation.InviterUserID),
			dto.IDArgPtr(invitation.InvitedUserID),
			dto.TextArg(invitation.Email),
			string(invitation.Role),
			string(invitation.Status),
			dto.TextArg(invitation.Message),
			invitation.ExpiresAt.UTC(),
		).
		Suffix("RETURNING " + dto.TripInvitationColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip invitation: %w", err)
	}

	created, err := dto.ScanTripInvitation(r.db.QueryRow(ctx, query, args...))
	if storage.UniqueConstraintViolation(err) {
		return nil, domainerrs.ErrConflict
	}
	return created, err
}

func (r *Repository) ListTripInvitationsByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripInvitation, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripInvitationColumns).
		From("trip_invitations").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip invitations: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip invitations: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripInvitationRows(rows)
}

func (r *Repository) ListPendingTripInvitationsForUser(ctx context.Context, userID uuid.UUID, email string) ([]entity.TripInvitation, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	conditions := sq.Or{sq.Eq{"invited_user_id": dto.IDArg(userID)}}
	if email != "" {
		conditions = append(conditions, sq.Expr("lower(email) = ?", email))
	}

	query, args, err := r.db.Builder.
		Select(dto.TripInvitationColumns).
		From("trip_invitations").
		Where(sq.Eq{"status": string(entity.TripInvitationStatusPending)}).
		Where(conditions).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list pending trip invitations: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending trip invitations: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripInvitationRows(rows)
}

func (r *Repository) GetTripInvitationByID(ctx context.Context, tripID, invitationID uuid.UUID) (*entity.TripInvitation, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripInvitationColumns).
		From("trip_invitations").
		Where(sq.Eq{
			"id":      dto.IDArg(invitationID),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip invitation: %w", err)
	}
	return dto.ScanTripInvitation(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) LinkTripInvitationToUser(ctx context.Context, invitationID, userID uuid.UUID, email string) (*entity.TripInvitation, error) {
	query, args, err := r.db.Builder.
		Update("trip_invitations").
		Set("invited_user_id", dto.IDArg(userID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(invitationID),
			"status": string(entity.TripInvitationStatusPending),
		}).
		Where(sq.Expr("invited_user_id IS NULL")).
		Where(sq.Expr("lower(email) = ?", strings.ToLower(strings.TrimSpace(email)))).
		Suffix("RETURNING " + dto.TripInvitationColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build link trip invitation: %w", err)
	}
	return dto.ScanTripInvitation(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateTripInvitationStatus(
	ctx context.Context,
	tripID, invitationID uuid.UUID,
	status entity.TripInvitationStatus,
	actorUserID *uuid.UUID,
	appliedAt time.Time,
) (*entity.TripInvitation, error) {
	builder := r.db.Builder.
		Update("trip_invitations").
		Set("status", string(status)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(invitationID),
			"trip_id": dto.IDArg(tripID),
		})

	switch status {
	case entity.TripInvitationStatusAccepted:
		builder = builder.Set("accepted_at", appliedAt.UTC())
	case entity.TripInvitationStatusDeclined:
		builder = builder.Set("declined_at", appliedAt.UTC())
	case entity.TripInvitationStatusRevoked:
		builder = builder.Set("revoked_at", appliedAt.UTC())
	case entity.TripInvitationStatusExpired:
		// Expiration is represented by status plus the existing expires_at.
	case entity.TripInvitationStatusPending:
		builder = builder.
			Set("accepted_at", nil).
			Set("declined_at", nil).
			Set("revoked_at", nil)
	}
	_ = actorUserID

	query, args, err := builder.Suffix("RETURNING " + dto.TripInvitationColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip invitation status: %w", err)
	}
	return dto.ScanTripInvitation(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ResendTripInvitation(
	ctx context.Context,
	tripID, invitationID, inviterUserID uuid.UUID,
	expiresAt time.Time,
	message string,
) (*entity.TripInvitation, error) {
	query, args, err := r.db.Builder.
		Update("trip_invitations").
		Set("inviter_user_id", dto.IDArg(inviterUserID)).
		Set("status", string(entity.TripInvitationStatusPending)).
		Set("message", dto.TextArg(message)).
		Set("expires_at", expiresAt.UTC()).
		Set("accepted_at", nil).
		Set("declined_at", nil).
		Set("revoked_at", nil).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(invitationID),
			"trip_id": dto.IDArg(tripID),
		}).
		Where(sq.NotEq{"status": string(entity.TripInvitationStatusAccepted)}).
		Suffix("RETURNING " + dto.TripInvitationColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build resend trip invitation: %w", err)
	}
	return dto.ScanTripInvitation(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) RemoveTripCollaboratorByUser(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Update("trip_collaborators").
		Set("status", string(entity.CollaboratorStatusRemoved)).
		Set("removed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		Where(sq.NotEq{"status": string(entity.CollaboratorStatusRemoved)}).
		Suffix("RETURNING " + dto.TripCollaboratorColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build remove trip collaborator by user: %w", err)
	}
	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) TransferTripOwnership(ctx context.Context, tripID, currentOwnerID, nextOwnerID uuid.UUID) (*entity.Trip, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transfer ownership tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	updateTripSQL, updateTripArgs, err := r.db.Builder.
		Update("trips").
		Set("user_id", dto.IDArg(nextOwnerID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(tripID),
			"user_id": dto.IDArg(currentOwnerID),
		}).
		Suffix("RETURNING " + dto.Columns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build transfer ownership trip update: %w", err)
	}
	updated, err := dto.Scan(tx.QueryRow(ctx, updateTripSQL, updateTripArgs...))
	if err != nil {
		return nil, err
	}

	oldOwnerSQL, oldOwnerArgs, err := r.db.Builder.
		Insert("trip_collaborators").
		Columns("id", "trip_id", "user_id", "role", "status", "invited_by_user_id", "accepted_at", "permissions").
		Values(
			dto.IDArg(uuid.New()),
			dto.IDArg(tripID),
			dto.IDArg(currentOwnerID),
			string(entity.CollaboratorRoleEditor),
			string(entity.CollaboratorStatusAccepted),
			dto.IDArg(currentOwnerID),
			time.Now().UTC(),
			dto.JSONArg(map[string]any{}),
		).
		Suffix(
			"ON CONFLICT (trip_id, user_id) DO UPDATE SET " +
				"role = EXCLUDED.role, status = EXCLUDED.status, " +
				"accepted_at = COALESCE(trip_collaborators.accepted_at, NOW()), " +
				"removed_at = NULL, revoked_at = NULL, declined_at = NULL, updated_at = NOW()",
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build old owner collaborator upsert: %w", err)
	}
	if _, err := tx.Exec(ctx, oldOwnerSQL, oldOwnerArgs...); err != nil {
		return nil, fmt.Errorf("upsert old owner collaborator: %w", err)
	}

	removeNewOwnerSQL, removeNewOwnerArgs, err := r.db.Builder.
		Update("trip_collaborators").
		Set("status", string(entity.CollaboratorStatusRemoved)).
		Set("removed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(nextOwnerID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build remove new owner collaborator: %w", err)
	}
	if _, err := tx.Exec(ctx, removeNewOwnerSQL, removeNewOwnerArgs...); err != nil {
		return nil, fmt.Errorf("remove new owner collaborator: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transfer ownership tx: %w", err)
	}
	committed = true
	return updated, nil
}

func (r *Repository) CreateTripSuggestion(ctx context.Context, suggestion *entity.TripSuggestion) (*entity.TripSuggestion, error) {
	metadata, err := dto.JSONBArg(suggestion.Metadata)
	if err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Insert("trip_suggestions").
		Columns(
			"id", "trip_id", "author_user_id", "suggestion_type", "target_type", "target_id",
			"status", "before_json", "after_json", "comment", "metadata",
		).
		Values(
			dto.IDArg(suggestion.ID),
			dto.IDArg(suggestion.TripID),
			dto.IDArg(suggestion.AuthorUserID),
			string(suggestion.SuggestionType),
			string(suggestion.TargetType),
			dto.TextArg(suggestion.TargetID),
			string(entity.TripSuggestionStatusOpen),
			dto.RawJSONArg(suggestion.Before),
			dto.RawJSONArg(suggestion.After),
			dto.TextArg(suggestion.Comment),
			metadata,
		).
		Suffix("RETURNING " + dto.TripSuggestionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip suggestion: %w", err)
	}
	return dto.ScanTripSuggestion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripSuggestionsByTrip(ctx context.Context, tripID uuid.UUID, status *entity.TripSuggestionStatus) ([]entity.TripSuggestion, error) {
	builder := r.db.Builder.
		Select(dto.TripSuggestionColumns).
		From("trip_suggestions").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)})
	if status != nil {
		builder = builder.Where(sq.Eq{"status": string(*status)})
	}
	query, args, err := builder.OrderBy("created_at DESC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip suggestions: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip suggestions: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripSuggestionRows(rows)
}

func (r *Repository) GetTripSuggestionByID(ctx context.Context, tripID, suggestionID uuid.UUID) (*entity.TripSuggestion, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripSuggestionColumns).
		From("trip_suggestions").
		Where(sq.Eq{
			"id":      dto.IDArg(suggestionID),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip suggestion: %w", err)
	}
	return dto.ScanTripSuggestion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateTripSuggestionStatus(
	ctx context.Context,
	tripID, suggestionID, actorUserID uuid.UUID,
	status entity.TripSuggestionStatus,
	appliedRevision *int,
) (*entity.TripSuggestion, error) {
	builder := r.db.Builder.
		Update("trip_suggestions").
		Set("status", string(status)).
		Set("resolved_at", sq.Expr("NOW()")).
		Set("resolved_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(suggestionID),
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.TripSuggestionStatusOpen),
		})
	if appliedRevision != nil {
		builder = builder.Set("applied_itinerary_revision", *appliedRevision)
	}
	query, args, err := builder.Suffix("RETURNING " + dto.TripSuggestionColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip suggestion status: %w", err)
	}
	return dto.ScanTripSuggestion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpsertTripVote(ctx context.Context, vote *entity.TripVote) (*entity.TripVote, error) {
	metadata, err := dto.JSONBArg(vote.Metadata)
	if err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Insert("trip_votes").
		Columns("id", "trip_id", "target_type", "target_id", "user_id", "vote_type", "metadata").
		Values(
			dto.IDArg(vote.ID),
			dto.IDArg(vote.TripID),
			string(vote.TargetType),
			vote.TargetID,
			dto.IDArg(vote.UserID),
			string(vote.VoteType),
			metadata,
		).
		Suffix(
			"ON CONFLICT (trip_id, target_type, target_id, user_id) DO UPDATE SET " +
				"vote_type = EXCLUDED.vote_type, metadata = EXCLUDED.metadata, updated_at = NOW() " +
				"RETURNING " + dto.TripVoteColumns,
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert trip vote: %w", err)
	}
	return dto.ScanTripVote(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripVoteByID(ctx context.Context, tripID, voteID uuid.UUID) (*entity.TripVote, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripVoteColumns).
		From("trip_votes").
		Where(sq.Eq{
			"id":      dto.IDArg(voteID),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip vote: %w", err)
	}
	return dto.ScanTripVote(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) DeleteTripVote(ctx context.Context, tripID, voteID, userID uuid.UUID) error {
	query, args, err := r.db.Builder.
		Delete("trip_votes").
		Where(sq.Eq{
			"id":      dto.IDArg(voteID),
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete trip vote: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete trip vote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerrs.ErrNotFound
	}
	return nil
}

func (r *Repository) ListTripVotesByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripVote, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripVoteColumns).
		From("trip_votes").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip votes: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip votes: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripVoteRows(rows)
}

func (r *Repository) ListTripVotesByTarget(ctx context.Context, tripID uuid.UUID, targetType entity.TripVoteTargetType, targetID string) ([]entity.TripVote, error) {
	if strings.TrimSpace(targetID) == "" {
		return nil, apperrs.NewInvalidInput("targetId is required")
	}
	query, args, err := r.db.Builder.
		Select(dto.TripVoteColumns).
		From("trip_votes").
		Where(sq.Eq{
			"trip_id":     dto.IDArg(tripID),
			"target_type": string(targetType),
			"target_id":   targetID,
		}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip votes by target: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip votes by target: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripVoteRows(rows)
}
