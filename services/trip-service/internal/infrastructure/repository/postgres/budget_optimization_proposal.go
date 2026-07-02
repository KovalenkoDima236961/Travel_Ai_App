package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateBudgetOptimizationProposal(
	ctx context.Context,
	proposal *entity.BudgetOptimizationProposal,
) (*entity.BudgetOptimizationProposal, error) {
	query, args, err := r.db.Builder.
		Insert("budget_optimization_proposals").
		Columns(dto.BudgetOptimizationProposalInsertColumns()...).
		Values(dto.BudgetOptimizationProposalInsertValues(proposal)...).
		Suffix("RETURNING " + dto.BudgetOptimizationProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create budget optimization proposal: %w", err)
	}

	return dto.ScanBudgetOptimizationProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetBudgetOptimizationProposalByID(
	ctx context.Context,
	id uuid.UUID,
) (*entity.BudgetOptimizationProposal, error) {
	query, args, err := r.db.Builder.
		Select(dto.BudgetOptimizationProposalColumns).
		From("budget_optimization_proposals").
		Where(sq.Eq{"id": dto.IDArg(id)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get budget optimization proposal: %w", err)
	}

	return dto.ScanBudgetOptimizationProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetBudgetOptimizationProposalByIDAndTrip(
	ctx context.Context,
	id, tripID uuid.UUID,
) (*entity.BudgetOptimizationProposal, error) {
	query, args, err := r.db.Builder.
		Select(dto.BudgetOptimizationProposalColumns).
		From("budget_optimization_proposals").
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get budget optimization proposal by trip: %w", err)
	}

	return dto.ScanBudgetOptimizationProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListBudgetOptimizationProposalsByTrip(
	ctx context.Context,
	tripID uuid.UUID,
	status *entity.BudgetOptimizationProposalStatus,
	limit int,
) ([]entity.BudgetOptimizationProposal, error) {
	builder := r.db.Builder.
		Select(dto.BudgetOptimizationProposalColumns).
		From("budget_optimization_proposals").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("created_at DESC").
		Limit(uint64(limit))
	if status != nil {
		builder = builder.Where(sq.Eq{"status": string(*status)})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list budget optimization proposals: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query budget optimization proposals: %w", err)
	}
	defer rows.Close()

	return dto.ScanBudgetOptimizationProposalRows(rows)
}

func (r *Repository) ListPendingBudgetOptimizationProposalsByTrip(
	ctx context.Context,
	tripID uuid.UUID,
	limit int,
) ([]entity.BudgetOptimizationProposal, error) {
	status := entity.BudgetOptimizationProposalStatusPending
	return r.ListBudgetOptimizationProposalsByTrip(ctx, tripID, &status, limit)
}

func (r *Repository) MarkBudgetOptimizationProposalApplied(
	ctx context.Context,
	id uuid.UUID,
	appliedItineraryRevision int,
) (*entity.BudgetOptimizationProposal, error) {
	query, args, err := r.db.Builder.
		Update("budget_optimization_proposals").
		Set("status", string(entity.BudgetOptimizationProposalStatusApplied)).
		Set("applied_itinerary_revision", appliedItineraryRevision).
		Set("applied_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.BudgetOptimizationProposalStatusPending),
		}).
		Suffix("RETURNING " + dto.BudgetOptimizationProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark budget optimization proposal applied: %w", err)
	}

	return dto.ScanBudgetOptimizationProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkBudgetOptimizationProposalDiscarded(
	ctx context.Context,
	id uuid.UUID,
) (*entity.BudgetOptimizationProposal, error) {
	query, args, err := r.db.Builder.
		Update("budget_optimization_proposals").
		Set("status", string(entity.BudgetOptimizationProposalStatusDiscarded)).
		Set("discarded_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.BudgetOptimizationProposalStatusPending),
		}).
		Suffix("RETURNING " + dto.BudgetOptimizationProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark budget optimization proposal discarded: %w", err)
	}

	return dto.ScanBudgetOptimizationProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkBudgetOptimizationProposalExpired(
	ctx context.Context,
	id uuid.UUID,
) (*entity.BudgetOptimizationProposal, error) {
	query, args, err := r.db.Builder.
		Update("budget_optimization_proposals").
		Set("status", string(entity.BudgetOptimizationProposalStatusExpired)).
		Set("expired_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.BudgetOptimizationProposalStatusPending),
		}).
		Suffix("RETURNING " + dto.BudgetOptimizationProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark budget optimization proposal expired: %w", err)
	}

	return dto.ScanBudgetOptimizationProposal(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkBudgetOptimizationProposalFailed(
	ctx context.Context,
	id uuid.UUID,
) (*entity.BudgetOptimizationProposal, error) {
	query, args, err := r.db.Builder.
		Update("budget_optimization_proposals").
		Set("status", string(entity.BudgetOptimizationProposalStatusFailed)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Suffix("RETURNING " + dto.BudgetOptimizationProposalColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark budget optimization proposal failed: %w", err)
	}

	return dto.ScanBudgetOptimizationProposal(r.db.QueryRow(ctx, query, args...))
}
