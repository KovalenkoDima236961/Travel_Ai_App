package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateTripRepairProposal(
	ctx context.Context,
	proposal *entity.TripRepairProposal,
) (*entity.TripRepairProposal, error) {
	query, args, err := r.db.Builder.
		Insert("trip_repair_proposals").
		Columns(dto.TripRepairProposalInsertColumns()...).
		Values(dto.TripRepairProposalInsertValues(proposal)...).
		Suffix("RETURNING " + dto.TripRepairProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip repair proposal: %w", err)
	}
	return dto.ScanTripRepairProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripRepairProposalByIDAndTrip(
	ctx context.Context,
	id, tripID uuid.UUID,
) (*entity.TripRepairProposal, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripRepairProposalColumns).
		From("trip_repair_proposals").
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip repair proposal: %w", err)
	}
	return dto.ScanTripRepairProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetPendingTripRepairProposalByJobID(
	ctx context.Context,
	jobID uuid.UUID,
) (*entity.TripRepairProposal, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripRepairProposalColumns).
		From("trip_repair_proposals").
		Where(sq.Eq{
			"job_id": dto.IDArg(jobID),
			"status": string(entity.TripRepairProposalStatusPending),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get pending trip repair proposal by job: %w", err)
	}
	return dto.ScanTripRepairProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripRepairProposalsByTrip(
	ctx context.Context,
	tripID uuid.UUID,
	status *entity.TripRepairProposalStatus,
	limit int,
) ([]entity.TripRepairProposal, error) {
	builder := r.db.Builder.
		Select(dto.TripRepairProposalColumns).
		From("trip_repair_proposals").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("created_at DESC").
		Limit(uint64(limit))
	if status != nil {
		builder = builder.Where(sq.Eq{"status": string(*status)})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip repair proposals: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip repair proposals: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripRepairProposalRows(rows)
}

func (r *Repository) MarkTripRepairProposalApplied(
	ctx context.Context,
	id, actorUserID uuid.UUID,
) (*entity.TripRepairProposal, error) {
	query, args, err := r.db.Builder.
		Update("trip_repair_proposals").
		Set("status", string(entity.TripRepairProposalStatusApplied)).
		Set("applied_at", sq.Expr("NOW()")).
		Set("applied_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.TripRepairProposalStatusPending),
		}).
		Suffix("RETURNING " + dto.TripRepairProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark trip repair proposal applied: %w", err)
	}
	return dto.ScanTripRepairProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkTripRepairProposalDiscarded(
	ctx context.Context,
	id, actorUserID uuid.UUID,
) (*entity.TripRepairProposal, error) {
	query, args, err := r.db.Builder.
		Update("trip_repair_proposals").
		Set("status", string(entity.TripRepairProposalStatusDiscarded)).
		Set("discarded_at", sq.Expr("NOW()")).
		Set("discarded_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.TripRepairProposalStatusPending),
		}).
		Suffix("RETURNING " + dto.TripRepairProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark trip repair proposal discarded: %w", err)
	}
	return dto.ScanTripRepairProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ExpirePendingTripRepairProposalsForTripRevision(
	ctx context.Context,
	tripID uuid.UUID,
	currentRevision int,
) (int64, error) {
	query, args, err := r.db.Builder.
		Update("trip_repair_proposals").
		Set("status", string(entity.TripRepairProposalStatusExpired)).
		Set("expired_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.TripRepairProposalStatusPending),
		}).
		Where(sq.NotEq{"base_itinerary_revision": currentRevision}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build expire trip repair proposals: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("expire trip repair proposals: %w", err)
	}
	return tag.RowsAffected(), nil
}
