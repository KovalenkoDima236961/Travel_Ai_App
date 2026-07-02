package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type BudgetOptimizationScope string

const (
	BudgetOptimizationScopeDay BudgetOptimizationScope = "day"
)

type BudgetOptimizationProposalStatus string

const (
	BudgetOptimizationProposalStatusPending   BudgetOptimizationProposalStatus = "pending"
	BudgetOptimizationProposalStatusApplied   BudgetOptimizationProposalStatus = "applied"
	BudgetOptimizationProposalStatusDiscarded BudgetOptimizationProposalStatus = "discarded"
	BudgetOptimizationProposalStatusExpired   BudgetOptimizationProposalStatus = "expired"
	BudgetOptimizationProposalStatusFailed    BudgetOptimizationProposalStatus = "failed"
)

type BudgetOptimizationProposal struct {
	ID                        uuid.UUID
	TripID                    uuid.UUID
	JobID                     *uuid.UUID
	CreatedByUserID           uuid.UUID
	Scope                     BudgetOptimizationScope
	DayNumber                 *int
	ExpectedItineraryRevision int
	BaseItineraryRevision     int
	Status                    BudgetOptimizationProposalStatus
	Currency                  string
	TargetReductionAmount     *float64
	EstimatedSavingsAmount    *float64
	ProposalJSON              json.RawMessage
	AppliedItineraryRevision  *int
	CreatedAt                 time.Time
	AppliedAt                 *time.Time
	DiscardedAt               *time.Time
	ExpiredAt                 *time.Time
	UpdatedAt                 time.Time
}
