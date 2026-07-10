package tripdiscovery

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
)

type Mode string

const (
	ModePrompt   Mode = "prompt"
	ModeSurprise Mode = "surprise"
	ModeRefine   Mode = "refine"
)

type Budget struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type BudgetEstimate struct {
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Confidence string  `json:"confidence"`
}

type TripPreview struct {
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	SampleDay []string `json:"sampleDay"`
}

type Concern struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Suggestion struct {
	ID                          string               `json:"id"`
	SuggestionType              string               `json:"suggestionType,omitempty"`
	Destination                 string               `json:"destination"`
	City                        string               `json:"city"`
	Country                     string               `json:"country"`
	Region                      *string              `json:"region,omitempty"`
	MatchScore                  int                  `json:"matchScore"`
	RecommendedDurationDays     int                  `json:"recommendedDurationDays"`
	BestFor                     []string             `json:"bestFor"`
	EstimatedBudget             BudgetEstimate       `json:"estimatedBudget"`
	BestTimeToGo                string               `json:"bestTimeToGo"`
	WhyItFits                   string               `json:"whyItFits"`
	PossibleDownsides           []string             `json:"possibleDownsides"`
	TripPreview                 TripPreview          `json:"tripPreview"`
	Tags                        []string             `json:"tags"`
	SuggestedPromptForItinerary string               `json:"suggestedPromptForItinerary"`
	Route                       *aggregate.TripRoute `json:"route,omitempty"`
	Concerns                    []Concern            `json:"concerns"`
}

type SuggestionResponse struct {
	SessionTitle      string       `json:"sessionTitle"`
	Suggestions       []Suggestion `json:"suggestions"`
	FollowUpQuestions []string     `json:"followUpQuestions"`
	Warnings          []string     `json:"warnings"`
}

type UserProfile struct {
	HomeCity          *string `json:"homeCity,omitempty"`
	HomeCountry       *string `json:"homeCountry,omitempty"`
	PreferredCurrency string  `json:"preferredCurrency"`
	PreferredLanguage string  `json:"preferredLanguage"`
}

type UserPreferences struct {
	TravelStyles       []string `json:"travelStyles"`
	Pace               string   `json:"pace,omitempty"`
	MaxWalkingKmPerDay *float64 `json:"maxWalkingKmPerDay,omitempty"`
	FoodPreferences    []string `json:"foodPreferences"`
	Avoid              []string `json:"avoid"`
	PreferredTransport []string `json:"preferredTransport"`
}

type UserContext struct {
	HomeCity          *string          `json:"homeCity,omitempty"`
	HomeCountry       *string          `json:"homeCountry,omitempty"`
	PreferredCurrency string           `json:"preferredCurrency"`
	PreferredLanguage string           `json:"preferredLanguage"`
	Preferences       *UserPreferences `json:"preferences,omitempty"`
}

type TripContext struct {
	DurationDays    *int    `json:"durationDays,omitempty"`
	StartDate       *string `json:"startDate,omitempty"`
	DateFlexibility string  `json:"dateFlexibility,omitempty"`
	Budget          *Budget `json:"budget,omitempty"`
	Travelers       int     `json:"travelers"`
	Origin          string  `json:"origin,omitempty"`
	Scope           string  `json:"scope"`
}

type PreviousTripSummary struct {
	Destination  string   `json:"destination"`
	Country      string   `json:"country,omitempty"`
	DurationDays int32    `json:"durationDays"`
	Budget       *Budget  `json:"budget,omitempty"`
	Tags         []string `json:"tags"`
	Pace         string   `json:"pace,omitempty"`
	CreatedAt    string   `json:"createdAt"`
}

type PolicyConstraints struct {
	Summary string          `json:"summary"`
	Rules   json.RawMessage `json:"rules"`
}

type Refinement struct {
	PreviousSuggestions  []Suggestion `json:"previousSuggestions"`
	SelectedSuggestionID string       `json:"selectedSuggestionId,omitempty"`
	Instruction          string       `json:"instruction"`
}

type Constraints struct {
	SuggestionCount        int    `json:"suggestionCount"`
	AvoidPreviouslyVisited bool   `json:"avoidPreviouslyVisited"`
	PreferNovelty          bool   `json:"preferNovelty"`
	IncludeReasoning       bool   `json:"includeReasoning"`
	MaxTravelComplexity    string `json:"maxTravelComplexity"`
}

type AIRequest struct {
	Prompt                     string                                   `json:"prompt"`
	Mode                       Mode                                     `json:"mode"`
	OutputLanguage             string                                   `json:"outputLanguage"`
	UserContext                *UserContext                             `json:"userContext,omitempty"`
	TripContext                TripContext                              `json:"tripContext"`
	PreviousTrips              []PreviousTripSummary                    `json:"previousTrips"`
	WorkspacePolicyConstraints *PolicyConstraints                       `json:"workspacePolicyConstraints,omitempty"`
	PlanningConstraints        *planningconstraints.PlanningConstraints `json:"planningConstraints,omitempty"`
	Refinement                 *Refinement                              `json:"refinement,omitempty"`
	Constraints                Constraints                              `json:"constraints"`
}

type Session struct {
	ID              uuid.UUID          `json:"id"`
	UserID          uuid.UUID          `json:"-"`
	WorkspaceID     *uuid.UUID         `json:"workspaceId,omitempty"`
	ParentSessionID *uuid.UUID         `json:"parentSessionId,omitempty"`
	Mode            Mode               `json:"mode"`
	Prompt          string             `json:"prompt,omitempty"`
	OutputLanguage  string             `json:"outputLanguage"`
	Status          string             `json:"status"`
	Request         AIRequest          `json:"request"`
	Response        SuggestionResponse `json:"response"`
	CreatedTripID   *uuid.UUID         `json:"createdTripId,omitempty"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}

type DiscoverInput struct {
	Prompt                 string     `json:"prompt"`
	Scope                  string     `json:"scope"`
	WorkspaceID            *uuid.UUID `json:"workspaceId,omitempty"`
	DurationDays           *int       `json:"durationDays,omitempty"`
	StartDate              *string    `json:"startDate,omitempty"`
	DateFlexibility        string     `json:"dateFlexibility,omitempty"`
	Budget                 *Budget    `json:"budget,omitempty"`
	Travelers              int        `json:"travelers"`
	Origin                 string     `json:"origin,omitempty"`
	QuickChips             []string   `json:"quickChips,omitempty"`
	OutputLanguage         string     `json:"outputLanguage,omitempty"`
	AvoidPreviouslyVisited *bool      `json:"avoidPreviouslyVisited,omitempty"`
	PreferNovelty          *bool      `json:"preferNovelty,omitempty"`
	NoveltyLevel           string     `json:"noveltyLevel,omitempty"`
}

type RefineInput struct {
	Instruction          string `json:"instruction"`
	SelectedSuggestionID string `json:"selectedSuggestionId,omitempty"`
	FeedbackType         string `json:"feedbackType,omitempty"`
	OutputLanguage       string `json:"outputLanguage,omitempty"`
}

type CreateTripInput struct {
	Title                 string               `json:"title,omitempty"`
	TripType              string               `json:"tripType,omitempty"`
	Route                 *aggregate.TripRoute `json:"route,omitempty"`
	StartDate             string               `json:"startDate,omitempty"`
	DurationDays          int                  `json:"durationDays"`
	Budget                *Budget              `json:"budget,omitempty"`
	Travelers             int32                `json:"travelers"`
	WorkspaceID           *uuid.UUID           `json:"workspaceId,omitempty"`
	AutoGenerateItinerary bool                 `json:"autoGenerateItinerary"`
}

type CreateTripResult struct {
	Trip          *entity.Trip
	GenerationJob *entity.GenerationJob
}

type VoteSuggestionInput struct {
	Vote     entity.DiscoverySuggestionVoteValue `json:"vote"`
	Metadata map[string]any                      `json:"metadata,omitempty"`
}

type SuggestionVoteSummary struct {
	SuggestionID string         `json:"suggestionId"`
	Counts       map[string]int `json:"counts"`
	Score        int            `json:"score"`
	CurrentUser  string         `json:"currentUserVote,omitempty"`
}

type SuggestionVotesResponse struct {
	SessionID uuid.UUID               `json:"sessionId"`
	Items     []SuggestionVoteSummary `json:"items"`
}
