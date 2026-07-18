package dataexport

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TypeTripArchive     = "trip_archive"
	TypeTripExpensesCSV = "trip_expenses_csv"
	TypeTripBudgetCSV   = "trip_budget_csv"
	TypeTripRecap       = "trip_recap"
	StatusQueued        = "queued"
	StatusRunning       = "running"
	StatusCompleted     = "completed"
	StatusFailed        = "failed"
	StatusExpired       = "expired"
	StatusCancelled     = "cancelled"
)

type Job struct {
	ID               uuid.UUID       `json:"exportId"`
	UserID           uuid.UUID       `json:"-"`
	ExportType       string          `json:"exportType"`
	Status           string          `json:"status"`
	Scope            json.RawMessage `json:"-"`
	FilePath         *string         `json:"-"`
	FileName         *string         `json:"fileName,omitempty"`
	MIMEType         *string         `json:"mimeType,omitempty"`
	SizeBytes        *int64          `json:"sizeBytes,omitempty"`
	ChecksumSHA256   *string         `json:"checksumSha256,omitempty"`
	ErrorCode        *string         `json:"errorCode,omitempty"`
	ErrorMessageSafe *string         `json:"errorMessageSafe,omitempty"`
	ExpiresAt        *time.Time      `json:"expiresAt,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	StartedAt        *time.Time      `json:"startedAt,omitempty"`
	CompletedAt      *time.Time      `json:"completedAt,omitempty"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

type Config struct {
	Enabled                      bool
	StorageDir                   string
	TTL                          time.Duration
	MaxTripBytes                 int64
	IncludeReceiptFilesByDefault bool
}

func DefaultConfig() Config {
	return Config{
		Enabled:      true,
		StorageDir:   "./data/exports",
		TTL:          24 * time.Hour,
		MaxTripBytes: 100 * 1024 * 1024,
	}
}
