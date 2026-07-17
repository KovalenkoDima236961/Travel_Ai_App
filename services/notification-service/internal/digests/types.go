package digests

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSent       = "sent"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

type Config struct {
	PublicWebBaseURL string
	MaxAttempts      int
	RetryDelay       time.Duration
}

type QueueInput struct {
	Notification entity.Notification
	Channel      string
	Mode         string
	ScheduledFor time.Time
}

type ProcessInput struct {
	Now   time.Time
	Limit int
}
type ProcessResult struct {
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
	Retrying  int `json:"retrying"`
}

type ListInput struct {
	UserID uuid.UUID
	Status string
	Limit  int
}
