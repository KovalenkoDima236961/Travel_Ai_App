package entity

import (
	"time"

	"github.com/google/uuid"
)

type NotificationDigestBatch struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Channel          string
	Mode             string
	Status           string
	ScheduledFor     time.Time
	SentAt           *time.Time
	Attempts         int
	NextAttemptAt    *time.Time
	ErrorCode        *string
	ErrorMessageSafe *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Items            []NotificationDigestItem
}

type NotificationDigestItem struct {
	ID             uuid.UUID
	BatchID        uuid.UUID
	NotificationID *uuid.UUID
	UserID         uuid.UUID
	TripID         *uuid.UUID
	Category       string
	Priority       string
	DigestKey      string
	Title          string
	Message        string
	Metadata       map[string]any
	EventCount     int
	LatestEventAt  time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
