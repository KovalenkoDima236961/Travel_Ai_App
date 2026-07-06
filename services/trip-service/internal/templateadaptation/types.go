// Package templateadaptation holds the shared types for AI template adaptation:
// the job payload persisted on trip_generation_jobs, the internal generator
// request/response, and the adaptation summary stored on the job result.
package templateadaptation

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
)

// Job error codes surfaced to the review UI. They are stable strings shared by
// the worker classifier and the appservice.
const (
	ErrorTemplateNotFound            = "template_not_found"
	ErrorTemplateAccessDenied        = "template_access_denied"
	ErrorTargetWorkspaceAccessDenied = "target_workspace_access_denied"
	ErrorAIAdaptationFailed          = "ai_adaptation_failed"
	ErrorValidationFailed            = "validation_failed"
	ErrorDeterministicFallbackFailed = "deterministic_fallback_failed"
	ErrorProviderEnrichmentFailed    = "provider_enrichment_failed"
)

// Error is a classified template-adaptation failure. The worker maps Code and
// Message directly onto the failed job so the review UI can show a safe reason.
type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func (e *Error) Unwrap() error { return e.Err }

// NewError builds a classified adaptation error.
func NewError(code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// Money is an optional target budget.
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// JobPayload is stored in trip_generation_jobs.payload for a template_adaptation
// job. It carries everything the worker needs to re-load the template, rebuild
// the AI request, and (on failure) fall back to a deterministic copy. It never
// stores the template body itself; the worker re-loads it by TemplateID so
// permissions are re-checked and the latest template content is used.
type JobPayload struct {
	TemplateID              uuid.UUID  `json:"templateId"`
	TemplateTitle           string     `json:"templateTitle"`
	WorkspaceID             *uuid.UUID `json:"workspaceId,omitempty"`
	Title                   string     `json:"title"`
	Destination             string     `json:"destination"`
	StartDate               string     `json:"startDate"`
	DurationDays            int        `json:"durationDays"`
	Budget                  *Money     `json:"budget,omitempty"`
	Travelers               int        `json:"travelers"`
	Pace                    string     `json:"pace"`
	Interests               []string   `json:"interests"`
	Avoid                   []string   `json:"avoid"`
	SpecialInstructions     string     `json:"specialInstructions,omitempty"`
	FallbackToDeterministic bool       `json:"fallbackToDeterministic"`
}

// DecodeJobPayload decodes the stored job payload, tolerating an empty payload.
func DecodeJobPayload(raw json.RawMessage) JobPayload {
	if len(raw) == 0 {
		return JobPayload{}
	}
	var payload JobPayload
	_ = json.Unmarshal(raw, &payload)
	payload.Pace = strings.TrimSpace(payload.Pace)
	if payload.Budget != nil {
		payload.Budget.Currency = strings.ToUpper(strings.TrimSpace(payload.Budget.Currency))
	}
	return payload
}

// Template is the sanitized template structure sent to AI Planning Service. It
// deliberately omits template metadata (source trip IDs, summary, tags) so no
// private data reaches the model prompt.
type Template struct {
	SchemaVersion int           `json:"schemaVersion"`
	DurationDays  int           `json:"durationDays"`
	Days          []TemplateDay `json:"days"`
}

type TemplateDay struct {
	DayOffset int            `json:"dayOffset"`
	Title     string         `json:"title"`
	Items     []TemplateItem `json:"items"`
}

type TemplateItem struct {
	Name          string                   `json:"name"`
	Type          string                   `json:"type"`
	Description   string                   `json:"description,omitempty"`
	Time          string                   `json:"time,omitempty"`
	StartTime     string                   `json:"startTime,omitempty"`
	EndTime       string                   `json:"endTime,omitempty"`
	Place         *TemplatePlace           `json:"place,omitempty"`
	EstimatedCost *aggregate.EstimatedCost `json:"estimatedCost,omitempty"`
	Notes         string                   `json:"notes,omitempty"`
}

type TemplatePlace struct {
	Name     string `json:"name,omitempty"`
	Category string `json:"category,omitempty"`
}

// Target mirrors the AI Planning Service adaptation target.
type Target struct {
	Destination  string   `json:"destination"`
	StartDate    string   `json:"startDate"`
	DurationDays int      `json:"durationDays"`
	Budget       *Money   `json:"budget,omitempty"`
	Travelers    int      `json:"travelers"`
	Pace         string   `json:"pace"`
	Interests    []string `json:"interests"`
	Avoid        []string `json:"avoid"`
}

// Constraints mirrors the AI Planning Service adaptation constraints.
type Constraints struct {
	PreserveStructure       bool   `json:"preserveStructure"`
	AdaptCosts              bool   `json:"adaptCosts"`
	PreserveMealStructure   bool   `json:"preserveMealStructure"`
	PreserveActivityDensity bool   `json:"preserveActivityDensity"`
	SpecialInstructions     string `json:"specialInstructions,omitempty"`
}

// AdaptInput is the internal generator request. Trip Service owns loading
// trusted user context; frontend callers cannot submit these fields.
type AdaptInput struct {
	TripID          uuid.UUID
	Template        Template
	Target          Target
	Constraints     Constraints
	UserProfile     *usercontext.UserProfile
	UserPreferences *usercontext.UserPreferences
}

// AdaptResult is the adapted itinerary plus a reviewable summary.
type AdaptResult struct {
	Itinerary aggregate.Itinerary
	Summary   Summary
}

// Summary is the adaptation summary persisted on the job result and shown in the
// review UI. It is never a guarantee: prices are estimates, availability is
// unchecked, and fallbackUsed is surfaced to the user.
type Summary struct {
	SourceDurationDays int      `json:"sourceDurationDays"`
	TargetDurationDays int      `json:"targetDurationDays"`
	PreservedStructure bool     `json:"preservedStructure"`
	ChangedDestination bool     `json:"changedDestination"`
	FallbackUsed       bool     `json:"fallbackUsed"`
	FallbackReason     string   `json:"fallbackReason,omitempty"`
	MajorChanges       []string `json:"majorChanges"`
	Warnings           []string `json:"warnings"`
}
