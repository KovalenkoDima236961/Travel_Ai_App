package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

const BudgetOptimizationProposalColumns = "id, trip_id, job_id, created_by_user_id, " +
	"scope, day_number, expected_itinerary_revision, base_itinerary_revision, status, " +
	"currency, target_reduction_amount, estimated_savings_amount, proposal_json, " +
	"applied_itinerary_revision, created_at, applied_at, discarded_at, expired_at, updated_at"

func BudgetOptimizationProposalInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"job_id",
		"created_by_user_id",
		"scope",
		"day_number",
		"expected_itinerary_revision",
		"base_itinerary_revision",
		"status",
		"currency",
		"target_reduction_amount",
		"estimated_savings_amount",
		"proposal_json",
	}
}

func BudgetOptimizationProposalInsertValues(p *entity.BudgetOptimizationProposal) []any {
	return []any{
		toPgUUID(p.ID),
		toPgUUID(p.TripID),
		toPgUUIDPtr(p.JobID),
		toPgUUID(p.CreatedByUserID),
		string(p.Scope),
		toPgIntPtr(p.DayNumber),
		p.ExpectedItineraryRevision,
		p.BaseItineraryRevision,
		string(p.Status),
		p.Currency,
		NumericArg(p.TargetReductionAmount),
		NumericArg(p.EstimatedSavingsAmount),
		p.ProposalJSON,
	}
}

func ScanBudgetOptimizationProposal(row pgx.Row) (*entity.BudgetOptimizationProposal, error) {
	var (
		id, tripID, jobID, createdByUserID pgtype.UUID
		scope, status, currency            string
		dayNumber                          pgtype.Int4
		expectedRevision                   int
		baseRevision                       int
		targetReduction                    pgtype.Numeric
		estimatedSavings                   pgtype.Numeric
		proposalRaw                        []byte
		appliedRevision                    pgtype.Int4
		createdAt                          pgtype.Timestamp
		appliedAt                          pgtype.Timestamp
		discardedAt                        pgtype.Timestamp
		expiredAt                          pgtype.Timestamp
		updatedAt                          pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&jobID,
		&createdByUserID,
		&scope,
		&dayNumber,
		&expectedRevision,
		&baseRevision,
		&status,
		&currency,
		&targetReduction,
		&estimatedSavings,
		&proposalRaw,
		&appliedRevision,
		&createdAt,
		&appliedAt,
		&discardedAt,
		&expiredAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan budget optimization proposal: %w", err)
	}

	return &entity.BudgetOptimizationProposal{
		ID:                        uuid.UUID(id.Bytes),
		TripID:                    uuid.UUID(tripID.Bytes),
		JobID:                     fromPgUUID(jobID),
		CreatedByUserID:           uuid.UUID(createdByUserID.Bytes),
		Scope:                     entity.BudgetOptimizationScope(scope),
		DayNumber:                 fromPgIntPtr(dayNumber),
		ExpectedItineraryRevision: expectedRevision,
		BaseItineraryRevision:     baseRevision,
		Status:                    entity.BudgetOptimizationProposalStatus(status),
		Currency:                  currency,
		TargetReductionAmount:     fromPgNumeric(targetReduction),
		EstimatedSavingsAmount:    fromPgNumeric(estimatedSavings),
		ProposalJSON:              proposalRaw,
		AppliedItineraryRevision:  fromPgIntPtr(appliedRevision),
		CreatedAt:                 createdAt.Time,
		AppliedAt:                 fromPgTimestampPtr(appliedAt),
		DiscardedAt:               fromPgTimestampPtr(discardedAt),
		ExpiredAt:                 fromPgTimestampPtr(expiredAt),
		UpdatedAt:                 updatedAt.Time,
	}, nil
}

func ScanBudgetOptimizationProposalRows(rows pgx.Rows) ([]entity.BudgetOptimizationProposal, error) {
	proposals := make([]entity.BudgetOptimizationProposal, 0)
	for rows.Next() {
		proposal, err := ScanBudgetOptimizationProposal(rows)
		if err != nil {
			return nil, err
		}
		proposals = append(proposals, *proposal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate budget optimization proposals: %w", err)
	}
	return proposals, nil
}
