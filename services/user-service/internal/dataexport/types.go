package dataexport

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TypeAccount = "account"
	Queued      = "queued"
	Completed   = "completed"
	Failed      = "failed"
	Expired     = "expired"
)

type Job struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	ExportType       string
	Status           string
	Scope            json.RawMessage
	FilePath         *string
	FileName         *string
	MIMEType         *string
	SizeBytes        *int64
	ChecksumSHA256   *string
	ErrorCode        *string
	ErrorMessageSafe *string
	ExpiresAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Config struct {
	Enabled         bool
	StorageDir      string
	TTL             time.Duration
	MaxAccountBytes int64
}

func DefaultConfig() Config {
	return Config{Enabled: true, StorageDir: "./data/exports", TTL: 24 * time.Hour, MaxAccountBytes: 250 * 1024 * 1024}
}
