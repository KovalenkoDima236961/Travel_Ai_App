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
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

// Repository persists trips using squirrel query building over the shared
// postgres pool.
type Repository struct {
	db *storage.DB
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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
	return r.updateItineraryByUserID(ctx, r.db, id, userID, itinerary, status)
}

// UpdateItineraryByUserIDAndCreateVersion stores a current itinerary snapshot
// and its version-history record atomically. If version creation fails, the
// itinerary update is rolled back so history cannot silently fall behind.
func (r *Repository) UpdateItineraryByUserIDAndCreateVersion(
	ctx context.Context,
	id, userID uuid.UUID,
	itinerary json.RawMessage,
	status entity.Status,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
) (*entity.Trip, *entity.ItineraryVersion, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin itinerary version tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	updated, err := r.updateItineraryByUserID(ctx, tx, id, userID, itinerary, status)
	if err != nil {
		return nil, nil, err
	}

	versionNumber, err := r.nextItineraryVersionNumber(ctx, tx, id)
	if err != nil {
		return nil, nil, err
	}

	version, err := r.createItineraryVersion(ctx, tx, &entity.ItineraryVersion{
		ID:            uuid.New(),
		TripID:        id,
		UserID:        userID,
		VersionNumber: versionNumber,
		Source:        source,
		Itinerary:     itinerary,
		Metadata:      metadata,
	})
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit itinerary version tx: %w", err)
	}
	committed = true

	return updated, version, nil
}

// CreateItineraryVersion inserts a snapshot row. It is exposed for focused
// repository tests and maintenance callers; normal itinerary mutations should
// use UpdateItineraryByUserIDAndCreateVersion.
func (r *Repository) CreateItineraryVersion(ctx context.Context, v *entity.ItineraryVersion) (*entity.ItineraryVersion, error) {
	return r.createItineraryVersion(ctx, r.db, v)
}

// GetNextItineraryVersionNumber returns the next per-trip version number.
func (r *Repository) GetNextItineraryVersionNumber(ctx context.Context, tripID uuid.UUID) (int, error) {
	return r.nextItineraryVersionNumber(ctx, r.db, tripID)
}

// ListItineraryVersionsByTripAndUser returns newest snapshots first, without
// selecting the large itinerary JSON payload.
func (r *Repository) ListItineraryVersionsByTripAndUser(
	ctx context.Context,
	tripID, userID uuid.UUID,
	limit, offset int,
) ([]entity.ItineraryVersion, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryVersionSummaryColumns).
		From("itinerary_versions").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		OrderBy("version_number DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list itinerary versions: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query itinerary versions: %w", err)
	}
	defer rows.Close()

	versions := make([]entity.ItineraryVersion, 0)
	for rows.Next() {
		version, err := dto.ScanItineraryVersionSummary(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, *version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate itinerary versions: %w", err)
	}

	return versions, nil
}

// GetItineraryVersionByIDTripAndUser loads one full snapshot only when it
// belongs to both the requested trip and authenticated user.
func (r *Repository) GetItineraryVersionByIDTripAndUser(
	ctx context.Context,
	id, tripID, userID uuid.UUID,
) (*entity.ItineraryVersion, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryVersionColumns).
		From("itinerary_versions").
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get itinerary version: %w", err)
	}

	return dto.ScanItineraryVersion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) updateItineraryByUserID(
	ctx context.Context,
	q rowQuerier,
	id, userID uuid.UUID,
	itinerary json.RawMessage,
	status entity.Status,
) (*entity.Trip, error) {
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

	return dto.Scan(q.QueryRow(ctx, query, args...))
}

func (r *Repository) nextItineraryVersionNumber(ctx context.Context, q rowQuerier, tripID uuid.UUID) (int, error) {
	query, args, err := r.db.Builder.
		Select("COALESCE(MAX(version_number), 0) + 1").
		From("itinerary_versions").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build next itinerary version number: %w", err)
	}

	var versionNumber int
	if err := q.QueryRow(ctx, query, args...).Scan(&versionNumber); err != nil {
		return 0, fmt.Errorf("scan next itinerary version number: %w", err)
	}
	return versionNumber, nil
}

func (r *Repository) createItineraryVersion(
	ctx context.Context,
	q rowQuerier,
	v *entity.ItineraryVersion,
) (*entity.ItineraryVersion, error) {
	values, err := dto.ItineraryVersionInsertValues(v)
	if err != nil {
		return nil, err
	}

	query, args, err := r.db.Builder.
		Insert("itinerary_versions").
		Columns(dto.ItineraryVersionInsertColumns()...).
		Values(values...).
		Suffix("RETURNING " + dto.ItineraryVersionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert itinerary version: %w", err)
	}

	return dto.ScanItineraryVersion(q.QueryRow(ctx, query, args...))
}
