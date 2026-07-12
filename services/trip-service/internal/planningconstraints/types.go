package planningconstraints

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

const SchemaVersion = 1

type Source string

const (
	SourceTripDiscovery                Source = "trip_discovery"
	SourceTripGeneration               Source = "trip_generation"
	SourceDayRegeneration              Source = "day_regeneration"
	SourceItemRegeneration             Source = "item_regeneration"
	SourceTemplateAdaptation           Source = "template_adaptation"
	SourcePolicyRepair                 Source = "policy_repair"
	SourceBudgetOptimization           Source = "budget_optimization"
	SourceRouteGeneration              Source = "route_generation"
	SourceRouteAlternatives            Source = "route_alternatives"
	SourceRouteAlternativeRefinement   Source = "route_alternative_refinement"
	SourceRouteAlternativeApplyPreview Source = "route_alternative_apply_preview"
	SourceRouteUpdatePreview           Source = "route_update_preview"
)

func (s Source) Valid() bool {
	switch s {
	case SourceTripDiscovery,
		SourceTripGeneration,
		SourceDayRegeneration,
		SourceItemRegeneration,
		SourceTemplateAdaptation,
		SourcePolicyRepair,
		SourceBudgetOptimization,
		SourceRouteGeneration,
		SourceRouteAlternatives,
		SourceRouteAlternativeRefinement,
		SourceRouteAlternativeApplyPreview,
		SourceRouteUpdatePreview:
		return true
	default:
		return false
	}
}

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityBlocking Severity = "blocking"
)

type PlanningConstraints struct {
	SchemaVersion       int                  `json:"schemaVersion"`
	Language            string               `json:"language"`
	Scope               string               `json:"scope"`
	WorkspaceID         *uuid.UUID           `json:"workspaceId"`
	Source              Source               `json:"source"`
	Profile             Profile              `json:"profile"`
	Budget              *Budget              `json:"budget,omitempty"`
	Dates               Dates                `json:"dates"`
	Travelers           Travelers            `json:"travelers"`
	Pace                string               `json:"pace"`
	Walking             Walking              `json:"walking"`
	Transport           Transport            `json:"transport"`
	TripStyles          []string             `json:"tripStyles"`
	Accommodation       Accommodation        `json:"accommodation"`
	Interests           []string             `json:"interests"`
	Avoid               []string             `json:"avoid"`
	MustHave            []string             `json:"mustHave"`
	Accessibility       Accessibility        `json:"accessibility"`
	Food                Food                 `json:"food"`
	Route               *Route               `json:"route,omitempty"`
	WorkspacePolicy     *WorkspacePolicy     `json:"workspacePolicy,omitempty"`
	GroupPreferences    *GroupPreferences    `json:"groupPreferences,omitempty"`
	GroupAvailability   *GroupAvailability   `json:"groupAvailability,omitempty"`
	PreviousTripSignals *PreviousTripSignals `json:"previousTripSignals,omitempty"`
	Prompt              *Prompt              `json:"prompt,omitempty"`
	Warnings            []Issue              `json:"warnings"`
	Blockers            []Issue              `json:"blockers"`
}

type Profile struct {
	HomeCity          string `json:"homeCity,omitempty"`
	HomeCountry       string `json:"homeCountry,omitempty"`
	PreferredCurrency string `json:"preferredCurrency,omitempty"`
}

type Budget struct {
	Amount     *float64 `json:"amount,omitempty"`
	Currency   string   `json:"currency"`
	Strictness string   `json:"strictness"`
}

type Dates struct {
	StartDate    string `json:"startDate,omitempty"`
	EndDate      string `json:"endDate,omitempty"`
	DurationDays int    `json:"durationDays,omitempty"`
	Flexibility  string `json:"flexibility"`
}

type Travelers struct {
	Count int32  `json:"count"`
	Type  string `json:"type,omitempty"`
}

type Walking struct {
	MaxKmPerDay    *float64 `json:"maxKmPerDay,omitempty"`
	AllowLongHikes bool     `json:"allowLongHikes"`
}

type Transport struct {
	PreferredModes         []string `json:"preferredModes"`
	AllowedModes           []string `json:"allowedModes"`
	AvoidModes             []string `json:"avoidModes"`
	DisallowedModes        []string `json:"disallowedModes"`
	CarAvailable           bool     `json:"carAvailable"`
	MaxTransferHoursPerDay *int     `json:"maxTransferHoursPerDay,omitempty"`
}

type Accommodation struct {
	PreferredTypes []string `json:"preferredTypes"`
	AvoidTypes     []string `json:"avoidTypes"`
	CampingAllowed bool     `json:"campingAllowed"`
}

type Accessibility struct {
	LowWalkingRequired bool   `json:"lowWalkingRequired"`
	StepFreePreferred  bool   `json:"stepFreePreferred"`
	Notes              string `json:"notes,omitempty"`
}

type Food struct {
	Preferences         []string `json:"preferences"`
	DietaryRestrictions []string `json:"dietaryRestrictions"`
}

type Route struct {
	TripType       string                     `json:"tripType,omitempty"`
	Origin         *aggregate.RoutePlace      `json:"origin,omitempty"`
	Stops          []aggregate.RouteStop      `json:"stops"`
	Legs           []aggregate.RouteLeg       `json:"legs"`
	ReturnToOrigin bool                       `json:"returnToOrigin"`
	Preferences    aggregate.RoutePreferences `json:"preferences"`
}

type WorkspacePolicy struct {
	PolicyID      string          `json:"policyId,omitempty"`
	Summary       string          `json:"summary,omitempty"`
	BlockingRules []string        `json:"blockingRules"`
	WarningRules  []string        `json:"warningRules"`
	Rules         json.RawMessage `json:"rules,omitempty"`
}

type GroupPreferences struct {
	Summary                     string                      `json:"summary"`
	MustHaveItems               []GroupPreferenceItem       `json:"mustHaveItems"`
	SkipCandidates              []GroupPreferenceItem       `json:"skipCandidates"`
	PreferredDestinations       []string                    `json:"preferredDestinations"`
	PreferredTransportModes     []string                    `json:"preferredTransportModes"`
	PreferredDates              []string                    `json:"preferredDates"`
	PreferredRouteAlternativeID string                      `json:"preferredRouteAlternativeId,omitempty"`
	PreferredRouteSessionID     string                      `json:"preferredRouteSessionId,omitempty"`
	RouteAlternativeVotes       []GroupRouteAlternativeVote `json:"routeAlternativeVotes,omitempty"`
	OpenDecisionCount           int                         `json:"openDecisionCount"`
}

type GroupAvailability struct {
	SubmittedCount       int                 `json:"submittedCount"`
	TotalCollaborators   int                 `json:"totalCollaborators"`
	SelectedDateOption   *SelectedDateOption `json:"selectedDateOption,omitempty"`
	MissingResponseCount int                 `json:"missingResponseCount"`
	Notes                string              `json:"notes,omitempty"`
}

type SelectedDateOption struct {
	StartDate         string `json:"startDate"`
	EndDate           string `json:"endDate"`
	DurationDays      int    `json:"durationDays"`
	Score             int    `json:"score"`
	ConflictUserCount int    `json:"conflictUserCount"`
}

type GroupPreferenceItem struct {
	DayNumber int    `json:"dayNumber"`
	ItemIndex int    `json:"itemIndex"`
	ItemID    string `json:"itemId,omitempty"`
	Name      string `json:"name"`
	Count     int    `json:"count"`
	Score     int    `json:"score"`
}

type GroupRouteAlternativeVote struct {
	SessionID     string `json:"sessionId"`
	AlternativeID string `json:"alternativeId"`
	Label         string `json:"label"`
	Score         int    `json:"score"`
	Votes         int    `json:"votes"`
}

type PreviousTripSignals struct {
	VisitedDestinations []string `json:"visitedDestinations"`
	LikedStyles         []string `json:"likedStyles"`
	TypicalDurationDays int      `json:"typicalDurationDays,omitempty"`
	TypicalBudget       *Budget  `json:"typicalBudget,omitempty"`
}

type Prompt struct {
	UserPrompt            string   `json:"userPrompt,omitempty"`
	QuickChips            []string `json:"quickChips"`
	RefinementInstruction string   `json:"refinementInstruction,omitempty"`
}

type Issue struct {
	Type             string            `json:"type"`
	Severity         Severity          `json:"severity"`
	Message          string            `json:"message"`
	Source           string            `json:"source"`
	Affected         map[string]any    `json:"affected,omitempty"`
	SuggestedActions []SuggestedAction `json:"suggestedActions"`
}

type SuggestedAction struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

type Summary struct {
	Language             string   `json:"language"`
	Budget               string   `json:"budget"`
	Pace                 string   `json:"pace"`
	Transport            string   `json:"transport"`
	TripStyles           []string `json:"tripStyles"`
	WorkspacePolicyRules int      `json:"workspacePolicyRules"`
	WarningCount         int      `json:"warningCount"`
	BlockerCount         int      `json:"blockerCount"`
}

type AIContext struct {
	PlanningConstraints *PlanningConstraints `json:"planningConstraints,omitempty"`
	ConstraintSummary   string               `json:"constraintSummary,omitempty"`
	Warnings            []Issue              `json:"warnings,omitempty"`
	Blockers            []Issue              `json:"blockers,omitempty"`
}
