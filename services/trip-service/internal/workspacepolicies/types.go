package workspacepolicies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const SchemaVersion = 1

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityBlocking Severity = "blocking"
)

func (s Severity) Valid() bool {
	return s == SeverityInfo || s == SeverityWarning || s == SeverityBlocking
}

type Rule struct {
	Enabled  bool     `json:"enabled"`
	Severity Severity `json:"severity"`
}

type MoneyRule struct {
	Rule
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type ItemCostRule struct {
	MoneyRule
	Categories []string `json:"categories"`
}

type WalkingRule struct {
	Rule
	Km float64 `json:"km"`
}

type LateActivityRule struct {
	Rule
	Time string `json:"time"`
}

type RestTimeRule struct {
	Rule
	Minutes int `json:"minutes"`
}

type TransportRule struct {
	Rule
	Modes []string `json:"modes"`
}

type ActivityTypesRule struct {
	Rule
	Types []string `json:"types"`
}

type Rules struct {
	RequireTripBudget                   Rule              `json:"requireTripBudget"`
	MaxTripBudget                       MoneyRule         `json:"maxTripBudget"`
	MaxDailyBudget                      MoneyRule         `json:"maxDailyBudget"`
	MaxItemCost                         ItemCostRule      `json:"maxItemCost"`
	MaxAccommodationTotal               MoneyRule         `json:"maxAccommodationTotal"`
	MaxAccommodationPerNight            MoneyRule         `json:"maxAccommodationPerNight"`
	RequireCostSplitting                Rule              `json:"requireCostSplitting"`
	RequireAvailabilityForTicketedItems Rule              `json:"requireAvailabilityForTicketedItems"`
	MaxWalkingKmPerDay                  WalkingRule       `json:"maxWalkingKmPerDay"`
	NoLateActivitiesAfter               LateActivityRule  `json:"noLateActivitiesAfter"`
	RequiredRestTimePerDay              RestTimeRule      `json:"requiredRestTimePerDay"`
	PreferredTransportModes             TransportRule     `json:"preferredTransportModes"`
	DisallowedActivityTypes             ActivityTypesRule `json:"disallowedActivityTypes"`
}

type RulesDocument struct {
	SchemaVersion int   `json:"schemaVersion"`
	Rules         Rules `json:"rules"`
}

type Policy struct {
	ID               uuid.UUID     `json:"id"`
	WorkspaceID      uuid.UUID     `json:"workspaceId"`
	Name             string        `json:"name"`
	Description      *string       `json:"description"`
	Rules            RulesDocument `json:"rules"`
	Status           string        `json:"status"`
	CreatedByUserID  uuid.UUID     `json:"createdByUserId"`
	UpdatedByUserID  *uuid.UUID    `json:"updatedByUserId"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	ArchivedAt       *time.Time    `json:"archivedAt,omitempty"`
	ArchivedByUserID *uuid.UUID    `json:"archivedByUserId,omitempty"`
}

type UpsertInput struct {
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	Rules       RulesDocument `json:"rules"`
}

func (input *UpsertInput) UnmarshalJSON(data []byte) error {
	type alias UpsertInput
	var decoded alias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var envelope struct {
		Rules struct {
			Rules map[string]json.RawMessage `json:"rules"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	for _, key := range []string{
		"requireTripBudget",
		"maxTripBudget",
		"maxDailyBudget",
		"maxItemCost",
		"maxAccommodationTotal",
		"maxAccommodationPerNight",
		"requireCostSplitting",
		"requireAvailabilityForTicketedItems",
		"maxWalkingKmPerDay",
		"noLateActivitiesAfter",
		"requiredRestTimePerDay",
		"preferredTransportModes",
		"disallowedActivityTypes",
	} {
		raw, ok := envelope.Rules.Rules[key]
		if !ok {
			return fmt.Errorf("rules.rules.%s is required", key)
		}
		var base struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			return fmt.Errorf("decode rules.rules.%s: %w", key, err)
		}
		if base.Enabled == nil {
			return fmt.Errorf("rules.rules.%s.enabled is required", key)
		}
	}
	*input = UpsertInput(decoded)
	return nil
}

type GetResponse struct {
	Policy   *Policy        `json:"policy"`
	Defaults *RulesDocument `json:"defaults,omitempty"`
}

type EvaluationStatus string

const (
	EvaluationOK            EvaluationStatus = "ok"
	EvaluationInfo          EvaluationStatus = "info"
	EvaluationWarning       EvaluationStatus = "warning"
	EvaluationBlocking      EvaluationStatus = "blocking"
	EvaluationNotApplicable EvaluationStatus = "not_applicable"
)

type RuleResultStatus string

const (
	ResultPassed         RuleResultStatus = "passed"
	ResultViolation      RuleResultStatus = "violation"
	ResultWarningUnknown RuleResultStatus = "warning_unknown"
	ResultInfoUnknown    RuleResultStatus = "info_unknown"
)

type SuggestedAction struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	DayNumber *int   `json:"dayNumber,omitempty"`
	ItemIndex *int   `json:"itemIndex,omitempty"`
}

type AffectedItem struct {
	DayNumber *int     `json:"dayNumber,omitempty"`
	ItemIndex *int     `json:"itemIndex,omitempty"`
	Name      string   `json:"name,omitempty"`
	Amount    *float64 `json:"amount,omitempty"`
	Currency  string   `json:"currency,omitempty"`
}

type EvaluationResult struct {
	RuleKey          string            `json:"ruleKey"`
	Status           RuleResultStatus  `json:"status"`
	Severity         Severity          `json:"severity"`
	Title            string            `json:"title"`
	Message          string            `json:"message"`
	Actual           any               `json:"actual,omitempty"`
	Expected         any               `json:"expected,omitempty"`
	AffectedItems    []AffectedItem    `json:"affectedItems"`
	SuggestedActions []SuggestedAction `json:"suggestedActions"`
}

type EvaluationSummary struct {
	RulesChecked  int `json:"rulesChecked"`
	PassedCount   int `json:"passedCount"`
	InfoCount     int `json:"infoCount"`
	WarningCount  int `json:"warningCount"`
	BlockingCount int `json:"blockingCount"`
}

type Evaluation struct {
	TripID              uuid.UUID          `json:"tripId"`
	WorkspaceID         *uuid.UUID         `json:"workspaceId"`
	PolicyID            *uuid.UUID         `json:"policyId"`
	Status              EvaluationStatus   `json:"status"`
	GeneratedAt         time.Time          `json:"generatedAt"`
	Summary             EvaluationSummary  `json:"summary"`
	Results             []EvaluationResult `json:"results"`
	Warnings            []string           `json:"warnings"`
	NotApplicableReason *string            `json:"notApplicableReason"`
}

type AIConstraints struct {
	Summary string          `json:"summary"`
	Rules   json.RawMessage `json:"rules"`
}
