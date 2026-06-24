// Package postgres is the PostgreSQL adapter for the trip repository port. It
// builds queries with squirrel and delegates row<->entity mapping to its dto
// subpackage.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
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

// GetByID loads a trip without owner scoping. It is used only after an enabled
// public share token has already been validated.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Trip, error) {
	query, args, err := r.db.Builder.
		Select(dto.Columns).
		From("trips").
		Where(sq.Eq{"id": dto.IDArg(id)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select by id: %w", err)
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
	return r.UpdateItineraryAndCreateVersion(ctx, id, userID, userID, itinerary, status, source, metadata)
}

// UpdateItineraryAndCreateVersion stores a current itinerary snapshot and
// records both the owning user (user_id) and the actor who caused the change
// (created_by_user_id). Collaborator permissions must be checked before
// calling this method.
func (r *Repository) UpdateItineraryAndCreateVersion(
	ctx context.Context,
	id, ownerUserID, actorUserID uuid.UUID,
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

	updated, err := r.updateItineraryByUserID(ctx, tx, id, ownerUserID, itinerary, status)
	if err != nil {
		return nil, nil, err
	}

	versionNumber, err := r.nextItineraryVersionNumber(ctx, tx, id)
	if err != nil {
		return nil, nil, err
	}

	version, err := r.createItineraryVersion(ctx, tx, &entity.ItineraryVersion{
		ID:              uuid.New(),
		TripID:          id,
		UserID:          ownerUserID,
		CreatedByUserID: &actorUserID,
		VersionNumber:   versionNumber,
		Source:          source,
		Itinerary:       itinerary,
		Metadata:        metadata,
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
	return r.ListItineraryVersionsByTrip(ctx, tripID, limit, offset)
}

func (r *Repository) ListItineraryVersionsByTrip(
	ctx context.Context,
	tripID uuid.UUID,
	limit, offset int,
) ([]entity.ItineraryVersion, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryVersionSummaryColumns).
		From("itinerary_versions").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
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
	return r.GetItineraryVersionByIDTrip(ctx, id, tripID)
}

func (r *Repository) GetItineraryVersionByIDTrip(
	ctx context.Context,
	id, tripID uuid.UUID,
) (*entity.ItineraryVersion, error) {
	query, args, err := r.db.Builder.
		Select(dto.ItineraryVersionColumns).
		From("itinerary_versions").
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get itinerary version: %w", err)
	}

	return dto.ScanItineraryVersion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpsertTripCollaborator(
	ctx context.Context,
	collaborator *entity.TripCollaborator,
) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Insert("trip_collaborators").
		Columns("id", "trip_id", "user_id", "role", "status", "invited_by_user_id", "accepted_at", "removed_at").
		Values(
			dto.IDArg(collaborator.ID),
			dto.IDArg(collaborator.TripID),
			dto.IDArg(collaborator.UserID),
			string(collaborator.Role),
			string(entity.CollaboratorStatusPending),
			dto.IDArg(collaborator.InvitedByUserID),
			nil,
			nil,
		).
		Suffix(
			"ON CONFLICT (trip_id, user_id) DO UPDATE SET " +
				"role = EXCLUDED.role, " +
				"status = CASE WHEN trip_collaborators.status = 'accepted' THEN 'accepted' ELSE 'pending' END, " +
				"invited_by_user_id = EXCLUDED.invited_by_user_id, " +
				"invited_at = CASE WHEN trip_collaborators.status = 'removed' THEN NOW() ELSE trip_collaborators.invited_at END, " +
				"accepted_at = CASE WHEN trip_collaborators.status = 'accepted' THEN trip_collaborators.accepted_at ELSE NULL END, " +
				"removed_at = NULL, " +
				"updated_at = NOW() " +
				"RETURNING " + dto.TripCollaboratorColumns,
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert trip collaborator: %w", err)
	}

	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripCollaboratorByTripAndUser(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripCollaboratorColumns).
		From("trip_collaborators").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip collaborator by user: %w", err)
	}

	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripCollaboratorByID(ctx context.Context, tripID, collaboratorID uuid.UUID) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripCollaboratorColumns).
		From("trip_collaborators").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(collaboratorID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip collaborator by id: %w", err)
	}

	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripCollaborators(ctx context.Context, tripID uuid.UUID) ([]entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripCollaboratorColumns).
		From("trip_collaborators").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		Where(sq.NotEq{"status": string(entity.CollaboratorStatusRemoved)}).
		OrderBy("invited_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip collaborators: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip collaborators: %w", err)
	}
	defer rows.Close()

	return dto.ScanTripCollaboratorRows(rows)
}

func (r *Repository) UpdateTripCollaboratorRole(ctx context.Context, tripID, collaboratorID uuid.UUID, role entity.CollaboratorRole) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Update("trip_collaborators").
		Set("role", string(role)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(collaboratorID),
		}).
		Where(sq.NotEq{"status": string(entity.CollaboratorStatusRemoved)}).
		Suffix("RETURNING " + dto.TripCollaboratorColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip collaborator role: %w", err)
	}

	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) RemoveTripCollaborator(ctx context.Context, tripID, collaboratorID uuid.UUID) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Update("trip_collaborators").
		Set("status", string(entity.CollaboratorStatusRemoved)).
		Set("removed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(collaboratorID),
		}).
		Suffix("RETURNING " + dto.TripCollaboratorColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build remove trip collaborator: %w", err)
	}

	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) AcceptTripCollaborator(ctx context.Context, tripID, collaboratorID, userID uuid.UUID) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Update("trip_collaborators").
		Set("status", string(entity.CollaboratorStatusAccepted)).
		Set("accepted_at", sq.Expr("NOW()")).
		Set("removed_at", nil).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(collaboratorID),
			"user_id": dto.IDArg(userID),
			"status":  string(entity.CollaboratorStatusPending),
		}).
		Suffix("RETURNING " + dto.TripCollaboratorColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build accept trip collaborator: %w", err)
	}

	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) DeclineTripCollaborator(ctx context.Context, tripID, collaboratorID, userID uuid.UUID) (*entity.TripCollaborator, error) {
	query, args, err := r.db.Builder.
		Update("trip_collaborators").
		Set("status", string(entity.CollaboratorStatusRemoved)).
		Set("removed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(collaboratorID),
			"user_id": dto.IDArg(userID),
			"status":  string(entity.CollaboratorStatusPending),
		}).
		Suffix("RETURNING " + dto.TripCollaboratorColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build decline trip collaborator: %w", err)
	}

	return dto.ScanTripCollaborator(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListPendingCollaborationInvitations(ctx context.Context, userID uuid.UUID) ([]entity.SharedTrip, error) {
	return r.listCollaborativeTripsByStatus(ctx, userID, entity.CollaboratorStatusPending)
}

func (r *Repository) ListSharedTripsByUser(ctx context.Context, userID uuid.UUID) ([]entity.SharedTrip, error) {
	return r.listCollaborativeTripsByStatus(ctx, userID, entity.CollaboratorStatusAccepted)
}

func (r *Repository) listCollaborativeTripsByStatus(ctx context.Context, userID uuid.UUID, status entity.CollaboratorStatus) ([]entity.SharedTrip, error) {
	query, args, err := r.db.Builder.
		Select(
			dto.TripColumnsWithAlias,
			dto.TripCollaboratorColumnsWithAlias,
		).
		From("trip_collaborators c").
		Join("trips t ON t.id = c.trip_id").
		Where(sq.Eq{
			"c.user_id": dto.IDArg(userID),
			"c.status":  string(status),
		}).
		OrderBy("c.updated_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list collaborative trips: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query collaborative trips: %w", err)
	}
	defer rows.Close()

	shared := make([]entity.SharedTrip, 0)
	for rows.Next() {
		row, err := dto.ScanSharedTrip(rows)
		if err != nil {
			return nil, err
		}
		shared = append(shared, *row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate collaborative trips: %w", err)
	}

	return shared, nil
}

// CreateTripShare inserts a new public share row for a trip.
func (r *Repository) CreateTripShare(ctx context.Context, share *entity.TripShare) (*entity.TripShare, error) {
	query, args, err := r.db.Builder.
		Insert("trip_shares").
		Columns(dto.TripShareInsertColumns()...).
		Values(dto.TripShareInsertValues(share)...).
		Suffix("RETURNING " + dto.TripShareColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert trip share: %w", err)
	}

	created, err := dto.ScanTripShare(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, mapTripShareWriteError(err)
	}
	return created, nil
}

// GetTripShareByTripAndUser returns one owned share row.
func (r *Repository) GetTripShareByTripAndUser(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripShareColumns).
		From("trip_shares").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip share by trip: %w", err)
	}

	return dto.ScanTripShare(r.db.QueryRow(ctx, query, args...))
}

// GetTripShareByToken returns a share row by its opaque public token.
func (r *Repository) GetTripShareByToken(ctx context.Context, shareToken string) (*entity.TripShare, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripShareColumns).
		From("trip_shares").
		Where(sq.Eq{"share_token": shareToken}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip share by token: %w", err)
	}

	return dto.ScanTripShare(r.db.QueryRow(ctx, query, args...))
}

// EnableTripShare re-enables an existing share row without rotating its token.
func (r *Repository) EnableTripShare(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error) {
	query, args, err := r.db.Builder.
		Update("trip_shares").
		Set("enabled", true).
		Set("disabled_at", nil).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		Suffix("RETURNING " + dto.TripShareColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build enable trip share: %w", err)
	}

	return dto.ScanTripShare(r.db.QueryRow(ctx, query, args...))
}

// UpdateTripShareSettings updates owner-controlled share settings without
// rotating the existing token.
func (r *Repository) UpdateTripShareSettings(
	ctx context.Context,
	tripID, userID uuid.UUID,
	expiresAt *time.Time,
	passwordRequired bool,
	passwordHash *string,
) (*entity.TripShare, error) {
	query, args, err := r.db.Builder.
		Update("trip_shares").
		Set("expires_at", expiresAt).
		Set("password_required", passwordRequired).
		Set("password_hash", passwordHash).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		Suffix("RETURNING " + dto.TripShareColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip share settings: %w", err)
	}

	return dto.ScanTripShare(r.db.QueryRow(ctx, query, args...))
}

// DisableTripShare disables an existing share row for an owned trip.
func (r *Repository) DisableTripShare(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error) {
	query, args, err := r.db.Builder.
		Update("trip_shares").
		Set("enabled", false).
		Set("disabled_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"user_id": dto.IDArg(userID),
		}).
		Suffix("RETURNING " + dto.TripShareColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build disable trip share: %w", err)
	}

	return dto.ScanTripShare(r.db.QueryRow(ctx, query, args...))
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

func mapTripShareWriteError(err error) error {
	if storage.UniqueConstraintViolation(err) {
		return fmt.Errorf("trip share conflict: %w", domainerrs.ErrConflict)
	}
	return err
}
