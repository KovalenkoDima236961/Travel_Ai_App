// Package aiobservability persists privacy-safe, ops-only AI generation traces.
package aiobservability

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	StatusStarted               = "started"
	StatusCompleted             = "completed"
	StatusCompletedWithWarnings = "completed_with_warnings"
	StatusFailed                = "failed"
	StatusCancelled             = "cancelled"
	StatusBlocked               = "blocked"
)

type Config struct {
	Enabled                   bool
	TraceEventsEnabled        bool
	StoreRedactedPrompts      bool
	StoreRedactedResponses    bool
	MaxPromptSnapshotChars    int
	RetentionDays             int
	FailOpen                  bool
	DebugLocalOnly            bool
	PromptLoggingEnabled      bool
	PromptLoggingRedactedOnly bool
	RedactionEnabled          bool
	Provider                  string
	Model                     string
	Mode                      string
}

func NormalizeConfig(cfg Config) Config {
	if cfg.MaxPromptSnapshotChars <= 0 {
		cfg.MaxPromptSnapshotChars = 12000
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}
	if cfg.Provider == "" {
		cfg.Provider = "other"
	}
	if cfg.Mode == "" {
		cfg.Mode = "other"
	}
	return cfg
}

type Trace struct {
	ID                      uuid.UUID       `json:"id"`
	TripID                  *uuid.UUID      `json:"tripId,omitempty"`
	JobID                   *uuid.UUID      `json:"jobId,omitempty"`
	UserID                  *uuid.UUID      `json:"userId,omitempty"`
	WorkspaceID             *uuid.UUID      `json:"workspaceId,omitempty"`
	RequestID               *string         `json:"requestId,omitempty"`
	CorrelationID           *string         `json:"correlationId,omitempty"`
	GenerationType          string          `json:"generationType"`
	Source                  string          `json:"source"`
	Provider                string          `json:"provider"`
	Model                   *string         `json:"model,omitempty"`
	AIMode                  string          `json:"aiMode"`
	PromptVersion           *string         `json:"promptVersion,omitempty"`
	PlanningContextVersion  *string         `json:"planningContextVersion,omitempty"`
	ValidatorVersion        *string         `json:"validatorVersion,omitempty"`
	Status                  string          `json:"status"`
	QualityStatus           *string         `json:"qualityStatus,omitempty"`
	InputSummary            json.RawMessage `json:"inputSummary,omitempty"`
	ConstraintsSummary      json.RawMessage `json:"constraintsSummary,omitempty"`
	RAGSummary              json.RawMessage `json:"ragSummary,omitempty"`
	PromptSummary           json.RawMessage `json:"promptSummary,omitempty"`
	GenerationSummary       json.RawMessage `json:"generationSummary,omitempty"`
	ValidationSummary       json.RawMessage `json:"validationSummary,omitempty"`
	RepairSummary           json.RawMessage `json:"repairSummary,omitempty"`
	OutputSummary           json.RawMessage `json:"outputSummary,omitempty"`
	ErrorCode               *string         `json:"errorCode,omitempty"`
	ErrorMessageSafe        *string         `json:"errorMessageSafe,omitempty"`
	DurationMS              *int            `json:"durationMs,omitempty"`
	QueueWaitMS             *int            `json:"queueWaitMs,omitempty"`
	AICallDurationMS        *int            `json:"aiCallDurationMs,omitempty"`
	ValidationDurationMS    *int            `json:"validationDurationMs,omitempty"`
	RepairDurationMS        *int            `json:"repairDurationMs,omitempty"`
	TokenPromptEstimate     *int            `json:"tokenPromptEstimate,omitempty"`
	TokenCompletionEstimate *int            `json:"tokenCompletionEstimate,omitempty"`
	TokenTotalEstimate      *int            `json:"tokenTotalEstimate,omitempty"`
	CreatedAt               time.Time       `json:"createdAt"`
	StartedAt               *time.Time      `json:"startedAt,omitempty"`
	CompletedAt             *time.Time      `json:"completedAt,omitempty"`
}

type TraceContext struct {
	TraceID        uuid.UUID
	CorrelationID  string
	RequestID      string
	GenerationType string
	StartedAt      time.Time
	Active         bool
}

type StartTraceInput struct {
	TripID, JobID, UserID, WorkspaceID *uuid.UUID
	RequestID, CorrelationID           *string
	GenerationType, Source             string
	Provider, Model, AIMode            string
	PromptVersion                      string
	PlanningContextVersion             string
	ValidatorVersion                   string
	InputSummary, ConstraintsSummary   json.RawMessage
	RAGSummary, PromptSummary          json.RawMessage
	QueueWaitMS                        *int
}

type TraceEventInput struct {
	EventType  string
	Status     string
	Title      string
	Message    string
	Metadata   json.RawMessage
	DurationMS *int
}

type CompleteTraceInput struct {
	Status                  string
	QualityStatus           string
	GenerationSummary       json.RawMessage
	ValidationSummary       json.RawMessage
	RepairSummary           json.RawMessage
	OutputSummary           json.RawMessage
	AICallDurationMS        *int
	ValidationDurationMS    *int
	RepairDurationMS        *int
	TokenPromptEstimate     *int
	TokenCompletionEstimate *int
	TokenTotalEstimate      *int
}

type FailTraceInput struct {
	Status        string
	QualityStatus string
	ErrorCode     string
	ErrorMessage  string
}

type TraceEvent struct {
	ID         uuid.UUID       `json:"id"`
	TraceID    uuid.UUID       `json:"traceId"`
	EventType  string          `json:"eventType"`
	Status     string          `json:"eventStatus"`
	Title      string          `json:"title"`
	Message    *string         `json:"message,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	DurationMS *int            `json:"durationMs,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type PromptSnapshot struct {
	ID              uuid.UUID `json:"id"`
	TraceID         uuid.UUID `json:"traceId"`
	SnapshotType    string    `json:"snapshotType"`
	ContentRedacted string    `json:"contentRedacted"`
	ContentHash     string    `json:"contentHash"`
	TokenEstimate   *int      `json:"tokenEstimate,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type ListFilters struct {
	Status, GenerationType, Provider, Model, QualityStatus string
	TripID, JobID, UserID, WorkspaceID                     *uuid.UUID
	From, To                                               *time.Time
	Limit                                                  int
	Cursor                                                 string
	ErrorOnly                                              bool
}

type ListResult struct {
	Items      []Trace `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

type Detail struct {
	Trace          Trace           `json:"trace"`
	Events         []TraceEvent    `json:"events"`
	PromptSnapshot *PromptSnapshot `json:"promptSnapshot,omitempty"`
}
