package entity

import (
	"time"

	"github.com/google/uuid"
)

// User is the local account identity used by the auth service.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
