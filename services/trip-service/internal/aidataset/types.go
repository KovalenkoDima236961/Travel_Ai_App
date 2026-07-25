package aidataset

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	SchemaVersion = "ai_dataset_v1"

	ConsentVersionV1 = "ai-training-consent-v1"

	ProjectStatusDraft    = "draft"
	ProjectStatusActive   = "active"
	ProjectStatusFrozen   = "frozen"
	ProjectStatusArchived = "archived"

	TaskGroundedItineraryGeneration = "grounded_itinerary_generation"
	TaskItineraryGeneration         = "itinerary_generation"
	TaskDayRegeneration             = "day_regeneration"
	TaskItemRegeneration            = "item_regeneration"
	TaskPlaceReplacement            = "place_replacement"
	TaskPolicyRepair                = "policy_repair"
	TaskBudgetOptimization          = "budget_optimization"
	TaskRouteAlternatives           = "route_alternatives"
	TaskChecklistGeneration         = "checklist_generation"
	TaskCopilotResponse             = "copilot_response"
	TaskRecapGeneration             = "recap_generation"

	ConsentNotRequired = "not_required"
	ConsentPending     = "pending"
	ConsentGranted     = "granted"
	ConsentRevoked     = "revoked"
	ConsentProhibited  = "prohibited"

	SanitizationPending     = "pending"
	SanitizationPassed      = "passed"
	SanitizationFailed      = "failed"
	SanitizationNeedsReview = "needs_review"

	QualityPending     = "pending"
	QualityPassed      = "passed"
	QualityFailed      = "failed"
	QualityNeedsReview = "needs_review"

	ReviewPending      = "pending"
	ReviewApproved     = "approved"
	ReviewRejected     = "rejected"
	ReviewNeedsChanges = "needs_changes"

	SplitTrain      = "train"
	SplitValidation = "validation"
	SplitTest       = "test"
	SplitHoldout    = "holdout"

	ExportNotExported = "not_exported"
	ExportExported    = "exported"
	ExportInvalidated = "invalidated"

	VersionStatusBuilding    = "building"
	VersionStatusReady       = "ready"
	VersionStatusExported    = "exported"
	VersionStatusInvalidated = "invalidated"
	VersionStatusArchived    = "archived"

	ReviewActionCandidateCreated   = "candidate_created"
	ReviewActionSanitizationPassed = "sanitization_passed"
	ReviewActionSanitizationFailed = "sanitization_failed"
	ReviewActionQualityScored      = "quality_scored"
	ReviewActionApproved           = "approved"
	ReviewActionRejected           = "rejected"
	ReviewActionSplitAssigned      = "split_assigned"
	ReviewActionExported           = "exported"
	ReviewActionInvalidated        = "invalidated"
	ReviewActionConsentRevoked     = "consent_revoked"

	ScopeGlobalFutureExamples = "global_future_examples"
	ScopeTrip                 = "trip"
	ScopeItineraryVersion     = "itinerary_version"
	ScopeFeedbackSignal       = "feedback_signal"
	ScopeTemplate             = "template"
	ScopeRecap                = "recap"

	JobExtractCandidates  = "ai_dataset_extract_candidates"
	JobResanitizeExamples = "ai_dataset_resanitize_examples"
	JobRescoreExamples    = "ai_dataset_rescore_examples"
	JobBuildVersion       = "ai_dataset_build_version"
	JobExportVersion      = "ai_dataset_export_version"
)

var allowedTaskTypes = map[string]struct{}{
	TaskGroundedItineraryGeneration: {},
	TaskItineraryGeneration:         {},
	TaskDayRegeneration:             {},
	TaskItemRegeneration:            {},
	TaskPlaceReplacement:            {},
	TaskPolicyRepair:                {},
	TaskBudgetOptimization:          {},
	TaskRouteAlternatives:           {},
	TaskChecklistGeneration:         {},
	TaskCopilotResponse:             {},
	TaskRecapGeneration:             {},
}

type Config struct {
	ExportEnabled           bool
	ExportDir               string
	ExportRetentionDays     int
	MinAutoReviewScore      float64
	MinApprovalScore        float64
	MaxDuplicateSimilarity  float64
	RequireHumanReview      bool
	GoldenCasesDir          string
	ManualExamplesDir       string
	TrainingInstructionText string
}

func DefaultConfig() Config {
	return Config{
		ExportEnabled:           false,
		ExportDir:               "./data/ai-datasets",
		ExportRetentionDays:     30,
		MinAutoReviewScore:      0.70,
		MinApprovalScore:        0.85,
		MaxDuplicateSimilarity:  0.95,
		RequireHumanReview:      true,
		GoldenCasesDir:          "./evals/ai-itinerary/cases",
		ManualExamplesDir:       "./data/ai-training/manual",
		TrainingInstructionText: "Generate schema-valid grounded travel itineraries.",
	}
}

type DatasetProject struct {
	ID                  uuid.UUID  `json:"id"`
	Key                 string     `json:"key"`
	Name                string     `json:"name"`
	Description         *string    `json:"description,omitempty"`
	TaskType            string     `json:"taskType"`
	SchemaVersion       string     `json:"schemaVersion"`
	Status              string     `json:"status"`
	MinimumQualityScore float64    `json:"minimumQualityScore"`
	ConsentRequired     bool       `json:"consentRequired"`
	CreatedByUserID     *uuid.UUID `json:"createdByUserId,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	ArchivedAt          *time.Time `json:"archivedAt,omitempty"`
}

type TrainingConsent struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"userId"`
	ScopeType      string     `json:"scopeType"`
	ScopeID        *string    `json:"scopeId,omitempty"`
	ConsentVersion string     `json:"consentVersion"`
	Status         string     `json:"status"`
	GrantedAt      *time.Time `json:"grantedAt,omitempty"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type TrainingConsentResponse struct {
	Status         string           `json:"status"`
	Granted        bool             `json:"granted"`
	ConsentVersion string           `json:"consentVersion"`
	Record         *TrainingConsent `json:"record,omitempty"`
	ExcludedData   []string         `json:"excludedData"`
}

type TrainingExample struct {
	ID                 uuid.UUID       `json:"id"`
	DatasetProjectID   uuid.UUID       `json:"datasetProjectId"`
	SourceType         string          `json:"sourceType"`
	SourceEntityType   *string         `json:"sourceEntityType,omitempty"`
	SourceEntityID     *string         `json:"sourceEntityId,omitempty"`
	UserID             *uuid.UUID      `json:"userId,omitempty"`
	TripID             *uuid.UUID      `json:"tripId,omitempty"`
	TaskType           string          `json:"taskType"`
	Language           string          `json:"language"`
	SchemaVersion      string          `json:"schemaVersion"`
	InputJSON          json.RawMessage `json:"input"`
	GroundingJSON      json.RawMessage `json:"grounding,omitempty"`
	ExpectedOutputJSON json.RawMessage `json:"expectedOutput"`
	NegativeOutputJSON json.RawMessage `json:"negativeOutput,omitempty"`
	LabelsJSON         json.RawMessage `json:"labels"`
	ProvenanceJSON     json.RawMessage `json:"provenance"`
	ConsentStatus      string          `json:"consentStatus"`
	ConsentRecordID    *uuid.UUID      `json:"consentRecordId,omitempty"`
	SanitizationStatus string          `json:"sanitizationStatus"`
	QualityStatus      string          `json:"qualityStatus"`
	ReviewStatus       string          `json:"reviewStatus"`
	QualityScore       *float64        `json:"qualityScore,omitempty"`
	DuplicateGroupID   *uuid.UUID      `json:"duplicateGroupId,omitempty"`
	Split              *string         `json:"split,omitempty"`
	ExportStatus       string          `json:"exportStatus"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
	ReviewedByUserID   *uuid.UUID      `json:"reviewedByUserId,omitempty"`
	ReviewedAt         *time.Time      `json:"reviewedAt,omitempty"`
	RejectedReason     *string         `json:"rejectedReason,omitempty"`
}

type DatasetVersion struct {
	ID                uuid.UUID       `json:"id"`
	DatasetProjectID  uuid.UUID       `json:"datasetProjectId"`
	Version           string          `json:"version"`
	Status            string          `json:"status"`
	SchemaVersion     string          `json:"schemaVersion"`
	ExampleCount      int             `json:"exampleCount"`
	TrainCount        int             `json:"trainCount"`
	ValidationCount   int             `json:"validationCount"`
	TestCount         int             `json:"testCount"`
	HoldoutCount      int             `json:"holdoutCount"`
	ManifestJSON      json.RawMessage `json:"manifest"`
	Checksum          *string         `json:"checksum,omitempty"`
	ExportPath        *string         `json:"exportPath,omitempty"`
	CreatedByUserID   *uuid.UUID      `json:"createdByUserId,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	FinalizedAt       *time.Time      `json:"finalizedAt,omitempty"`
	InvalidatedAt     *time.Time      `json:"invalidatedAt,omitempty"`
	InvalidatedReason *string         `json:"invalidatedReason,omitempty"`
}

type ReviewEvent struct {
	ID                uuid.UUID       `json:"id"`
	TrainingExampleID *uuid.UUID      `json:"trainingExampleId,omitempty"`
	DatasetVersionID  *uuid.UUID      `json:"datasetVersionId,omitempty"`
	ActorUserID       *uuid.UUID      `json:"actorUserId,omitempty"`
	Action            string          `json:"action"`
	OldStatus         *string         `json:"oldStatus,omitempty"`
	NewStatus         *string         `json:"newStatus,omitempty"`
	Reason            *string         `json:"reason,omitempty"`
	Metadata          json.RawMessage `json:"metadata"`
	CreatedAt         time.Time       `json:"createdAt"`
}

type CreateProjectInput struct {
	Key                 string  `json:"key"`
	Name                string  `json:"name"`
	Description         *string `json:"description"`
	TaskType            string  `json:"taskType"`
	SchemaVersion       string  `json:"schemaVersion"`
	MinimumQualityScore float64 `json:"minimumQualityScore"`
	ConsentRequired     bool    `json:"consentRequired"`
}

type CreateExampleInput struct {
	DatasetProjectID   uuid.UUID       `json:"datasetProjectId"`
	SourceType         string          `json:"sourceType"`
	SourceEntityType   *string         `json:"sourceEntityType"`
	SourceEntityID     *string         `json:"sourceEntityId"`
	UserID             *uuid.UUID      `json:"userId"`
	TripID             *uuid.UUID      `json:"tripId"`
	TaskType           string          `json:"taskType"`
	Language           string          `json:"language"`
	SchemaVersion      string          `json:"schemaVersion"`
	InputJSON          json.RawMessage `json:"input"`
	GroundingJSON      json.RawMessage `json:"grounding"`
	ExpectedOutputJSON json.RawMessage `json:"expectedOutput"`
	NegativeOutputJSON json.RawMessage `json:"negativeOutput"`
	LabelsJSON         json.RawMessage `json:"labels"`
	ProvenanceJSON     json.RawMessage `json:"provenance"`
	ConsentStatus      string          `json:"consentStatus"`
	ConsentRecordID    *uuid.UUID      `json:"consentRecordId"`
}

type ExampleFilters struct {
	DatasetProjectID   *uuid.UUID
	ReviewStatus       string
	SanitizationStatus string
	QualityStatus      string
	ConsentStatus      string
	Language           string
	TaskType           string
	Split              string
	SourceType         string
	Limit              int
	Offset             int
}

type ReviewInput struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type BuildVersionInput struct {
	DatasetProjectID    uuid.UUID `json:"datasetProjectId"`
	Version             string    `json:"version"`
	MinimumQualityScore float64   `json:"minimumQualityScore"`
	Languages           []string  `json:"languages"`
	TaskTypes           []string  `json:"taskTypes"`
	IncludeManual       bool      `json:"includeManual"`
}

type FineTuningReadiness struct {
	Ready                    bool           `json:"ready"`
	Blockers                 []string       `json:"blockers"`
	ApprovedExampleCount     int            `json:"approvedExampleCount"`
	TaskDistribution         map[string]int `json:"taskDistribution"`
	LanguageDistribution     map[string]int `json:"languageDistribution"`
	ConsentCoverage          map[string]int `json:"consentCoverage"`
	SanitizationFailureCount int            `json:"sanitizationFailureCount"`
	DuplicateCount           int            `json:"duplicateCount"`
	HoldoutCount             int            `json:"holdoutCount"`
	BaselineEvalStatus       string         `json:"baselineEvalStatus"`
	Recommendation           string         `json:"recommendation"`
}

type ExportStatus struct {
	VersionID  uuid.UUID `json:"versionId"`
	Status     string    `json:"status"`
	ExportPath *string   `json:"exportPath,omitempty"`
	Checksum   *string   `json:"checksum,omitempty"`
}

func IsAllowedTaskType(taskType string) bool {
	_, ok := allowedTaskTypes[taskType]
	return ok
}
