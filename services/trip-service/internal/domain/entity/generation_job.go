package entity

import (
	"time"

	"github.com/google/uuid"
)

type GenerationJobType string

const (
	GenerationJobTypeFullGeneration         GenerationJobType = "full_generation"
	GenerationJobTypeDayRegeneration        GenerationJobType = "day_regeneration"
	GenerationJobTypeItemRegeneration       GenerationJobType = "item_regeneration"
	GenerationJobTypeQualityImprovementDay  GenerationJobType = "quality_improvement_day"
	GenerationJobTypeQualityImprovementItem GenerationJobType = "quality_improvement_item"
)

type GenerationJobStatus string

const (
	GenerationJobStatusQueued    GenerationJobStatus = "queued"
	GenerationJobStatusRunning   GenerationJobStatus = "running"
	GenerationJobStatusCompleted GenerationJobStatus = "completed"
	GenerationJobStatusFailed    GenerationJobStatus = "failed"
	GenerationJobStatusCancelled GenerationJobStatus = "cancelled"
)

type GenerationJob struct {
	ID                        uuid.UUID
	TripID                    uuid.UUID
	RequestedByUserID         uuid.UUID
	JobType                   GenerationJobType
	Status                    GenerationJobStatus
	ExpectedItineraryRevision int
	Instruction               *string
	DayNumber                 *int
	ItemIndex                 *int
	ErrorCode                 *string
	ErrorMessage              *string
	ResultItineraryRevision   *int
	CreatedAt                 time.Time
	StartedAt                 *time.Time
	CompletedAt               *time.Time
	CancelledAt               *time.Time
	UpdatedAt                 time.Time
}
