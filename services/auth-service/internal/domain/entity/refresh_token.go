package entity

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken is the stored, hashed representation of an issued refresh token.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
