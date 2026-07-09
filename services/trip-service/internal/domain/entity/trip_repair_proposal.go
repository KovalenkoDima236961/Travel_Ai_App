package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TripRepairProposalStatus string

const (
	TripRepairProposalStatusPending   TripRepairProposalStatus = "pending"
	TripRepairProposalStatusApplied   TripRepairProposalStatus = "applied"
	TripRepairProposalStatusDiscarded TripRepairProposalStatus = "discarded"
	TripRepairProposalStatusExpired   TripRepairProposalStatus = "expired"
	TripRepairProposalStatusFailed    TripRepairProposalStatus = "failed"
)

type TripRepairProposal struct {
	ID                    uuid.UUID
	TripID                uuid.UUID
	JobID                 *uuid.UUID
	CreatedByUserID       uuid.UUID
	Status                TripRepairProposalStatus
	RepairMode            string
	BaseItineraryRevision int
	BaseRiskScore         *int
	ProposedRiskScore     *int
	BasePolicyStatus      *string
	ProposedPolicyStatus  *string
	IssuesJSON            json.RawMessage
	ProposalJSON          json.RawMessage
	CreatedAt             time.Time
	UpdatedAt             time.Time
	AppliedAt             *time.Time
	AppliedByUserID       *uuid.UUID
	DiscardedAt           *time.Time
	DiscardedByUserID     *uuid.UUID
	ExpiredAt             *time.Time
}
