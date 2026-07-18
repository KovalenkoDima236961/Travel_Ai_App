package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const TripRecapSchemaVersion = "trip_recap_v1"

type RecapMoney struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type RecapHighlight struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	DayNumber   int    `json:"dayNumber,omitempty"`
	ItemID      string `json:"itemId,omitempty"`
}

type RecapPlannedVsActual struct {
	PlannedItemCount int      `json:"plannedItemCount"`
	DoneItemCount    int      `json:"doneItemCount"`
	SkippedItemCount int      `json:"skippedItemCount"`
	DelayedItemCount int      `json:"delayedItemCount"`
	UnknownItemCount int      `json:"unknownItemCount"`
	CompletionRate   float64  `json:"completionRate"`
	Notes            string   `json:"notes,omitempty"`
	SkippedItems     []string `json:"skippedItems"`
	DelayedItems     []string `json:"delayedItems"`
}

type BudgetRecap struct {
	PlannedTotal           *RecapMoney          `json:"plannedTotal,omitempty"`
	ActualTotal            *RecapMoney          `json:"actualTotal,omitempty"`
	VarianceAmount         *RecapMoney          `json:"varianceAmount,omitempty"`
	VariancePercent        *float64             `json:"variancePercent,omitempty"`
	ReceiptCoveragePercent int                  `json:"receiptCoveragePercent"`
	TopCategories          []RecapCategoryTotal `json:"topCategories"`
	Notes                  string               `json:"notes,omitempty"`
}

type RecapCategoryTotal struct {
	Category string     `json:"category"`
	Total    RecapMoney `json:"total"`
}

type RouteTransportRecap struct {
	Summary         string   `json:"summary,omitempty"`
	Issues          []string `json:"issues"`
	SuccessfulModes []string `json:"successfulModes"`
	ProblemModes    []string `json:"problemModes"`
}

type VerificationRecap struct {
	Summary string   `json:"summary,omitempty"`
	Issues  []string `json:"issues"`
}

type ChecklistReminderRecap struct {
	CompletedChecklistItems int    `json:"completedChecklistItems"`
	TotalChecklistItems     int    `json:"totalChecklistItems"`
	CompletedReminders      int    `json:"completedReminders"`
	TotalReminders          int    `json:"totalReminders"`
	Notes                   string `json:"notes,omitempty"`
}

type LearningCandidate struct {
	FeedbackType string         `json:"feedbackType"`
	Label        string         `json:"label"`
	EntityType   string         `json:"entityType,omitempty"`
	EntityID     string         `json:"entityId,omitempty"`
	Value        string         `json:"value,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Approved     bool           `json:"approved"`
}

type TemplateSuggestion struct {
	Recommended bool   `json:"recommended"`
	Title       string `json:"title,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type RecapJSON struct {
	SchemaVersion         string                 `json:"schemaVersion"`
	Title                 string                 `json:"title"`
	Summary               string                 `json:"summary"`
	Highlights            []RecapHighlight       `json:"highlights"`
	PlannedVsActual       RecapPlannedVsActual   `json:"plannedVsActual"`
	Budget                BudgetRecap            `json:"budget"`
	RouteAndTransport     RouteTransportRecap    `json:"routeAndTransport"`
	Verification          VerificationRecap      `json:"verification"`
	ChecklistAndReminders ChecklistReminderRecap `json:"checklistAndReminders"`
	LessonsLearned        []string               `json:"lessonsLearned"`
	FuturePreferences     []LearningCandidate    `json:"futurePreferences"`
	TemplateSuggestion    TemplateSuggestion     `json:"templateSuggestion"`
	UserEditableNotes     string                 `json:"userEditableNotes"`
}

type RecapPermissions struct {
	CanEdit           bool `json:"canEdit"`
	CanFinalize       bool `json:"canFinalize"`
	CanCreateTemplate bool `json:"canCreateTemplate"`
	CanApplyLearning  bool `json:"canApplyLearning"`
}

type TripRecapView struct {
	ID          uuid.UUID              `json:"id"`
	TripID      uuid.UUID              `json:"tripId"`
	Status      entity.TripRecapStatus `json:"status"`
	Recap       RecapJSON              `json:"recap"`
	FinalizedAt *time.Time             `json:"finalizedAt,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

type TripRecapStatusResponse struct {
	Eligible    bool       `json:"eligible"`
	Reason      string     `json:"reason"`
	HasRecap    bool       `json:"hasRecap"`
	RecapID     *uuid.UUID `json:"recapId"`
	TripEndedAt *string    `json:"tripEndedAt"`
	CanGenerate bool       `json:"canGenerate"`
	CanEdit     bool       `json:"canEdit"`
}

type GetTripRecapResponse struct {
	Recap       TripRecapView       `json:"recap"`
	Permissions RecapPermissions    `json:"permissions"`
	Feedback    []RecapFeedbackView `json:"feedback"`
}

type RecapFeedbackView struct {
	ID                         uuid.UUID      `json:"id"`
	FeedbackType               string         `json:"feedbackType"`
	EntityType                 *string        `json:"entityType,omitempty"`
	EntityID                   *string        `json:"entityId,omitempty"`
	Label                      string         `json:"label"`
	Value                      *string        `json:"value,omitempty"`
	ApprovedForPersonalization bool           `json:"approvedForPersonalization"`
	Metadata                   map[string]any `json:"metadata"`
	CreatedAt                  time.Time      `json:"createdAt"`
}

type GenerateTripRecapInput struct {
	ForceRegenerate bool
	GenerateEarly   bool
	Language        string
}

type SubmitRecapFeedbackInput struct {
	FeedbackType               string
	EntityType                 string
	EntityID                   string
	Label                      string
	Value                      string
	ApprovedForPersonalization bool
	Metadata                   map[string]any
}

type ApplyRecapLearningInput struct {
	FeedbackIDs        []uuid.UUID
	LearningCandidates []LearningCandidate
}

type CreateTemplateFromRecapInput struct {
	Title           string
	Description     string
	Visibility      entity.TripTemplateVisibility
	Tags            []string
	UseRecapLessons bool
}
