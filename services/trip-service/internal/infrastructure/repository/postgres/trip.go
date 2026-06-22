// Package postgres is the PostgreSQL adapter for the trip repository port. It
// builds queries with squirrel and delegates row<->entity mapping to its dto
// subpackage.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

// Repository persists trips using squirrel query building over the shared
// postgres pool.
type Repository struct {
	db *storage.DB
}

// New constructs the trip repository.
func New(db *storage.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new trip and returns the stored row.
func (r *Repository) Create(ctx context.Context, t *entity.Trip) (*entity.Trip, error) {
	values, err := dto.InsertValues(t)
	if err != nil {
		return nil, err
	}

	query, args, err := r.db.Builder.
		Insert("trips").
		Columns(dto.InsertColumns()...).
		Values(values...).
		Suffix("RETURNING " + dto.Columns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert: %w", err)
	}

	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}

// GetByIDAndUserID loads a trip only when it belongs to the given user.
func (r *Repository) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*entity.Trip, error) {
	query, args, err := r.db.Builder.
		Select(dto.Columns).
		From("trips").
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"user_id": dto.IDArg(userID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select: %w", err)
	}

	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}

// ListByUser returns one user's trips ordered by created_at DESC.
// Callers are expected to pass already-validated, normalised parameters.
func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Trip, error) {
	query, args, err := r.db.Builder.
		Select(dto.Columns).
		From("trips").
		Where(sq.Eq{"user_id": dto.IDArg(userID)}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trips: %w", err)
	}
	defer rows.Close()

	trips := make([]entity.Trip, 0)
	for rows.Next() {
		t, err := dto.Scan(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trips: %w", err)
	}

	return trips, nil
}

// UpdateStatusByUserID transitions a trip only when it belongs to the user.
func (r *Repository) UpdateStatusByUserID(ctx context.Context, id, userID uuid.UUID, status entity.Status) (*entity.Trip, error) {
	query, args, err := r.db.Builder.
		Update("trips").
		Set("status", string(status)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"user_id": dto.IDArg(userID),
		}).
		Suffix("RETURNING " + dto.Columns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update status: %w", err)
	}

	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}

// UpdateItineraryByUserID stores the itinerary only when the trip belongs to the user.
func (r *Repository) UpdateItineraryByUserID(ctx context.Context, id, userID uuid.UUID, itinerary json.RawMessage, status entity.Status) (*entity.Trip, error) {
	query, args, err := r.db.Builder.
		Update("trips").
		Set("itinerary", []byte(itinerary)).
		Set("status", string(status)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"user_id": dto.IDArg(userID),
		}).
		Suffix("RETURNING " + dto.Columns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update itinerary: %w", err)
	}

	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}
