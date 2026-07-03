package entity

import (
	"time"

	"github.com/google/uuid"
)

type WorkspaceBudgetStatus string

const (
	WorkspaceBudgetStatusActive   WorkspaceBudgetStatus = "active"
	WorkspaceBudgetStatusArchived WorkspaceBudgetStatus = "archived"
)

type WorkspaceBudget struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	Name             string
	Description      *string
	Amount           float64
	Currency         string
	PeriodStart      *time.Time
	PeriodEnd        *time.Time
	Status           WorkspaceBudgetStatus
	IsPrimary        bool
	CreatedByUserID  uuid.UUID
	ArchivedByUserID *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ArchivedAt       *time.Time
}
