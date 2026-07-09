package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const TripRepairProposalColumns = "id, trip_id, job_id, created_by_user_id, status, " +
	"repair_mode, base_itinerary_revision, base_risk_score, proposed_risk_score, " +
	"base_policy_status, proposed_policy_status, issues_json, proposal_json, " +
	"created_at, updated_at, applied_at, applied_by_user_id, discarded_at, " +
	"discarded_by_user_id, expired_at"

func TripRepairProposalInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"job_id",
		"created_by_user_id",
		"status",
		"repair_mode",
		"base_itinerary_revision",
		"base_risk_score",
		"proposed_risk_score",
		"base_policy_status",
		"proposed_policy_status",
		"issues_json",
		"proposal_json",
	}
}

func TripRepairProposalInsertValues(p *entity.TripRepairProposal) []any {
	return []any{
		toPgUUID(p.ID),
		toPgUUID(p.TripID),
		toPgUUIDPtr(p.JobID),
		toPgUUID(p.CreatedByUserID),
		string(p.Status),
		p.RepairMode,
		p.BaseItineraryRevision,
		toPgIntPtr(p.BaseRiskScore),
		toPgIntPtr(p.ProposedRiskScore),
		toPgTextPtr(p.BasePolicyStatus),
		toPgTextPtr(p.ProposedPolicyStatus),
		p.IssuesJSON,
		p.ProposalJSON,
	}
}

func ScanTripRepairProposal(row pgx.Row) (*entity.TripRepairProposal, error) {
	var (
		id, tripID, jobID, createdByUserID pgtype.UUID
		status, repairMode                 string
		baseRevision                       int
		baseRisk, proposedRisk             pgtype.Int4
		basePolicy, proposedPolicy         pgtype.Text
		issuesRaw, proposalRaw             []byte
		createdAt, updatedAt               pgtype.Timestamp
		appliedAt, discardedAt, expiredAt  pgtype.Timestamp
		appliedBy, discardedBy             pgtype.UUID
	)
	err := row.Scan(
		&id,
		&tripID,
		&jobID,
		&createdByUserID,
		&status,
		&repairMode,
		&baseRevision,
		&baseRisk,
		&proposedRisk,
		&basePolicy,
		&proposedPolicy,
		&issuesRaw,
		&proposalRaw,
		&createdAt,
		&updatedAt,
		&appliedAt,
		&appliedBy,
		&discardedAt,
		&discardedBy,
		&expiredAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip repair proposal: %w", err)
	}
	return &entity.TripRepairProposal{
		ID:                    uuid.UUID(id.Bytes),
		TripID:                uuid.UUID(tripID.Bytes),
		JobID:                 fromPgUUID(jobID),
		CreatedByUserID:       uuid.UUID(createdByUserID.Bytes),
		Status:                entity.TripRepairProposalStatus(status),
		RepairMode:            repairMode,
		BaseItineraryRevision: baseRevision,
		BaseRiskScore:         fromPgIntPtr(baseRisk),
		ProposedRiskScore:     fromPgIntPtr(proposedRisk),
		BasePolicyStatus:      fromPgText(basePolicy),
		ProposedPolicyStatus:  fromPgText(proposedPolicy),
		IssuesJSON:            issuesRaw,
		ProposalJSON:          proposalRaw,
		CreatedAt:             createdAt.Time,
		UpdatedAt:             updatedAt.Time,
		AppliedAt:             fromPgTimestampPtr(appliedAt),
		AppliedByUserID:       fromPgUUID(appliedBy),
		DiscardedAt:           fromPgTimestampPtr(discardedAt),
		DiscardedByUserID:     fromPgUUID(discardedBy),
		ExpiredAt:             fromPgTimestampPtr(expiredAt),
	}, nil
}

func ScanTripRepairProposalRows(rows pgx.Rows) ([]entity.TripRepairProposal, error) {
	proposals := make([]entity.TripRepairProposal, 0)
	for rows.Next() {
		proposal, err := ScanTripRepairProposal(rows)
		if err != nil {
			return nil, err
		}
		proposals = append(proposals, *proposal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip repair proposals: %w", err)
	}
	return proposals, nil
}
