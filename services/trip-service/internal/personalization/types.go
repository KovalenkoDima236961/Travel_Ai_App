// Package personalization builds deterministic, user-scoped context for
// suggestions. It deliberately contains only saved preferences, lightweight
// feedback, safe trip aggregates, and applicable workspace policy.
package personalization

import (
	"time"

	"github.com/google/uuid"
)

const SchemaVersion = "personalization_v2"

type Source string

const (
	SourceTripDiscovery       Source = "trip_discovery"
	SourceItineraryGeneration Source = "itinerary_generation"
	SourceDayRegeneration     Source = "day_regeneration"
	SourceItemRegeneration    Source = "item_regeneration"
	SourceRouteAlternatives   Source = "route_alternatives"
	SourceTemplateRanking     Source = "template_ranking"
	SourceTemplateAdaptation  Source = "template_adaptation"
	SourceBudgetSuggestion    Source = "budget_suggestion"
	SourceBudgetOptimization  Source = "budget_optimization"
	SourceChecklistGeneration Source = "checklist_generation"
	SourceCommandCenter       Source = "command_center"
	SourceOnboarding          Source = "onboarding"
	SourceSettings            Source = "settings"
)

type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Profile struct {
	HomeCity          string `json:"homeCity,omitempty"`
	HomeCountry       string `json:"homeCountry,omitempty"`
	PreferredCurrency string `json:"preferredCurrency,omitempty"`
	PreferredLanguage string `json:"preferredLanguage,omitempty"`
}

type Preferences struct {
	TravelStyles        []string `json:"travelStyles"`
	Pace                string   `json:"pace,omitempty"`
	MaxWalkingKmPerDay  *float64 `json:"maxWalkingKmPerDay,omitempty"`
	FoodPreferences     []string `json:"foodPreferences"`
	DietaryRestrictions []string `json:"dietaryRestrictions"`
	Avoid               []string `json:"avoid"`
	PreferredTransport  []string `json:"preferredTransport"`
	AccommodationStyle  []string `json:"accommodationStyle"`
}

type DerivedSignals struct {
	BudgetComfort     string   `json:"budgetComfort"`
	WalkingTolerance  string   `json:"walkingTolerance"`
	NoveltyPreference string   `json:"noveltyPreference"`
	TransportBias     []string `json:"transportBias"`
	ActivityBias      []string `json:"activityBias"`
	AvoidBias         []string `json:"avoidBias"`
	PlanningStyle     string   `json:"planningStyle"`
}

type PastTripSignals struct {
	PastDestinationCount          int      `json:"pastDestinationCount"`
	RecentDestinations            []string `json:"recentDestinations"`
	RepeatedStyles                []string `json:"repeatedStyles"`
	AverageTripDurationDays       int      `json:"averageTripDurationDays,omitempty"`
	AverageBudgetPerDay           *Money   `json:"averageBudgetPerDay,omitempty"`
	PreferredTransportFromHistory []string `json:"preferredTransportFromHistory"`
	OverBudgetPattern             bool     `json:"overBudgetPattern"`
}

type FeedbackSignals struct {
	LikedDestinations    []string `json:"likedDestinations"`
	DislikedDestinations []string `json:"dislikedDestinations"`
	LikedStyles          []string `json:"likedStyles"`
	DislikedStyles       []string `json:"dislikedStyles"`
	TooExpensiveCount    int      `json:"tooExpensiveCount"`
	TooMuchWalkingCount  int      `json:"tooMuchWalkingCount"`
	PreferTrainCount     int      `json:"preferTrainCount"`
	BudgetSensitivity    string   `json:"budgetSensitivity"`
	WalkingSensitivity   string   `json:"walkingSensitivity"`
	RecentFeedbackCount  int      `json:"recentFeedbackCount"`
}

type WorkspacePolicy struct {
	Enabled                  bool     `json:"enabled"`
	PreferredCurrency        string   `json:"preferredCurrency,omitempty"`
	BlockingRuleCount        int      `json:"blockingRuleCount"`
	DisallowedTransportModes []string `json:"disallowedTransportModes"`
	MaxDailyBudget           *Money   `json:"maxDailyBudget,omitempty"`
}

type MissingField struct {
	Field  string `json:"field"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}
type RecommendedAction struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}
type Completeness struct {
	Score              int                 `json:"score"`
	Level              string              `json:"level"`
	MissingFields      []MissingField      `json:"missingFields"`
	RecommendedActions []RecommendedAction `json:"recommendedActions"`
}

type Context struct {
	SchemaVersion     string          `json:"schemaVersion"`
	UserID            uuid.UUID       `json:"userId"`
	WorkspaceID       *uuid.UUID      `json:"workspaceId,omitempty"`
	Source            Source          `json:"source"`
	Profile           Profile         `json:"profile"`
	Preferences       Preferences     `json:"preferences"`
	DerivedSignals    DerivedSignals  `json:"derivedSignals"`
	PastTripSignals   PastTripSignals `json:"pastTripSignals"`
	FeedbackSignals   FeedbackSignals `json:"feedbackSignals"`
	WorkspacePolicy   WorkspacePolicy `json:"workspacePolicy"`
	Completeness      Completeness    `json:"completeness"`
	Warnings          []string        `json:"warnings"`
	ExplanationInputs []string        `json:"explanationInputs"`
}

// PlanningSummary is the privacy-minimized form forwarded to AI Planning. It
// intentionally excludes user/workspace IDs and individual feedback records.
type PlanningSummary struct {
	SchemaVersion     string          `json:"schemaVersion"`
	CompletenessScore int             `json:"completenessScore"`
	TravelStyles      []string        `json:"travelStyles"`
	TransportBias     []string        `json:"transportBias"`
	ActivityBias      []string        `json:"activityBias"`
	AvoidBias         []string        `json:"avoidBias"`
	BudgetComfort     string          `json:"budgetComfort"`
	WalkingTolerance  string          `json:"walkingTolerance"`
	PastTripSignals   PastTripSignals `json:"pastTripSignals"`
	FeedbackSignals   FeedbackSignals `json:"feedbackSignals"`
	ExplanationInputs []string        `json:"explanationInputs"`
}

func (c Context) PlanningSummary() PlanningSummary {
	return PlanningSummary{SchemaVersion: c.SchemaVersion, CompletenessScore: c.Completeness.Score, TravelStyles: append([]string(nil), c.Preferences.TravelStyles...), TransportBias: append([]string(nil), c.DerivedSignals.TransportBias...), ActivityBias: append([]string(nil), c.DerivedSignals.ActivityBias...), AvoidBias: append([]string(nil), c.DerivedSignals.AvoidBias...), BudgetComfort: c.DerivedSignals.BudgetComfort, WalkingTolerance: c.DerivedSignals.WalkingTolerance, PastTripSignals: c.PastTripSignals, FeedbackSignals: c.FeedbackSignals, ExplanationInputs: append([]string(nil), c.ExplanationInputs...)}
}

type FeedbackType string

const (
	FeedbackLike             FeedbackType = "like"
	FeedbackDislike          FeedbackType = "dislike"
	FeedbackTooExpensive     FeedbackType = "too_expensive"
	FeedbackTooMuchWalking   FeedbackType = "too_much_walking"
	FeedbackTooPacked        FeedbackType = "too_packed"
	FeedbackNotMyVibe        FeedbackType = "not_my_vibe"
	FeedbackMoreNature       FeedbackType = "more_nature"
	FeedbackMoreFood         FeedbackType = "more_food"
	FeedbackLessMuseums      FeedbackType = "less_museums"
	FeedbackPreferTrains     FeedbackType = "prefer_trains"
	FeedbackAvoidNightlife   FeedbackType = "avoid_nightlife"
	FeedbackPreferRelaxed    FeedbackType = "prefer_relaxed"
	FeedbackPreferFastPaced  FeedbackType = "prefer_fast_paced"
	FeedbackTooFar           FeedbackType = "too_far"
	FeedbackTooManyTransfers FeedbackType = "too_many_transfers"
	FeedbackOther            FeedbackType = "other"
)

type Feedback struct {
	ID            uuid.UUID      `json:"id"`
	UserID        uuid.UUID      `json:"userId"`
	WorkspaceID   *uuid.UUID     `json:"workspaceId,omitempty"`
	TripID        *uuid.UUID     `json:"tripId,omitempty"`
	EntityType    string         `json:"entityType"`
	EntityID      string         `json:"entityId,omitempty"`
	FeedbackType  FeedbackType   `json:"feedbackType"`
	FeedbackValue string         `json:"feedbackValue,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}
type SubmitFeedbackInput struct {
	WorkspaceID   *uuid.UUID     `json:"workspaceId,omitempty"`
	TripID        *uuid.UUID     `json:"tripId,omitempty"`
	EntityType    string         `json:"entityType"`
	EntityID      string         `json:"entityId,omitempty"`
	FeedbackType  FeedbackType   `json:"feedbackType"`
	FeedbackValue string         `json:"feedbackValue,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type WhyThisFitsYou struct {
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons"`
	Concerns    []string `json:"concerns"`
	SignalsUsed []string `json:"signalsUsed"`
}
