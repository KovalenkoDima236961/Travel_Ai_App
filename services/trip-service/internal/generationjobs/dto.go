package generationjobs

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateRequest struct {
	JobType                   entity.GenerationJobType `json:"jobType"`
	ExpectedItineraryRevision *int                     `json:"expectedItineraryRevision"`
	Instruction               *string                  `json:"instruction"`
	DayNumber                 *int                     `json:"dayNumber"`
	ItemIndex                 *int                     `json:"itemIndex"`
	Payload                   json.RawMessage          `json:"payload,omitempty"`
}

// TemplateAdaptationBudget is the optional target budget in an adaptation job
// request.
type TemplateAdaptationBudget struct {
	Amount   *float64 `json:"amount"`
	Currency string   `json:"currency"`
}

// CreateTemplateAdaptationRequest is the decoded POST body for
// /trip-templates/{templateId}/adaptation-jobs.
type CreateTemplateAdaptationRequest struct {
	Title                   string                    `json:"title"`
	Destination             string                    `json:"destination"`
	StartDate               string                    `json:"startDate"`
	DurationDays            int                       `json:"durationDays"`
	WorkspaceID             *uuid.UUID                `json:"workspaceId"`
	Budget                  *TemplateAdaptationBudget `json:"budget"`
	Travelers               *int32                    `json:"travelers"`
	Pace                    string                    `json:"pace"`
	Interests               []string                  `json:"interests"`
	Avoid                   []string                  `json:"avoid"`
	SpecialInstructions     string                    `json:"specialInstructions"`
	FallbackToDeterministic *bool                     `json:"fallbackToDeterministic"`
}

type JobEnvelope struct {
	Job JobResponse `json:"job"`
}

type ListResponse struct {
	Items []JobResponse `json:"items"`
	Limit int           `json:"limit"`
}

type JobResponse struct {
	ID                        uuid.UUID                  `json:"id"`
	TripID                    uuid.UUID                  `json:"tripId"`
	RequestedByUserID         uuid.UUID                  `json:"requestedByUserId"`
	JobType                   entity.GenerationJobType   `json:"jobType"`
	Status                    entity.GenerationJobStatus `json:"status"`
	ExpectedItineraryRevision int                        `json:"expectedItineraryRevision"`
	Instruction               *string                    `json:"instruction"`
	DayNumber                 *int                       `json:"dayNumber"`
	ItemIndex                 *int                       `json:"itemIndex"`
	Payload                   json.RawMessage            `json:"payload,omitempty"`
	ResultPayload             json.RawMessage            `json:"resultPayload,omitempty"`
	GenerationQuality         any                        `json:"generationQuality,omitempty"`
	ErrorCode                 *string                    `json:"errorCode"`
	// ErrorMessage is retained internally for operations logging. The public job
	// response deliberately exposes only ErrorMessageSafe so provider payloads,
	// raw model output, and implementation details never reach the browser.
	ErrorMessage            *string    `json:"-"`
	ErrorMessageSafe        *string    `json:"errorMessageSafe,omitempty"`
	CanRetry                bool       `json:"canRetry"`
	RetryRecommendedMode    *string    `json:"retryRecommendedMode,omitempty"`
	ResultItineraryRevision *int       `json:"resultItineraryRevision"`
	CreatedAt               time.Time  `json:"createdAt"`
	StartedAt               *time.Time `json:"startedAt"`
	CompletedAt             *time.Time `json:"completedAt"`
	CancelledAt             *time.Time `json:"cancelledAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

func NewJobEnvelope(job *entity.GenerationJob) JobEnvelope {
	return JobEnvelope{Job: NewJobResponse(job)}
}

func NewListResponse(jobs []entity.GenerationJob, limit int) ListResponse {
	items := make([]JobResponse, 0, len(jobs))
	for i := range jobs {
		items = append(items, NewJobResponse(&jobs[i]))
	}
	return ListResponse{Items: items, Limit: limit}
}

func NewJobResponse(job *entity.GenerationJob) JobResponse {
	return JobResponse{
		ID:                        job.ID,
		TripID:                    job.TripID,
		RequestedByUserID:         job.RequestedByUserID,
		JobType:                   job.JobType,
		Status:                    job.Status,
		ExpectedItineraryRevision: job.ExpectedItineraryRevision,
		Instruction:               job.Instruction,
		DayNumber:                 job.DayNumber,
		ItemIndex:                 job.ItemIndex,
		Payload:                   job.Payload,
		ResultPayload:             job.ResultPayload,
		GenerationQuality:         generationQualityFromPayload(job.ResultPayload),
		ErrorCode:                 job.ErrorCode,
		ErrorMessage:              job.ErrorMessage,
		ErrorMessageSafe:          safeGenerationErrorMessage(job.ErrorCode),
		CanRetry:                  generationJobCanRetry(job),
		RetryRecommendedMode:      retryRecommendedMode(job),
		ResultItineraryRevision:   job.ResultItineraryRevision,
		CreatedAt:                 job.CreatedAt,
		StartedAt:                 job.StartedAt,
		CompletedAt:               job.CompletedAt,
		CancelledAt:               job.CancelledAt,
		UpdatedAt:                 job.UpdatedAt,
	}
}

func safeGenerationErrorMessage(code *string) *string {
	if code == nil || *code == "" {
		return nil
	}
	message := "We could not finish this generation. Please try again."
	switch *code {
	case ErrorItineraryConflict:
		message = "This trip changed while generation was running. Reload the latest itinerary before trying again."
	case "ai_invalid_json", "ai_generation_schema_invalid", "ai_output_invalid":
		message = "The AI response could not be converted into a valid itinerary."
	case "ai_validation_failed", "ai_generation_validation_failed":
		message = "The generated itinerary did not pass the app’s consistency checks."
	case "ai_repair_failed", "ai_generation_repair_failed":
		message = "The itinerary needed fixes that could not be applied automatically."
	case ErrorProviderRateLimited, ErrorProviderQuotaExceeded:
		message = "A planning provider has reached a temporary limit."
	case ErrorProviderLimitsUnavailable:
		message = "A planning provider is temporarily unavailable."
	case "permission_denied", "forbidden":
		message = "You do not have permission to generate this itinerary."
	case "missing_required_trip_data", ErrorValidationFailed:
		message = "Some trip details needed for generation are missing or invalid."
	}
	return &message
}

func generationJobCanRetry(job *entity.GenerationJob) bool {
	if job.Status != entity.GenerationJobStatusFailed || job.ErrorCode == nil {
		return false
	}
	if *job.ErrorCode == "permission_denied" || *job.ErrorCode == "forbidden" {
		return false
	}
	return true
}

func retryRecommendedMode(job *entity.GenerationJob) *string {
	if !generationJobCanRetry(job) {
		return nil
	}
	mode := "retry"
	if job.JobType == entity.GenerationJobTypeFullGeneration {
		mode = "simpler_request"
	}
	return &mode
}

func generationQualityFromPayload(payload json.RawMessage) any {
	if len(payload) == 0 {
		return nil
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil
	}
	return body["generationQuality"]
}
