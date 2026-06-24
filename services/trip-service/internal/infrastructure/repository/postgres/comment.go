package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

// CreateItineraryComment inserts a new active comment and returns the stored row.
func (r *Repository) CreateItineraryComment(ctx context.Context, comment *entity.ItineraryComment) (*entity.ItineraryComment, error) {
	query, args, err := r.db.Builder.
		Insert("itinerary_comments").
		Columns("id", "trip_id", "day_number", "item_index", "author_user_id", "body", "status").
		Values(
			dto.IDArg(comment.ID),
			dto.IDArg(comment.TripID),
			comment.DayNumber,
			comment.ItemIndex,
			dto.IDArg(comment.AuthorUserID),
			comment.Body,
			string(entity.CommentStatusActive),
		).
		Suffix("RETURNING " + dto.ItineraryCommentColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert itinerary comment: %w", err)
	}

	return dto.ScanItineraryComment(r.db.QueryRow(ctx, query, args...))
}

// ListItineraryCommentsByTrip returns all active comments for a trip ordered by
// day, then item, then creation time.
func (r *Repository) ListItineraryCommentsByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.ItineraryComment, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryCommentColumns).
		From("itinerary_comments").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.CommentStatusActive),
		}).
		OrderBy("day_number ASC", "item_index ASC", "created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list itinerary comments by trip: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query itinerary comments by trip: %w", err)
	}
	defer rows.Close()

	return dto.ScanItineraryCommentRows(rows)
}

// ListItineraryCommentsByItem returns active comments for one itinerary item
// ordered by creation time.
func (r *Repository) ListItineraryCommentsByItem(ctx context.Context, tripID uuid.UUID, dayNumber, itemIndex int) ([]entity.ItineraryComment, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryCommentColumns).
		From("itinerary_comments").
		Where(sq.Eq{
			"trip_id":    dto.IDArg(tripID),
			"day_number": dayNumber,
			"item_index": itemIndex,
			"status":     string(entity.CommentStatusActive),
		}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list itinerary comments by item: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query itinerary comments by item: %w", err)
	}
	defer rows.Close()

	return dto.ScanItineraryCommentRows(rows)
}

// GetItineraryCommentByID loads a single comment scoped to its trip. Scoping by
// trip_id prevents acting on a comment through a different trip's path.
func (r *Repository) GetItineraryCommentByID(ctx context.Context, tripID, commentID uuid.UUID) (*entity.ItineraryComment, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryCommentColumns).
		From("itinerary_comments").
		Where(sq.Eq{
			"id":      dto.IDArg(commentID),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get itinerary comment by id: %w", err)
	}

	return dto.ScanItineraryComment(r.db.QueryRow(ctx, query, args...))
}

// UpdateItineraryCommentBody updates an active comment's body and returns the
// updated row. Deleted comments are not updated.
func (r *Repository) UpdateItineraryCommentBody(ctx context.Context, tripID, commentID uuid.UUID, body string) (*entity.ItineraryComment, error) {
	query, args, err := r.db.Builder.
		Update("itinerary_comments").
		Set("body", body).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(commentID),
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.CommentStatusActive),
		}).
		Suffix("RETURNING " + dto.ItineraryCommentColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update itinerary comment body: %w", err)
	}

	return dto.ScanItineraryComment(r.db.QueryRow(ctx, query, args...))
}

// SoftDeleteItineraryComment marks an active comment as deleted and returns the
// updated row. The body is intentionally kept for audit simplicity; deleted
// comments are excluded from normal list/count queries.
func (r *Repository) SoftDeleteItineraryComment(ctx context.Context, tripID, commentID uuid.UUID) (*entity.ItineraryComment, error) {
	query, args, err := r.db.Builder.
		Update("itinerary_comments").
		Set("status", string(entity.CommentStatusDeleted)).
		Set("deleted_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(commentID),
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.CommentStatusActive),
		}).
		Suffix("RETURNING " + dto.ItineraryCommentColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build soft delete itinerary comment: %w", err)
	}

	return dto.ScanItineraryComment(r.db.QueryRow(ctx, query, args...))
}

// CountItineraryCommentsByTripGrouped returns active comment counts grouped by
// itinerary item for a trip.
func (r *Repository) CountItineraryCommentsByTripGrouped(ctx context.Context, tripID uuid.UUID) ([]entity.ItineraryCommentCount, error) {
	query, args, err := r.db.Builder.
		Select("day_number", "item_index", "COUNT(*)").
		From("itinerary_comments").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.CommentStatusActive),
		}).
		GroupBy("day_number", "item_index").
		OrderBy("day_number ASC", "item_index ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count itinerary comments grouped: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query itinerary comment counts: %w", err)
	}
	defer rows.Close()

	return dto.ScanItineraryCommentCounts(rows)
}
