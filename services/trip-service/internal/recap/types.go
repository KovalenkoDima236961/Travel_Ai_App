// Package recap contains the bounded, privacy-safe contract shared by Trip
// Service and AI Planning Service for post-trip recap generation.
package recap

import (
	"context"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
)

type SourceSummary struct {
	Trip                SourceTrip                 `json:"trip"`
	ItineraryOutcome    SourceItineraryOutcome     `json:"itineraryOutcome"`
	BudgetOutcome       SourceBudgetOutcome        `json:"budgetOutcome"`
	RouteOutcome        SourceRouteOutcome         `json:"routeOutcome"`
	ChecklistOutcome    SourceChecklistOutcome     `json:"checklistOutcome"`
	VerificationOutcome SourceVerificationOutcome  `json:"verificationOutcome"`
	LearningCandidates  []appdto.LearningCandidate `json:"learningCandidates"`
}

type SourceTrip struct {
	Title        string `json:"title"`
	Destination  string `json:"destination"`
	StartDate    string `json:"startDate,omitempty"`
	EndDate      string `json:"endDate,omitempty"`
	DurationDays int    `json:"durationDays"`
	TripType     string `json:"tripType"`
}

type SourceItineraryOutcome struct {
	PlannedItemCount  int      `json:"plannedItemCount"`
	DoneItemCount     int      `json:"doneItemCount"`
	SkippedItemCount  int      `json:"skippedItemCount"`
	DelayedItemCount  int      `json:"delayedItemCount"`
	UnknownItemCount  int      `json:"unknownItemCount"`
	TopCompletedItems []string `json:"topCompletedItems"`
	TopSkippedItems   []string `json:"topSkippedItems"`
}

type SourceBudgetOutcome struct {
	PlannedTotal           *appdto.RecapMoney          `json:"plannedTotal,omitempty"`
	ActualTotal            *appdto.RecapMoney          `json:"actualTotal,omitempty"`
	Variance               *appdto.RecapMoney          `json:"variance,omitempty"`
	ReceiptCoveragePercent int                         `json:"receiptCoveragePercent"`
	TopCategories          []appdto.RecapCategoryTotal `json:"topCategories"`
}

type SourceRouteOutcome struct {
	Stops                    []string `json:"stops"`
	TransportModes           []string `json:"transportModes"`
	VerifiedTransportCount   int      `json:"verifiedTransportCount"`
	UnverifiedTransportCount int      `json:"unverifiedTransportCount"`
	Issues                   []string `json:"issues"`
}

type SourceChecklistOutcome struct {
	CompletedChecklistItems int `json:"completedChecklistItems"`
	TotalChecklistItems     int `json:"totalChecklistItems"`
	CompletedReminders      int `json:"completedReminders"`
	TotalReminders          int `json:"totalReminders"`
}

type SourceVerificationOutcome struct {
	Score         int      `json:"score,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	VerifiedCount int      `json:"verifiedCount"`
	StaleCount    int      `json:"staleCount"`
	MissingCount  int      `json:"missingCount"`
	Issues        []string `json:"issues"`
}

type GenerateRequest struct {
	Language                  string        `json:"language"`
	SourceSummary             SourceSummary `json:"sourceSummary"`
	Style                     string        `json:"style"`
	IncludeLearningCandidates bool          `json:"includeLearningCandidates"`
}

type GenerateResponse struct {
	Recap       appdto.RecapJSON `json:"recap"`
	Warnings    []string         `json:"warnings"`
	Assumptions []string         `json:"assumptions"`
}

type Client interface {
	Generate(context.Context, GenerateRequest) (GenerateResponse, error)
}
