package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateTripPollWithOptions(
	ctx context.Context,
	poll *entity.TripPoll,
	options []entity.TripPollOption,
) (*entity.TripPoll, []entity.TripPollOption, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin trip poll tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	created, err := r.createTripPoll(ctx, tx, poll)
	if err != nil {
		return nil, nil, err
	}
	createdOptions := make([]entity.TripPollOption, 0, len(options))
	for i := range options {
		options[i].PollID = created.ID
		option, err := r.createTripPollOption(ctx, tx, &options[i])
		if err != nil {
			return nil, nil, err
		}
		createdOptions = append(createdOptions, *option)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit trip poll tx: %w", err)
	}
	committed = true
	return created, createdOptions, nil
}

func (r *Repository) createTripPoll(
	ctx context.Context,
	q rowQuerier,
	poll *entity.TripPoll,
) (*entity.TripPoll, error) {
	metadata, err := dto.JSONBArg(poll.Metadata)
	if err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Insert("trip_polls").
		Columns(
			"id",
			"trip_id",
			"created_by_user_id",
			"title",
			"description",
			"poll_type",
			"status",
			"allow_multiple_votes",
			"closes_at",
			"metadata",
		).
		Values(
			dto.IDArg(poll.ID),
			dto.IDArg(poll.TripID),
			dto.IDArg(poll.CreatedByUserID),
			poll.Title,
			dto.TextNullableArg(poll.Description),
			string(poll.PollType),
			string(poll.Status),
			poll.AllowMultipleVotes,
			poll.ClosesAt,
			metadata,
		).
		Suffix("RETURNING " + dto.TripPollColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip poll: %w", err)
	}
	return dto.ScanTripPoll(q.QueryRow(ctx, query, args...))
}

func (r *Repository) createTripPollOption(
	ctx context.Context,
	q rowQuerier,
	option *entity.TripPollOption,
) (*entity.TripPollOption, error) {
	metadata, err := dto.JSONBArg(option.Metadata)
	if err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Insert("trip_poll_options").
		Columns("id", "poll_id", "option_key", "label", "description", "sort_order", "metadata").
		Values(
			dto.IDArg(option.ID),
			dto.IDArg(option.PollID),
			option.OptionKey,
			option.Label,
			dto.TextNullableArg(option.Description),
			option.SortOrder,
			metadata,
		).
		Suffix("RETURNING " + dto.TripPollOptionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip poll option: %w", err)
	}
	return dto.ScanTripPollOption(q.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripPollByID(ctx context.Context, tripID, pollID uuid.UUID) (*entity.TripPoll, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripPollColumns).
		From("trip_polls").
		Where(sq.Eq{"id": dto.IDArg(pollID), "trip_id": dto.IDArg(tripID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip poll: %w", err)
	}
	return dto.ScanTripPoll(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripPollsByTrip(ctx context.Context, tripID uuid.UUID, includeArchived bool) ([]entity.TripPoll, error) {
	builder := r.db.Builder.
		Select(dto.TripPollColumns).
		From("trip_polls").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)})
	if !includeArchived {
		builder = builder.Where(sq.NotEq{"status": string(entity.PollStatusArchived)})
	}
	query, args, err := builder.
		OrderBy("status = 'open' DESC", "created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip polls: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip polls: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripPollRows(rows)
}

func (r *Repository) ListPollOptions(ctx context.Context, pollID uuid.UUID) ([]entity.TripPollOption, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripPollOptionColumns).
		From("trip_poll_options").
		Where(sq.Eq{"poll_id": dto.IDArg(pollID)}).
		OrderBy("sort_order ASC", "created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list poll options: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query poll options: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripPollOptionRows(rows)
}

func (r *Repository) ListPollVotesByPoll(ctx context.Context, pollID uuid.UUID) ([]entity.TripPollVote, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripPollVoteColumns).
		From("trip_poll_votes").
		Where(sq.Eq{"poll_id": dto.IDArg(pollID)}).
		OrderBy("created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list poll votes: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query poll votes: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripPollVoteRows(rows)
}

func (r *Repository) ReplaceUserPollVotes(
	ctx context.Context,
	pollID, userID uuid.UUID,
	votes []entity.TripPollVote,
) ([]entity.TripPollVote, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin poll vote tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if _, err := tx.Exec(ctx, "DELETE FROM trip_poll_votes WHERE poll_id = $1 AND user_id = $2", pollID, userID); err != nil {
		return nil, fmt.Errorf("delete user poll votes: %w", err)
	}
	created := make([]entity.TripPollVote, 0, len(votes))
	for i := range votes {
		votes[i].PollID = pollID
		votes[i].UserID = userID
		vote, err := r.insertPollVote(ctx, tx, &votes[i])
		if err != nil {
			return nil, err
		}
		created = append(created, *vote)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit poll vote tx: %w", err)
	}
	committed = true
	return created, nil
}

func (r *Repository) insertPollVote(
	ctx context.Context,
	q rowQuerier,
	vote *entity.TripPollVote,
) (*entity.TripPollVote, error) {
	metadata, err := dto.JSONBArg(vote.Metadata)
	if err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Insert("trip_poll_votes").
		Columns("id", "poll_id", "option_id", "user_id", "vote_value", "rating_value", "metadata").
		Values(
			dto.IDArg(vote.ID),
			dto.IDArg(vote.PollID),
			dto.UUIDNullableArg(vote.OptionID),
			dto.IDArg(vote.UserID),
			dto.TextNullableArg(vote.VoteValue),
			vote.RatingValue,
			metadata,
		).
		Suffix("RETURNING " + dto.TripPollVoteColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert poll vote: %w", err)
	}
	return dto.ScanTripPollVote(q.QueryRow(ctx, query, args...))
}

func (r *Repository) CloseTripPoll(ctx context.Context, tripID, pollID, actorUserID uuid.UUID) (*entity.TripPoll, error) {
	query, args, err := r.db.Builder.
		Update("trip_polls").
		Set("status", string(entity.PollStatusClosed)).
		Set("closed_at", sq.Expr("NOW()")).
		Set("closed_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(pollID), "trip_id": dto.IDArg(tripID)}).
		Suffix("RETURNING " + dto.TripPollColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build close trip poll: %w", err)
	}
	return dto.ScanTripPoll(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ArchiveTripPoll(ctx context.Context, tripID, pollID uuid.UUID) (*entity.TripPoll, error) {
	query, args, err := r.db.Builder.
		Update("trip_polls").
		Set("status", string(entity.PollStatusArchived)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(pollID), "trip_id": dto.IDArg(tripID)}).
		Suffix("RETURNING " + dto.TripPollColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build archive trip poll: %w", err)
	}
	return dto.ScanTripPoll(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpsertItineraryItemReaction(
	ctx context.Context,
	reaction *entity.ItineraryItemReaction,
) (*entity.ItineraryItemReaction, error) {
	metadata, err := dto.JSONBArg(reaction.Metadata)
	if err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Insert("itinerary_item_reactions").
		Columns(
			"id",
			"trip_id",
			"day_number",
			"item_index",
			"item_id",
			"user_id",
			"reaction",
			"metadata",
		).
		Values(
			dto.IDArg(reaction.ID),
			dto.IDArg(reaction.TripID),
			reaction.DayNumber,
			reaction.ItemIndex,
			dto.TextNullableArg(reaction.ItemID),
			dto.IDArg(reaction.UserID),
			string(reaction.Reaction),
			metadata,
		).
		Suffix(
			"ON CONFLICT (trip_id, day_number, item_index, user_id) DO UPDATE SET " +
				"item_id = EXCLUDED.item_id, reaction = EXCLUDED.reaction, metadata = EXCLUDED.metadata, updated_at = NOW() " +
				"RETURNING " + dto.ItineraryItemReactionColumns,
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert itinerary item reaction: %w", err)
	}
	return dto.ScanItineraryItemReaction(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) DeleteItineraryItemReaction(
	ctx context.Context,
	tripID uuid.UUID,
	dayNumber int,
	itemIndex int,
	userID uuid.UUID,
) error {
	query, args, err := r.db.Builder.
		Delete("itinerary_item_reactions").
		Where(sq.Eq{
			"trip_id":    dto.IDArg(tripID),
			"day_number": dayNumber,
			"item_index": itemIndex,
			"user_id":    dto.IDArg(userID),
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete itinerary item reaction: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete itinerary item reaction: %w", err)
	}
	return nil
}

func (r *Repository) ListItineraryItemReactionsByTrip(
	ctx context.Context,
	tripID uuid.UUID,
) ([]entity.ItineraryItemReaction, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryItemReactionColumns).
		From("itinerary_item_reactions").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("day_number ASC", "item_index ASC", "created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list itinerary item reactions by trip: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query itinerary item reactions by trip: %w", err)
	}
	defer rows.Close()
	return dto.ScanItineraryItemReactionRows(rows)
}

func (r *Repository) ListItineraryItemReactionsByItem(
	ctx context.Context,
	tripID uuid.UUID,
	dayNumber int,
	itemIndex int,
) ([]entity.ItineraryItemReaction, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryItemReactionColumns).
		From("itinerary_item_reactions").
		Where(sq.Eq{
			"trip_id":    dto.IDArg(tripID),
			"day_number": dayNumber,
			"item_index": itemIndex,
		}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list itinerary item reactions by item: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query itinerary item reactions by item: %w", err)
	}
	defer rows.Close()
	return dto.ScanItineraryItemReactionRows(rows)
}

func (r *Repository) UpsertDiscoverySuggestionVote(
	ctx context.Context,
	vote *entity.DiscoverySuggestionVote,
) (*entity.DiscoverySuggestionVote, error) {
	metadata, err := dto.JSONBArg(vote.Metadata)
	if err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Insert("trip_discovery_suggestion_votes").
		Columns("id", "session_id", "suggestion_id", "trip_id", "user_id", "vote", "metadata").
		Values(
			dto.IDArg(vote.ID),
			dto.IDArg(vote.SessionID),
			vote.SuggestionID,
			dto.UUIDNullableArg(vote.TripID),
			dto.IDArg(vote.UserID),
			string(vote.Vote),
			metadata,
		).
		Suffix(
			"ON CONFLICT (session_id, suggestion_id, user_id) DO UPDATE SET " +
				"trip_id = EXCLUDED.trip_id, vote = EXCLUDED.vote, metadata = EXCLUDED.metadata, updated_at = NOW() " +
				"RETURNING " + dto.DiscoverySuggestionVoteColumns,
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert discovery suggestion vote: %w", err)
	}
	return dto.ScanDiscoverySuggestionVote(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListDiscoverySuggestionVotesBySession(
	ctx context.Context,
	sessionID uuid.UUID,
) ([]entity.DiscoverySuggestionVote, error) {
	query, args, err := r.db.Builder.
		Select(dto.DiscoverySuggestionVoteColumns).
		From("trip_discovery_suggestion_votes").
		Where(sq.Eq{"session_id": dto.IDArg(sessionID)}).
		OrderBy("suggestion_id ASC", "created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list discovery suggestion votes by session: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query discovery suggestion votes by session: %w", err)
	}
	defer rows.Close()
	return dto.ScanDiscoverySuggestionVoteRows(rows)
}

func (r *Repository) ListDiscoverySuggestionVotesByTrip(
	ctx context.Context,
	tripID uuid.UUID,
) ([]entity.DiscoverySuggestionVote, error) {
	query, args, err := r.db.Builder.
		Select(dto.DiscoverySuggestionVoteColumns).
		From("trip_discovery_suggestion_votes").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("suggestion_id ASC", "created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list discovery suggestion votes by trip: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query discovery suggestion votes by trip: %w", err)
	}
	defer rows.Close()
	return dto.ScanDiscoverySuggestionVoteRows(rows)
}
