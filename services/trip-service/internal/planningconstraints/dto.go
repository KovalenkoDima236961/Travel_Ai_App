package planningconstraints

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

type PreviewRequest struct {
	Source                     Source          `json:"source"`
	TripID                     *uuid.UUID      `json:"tripId,omitempty"`
	WorkspaceID                *uuid.UUID      `json:"workspaceId,omitempty"`
	Request                    RequestOverride `json:"request"`
	IncludePreviousTripSignals *bool           `json:"includePreviousTripSignals,omitempty"`
	IncludeWorkspacePolicy     *bool           `json:"includeWorkspacePolicy,omitempty"`
	IncludeRoute               *bool           `json:"includeRoute,omitempty"`
	IncludeTripState           *bool           `json:"includeTripState,omitempty"`
}

type PreviewResponse struct {
	Constraints PlanningConstraints `json:"constraints"`
	Summary     Summary             `json:"summary"`
	Warnings    []Issue             `json:"warnings"`
	Blockers    []Issue             `json:"blockers"`
}

type RequestOverride struct {
	TripType        string                     `json:"tripType,omitempty"`
	Destination     string                     `json:"destination,omitempty"`
	OutputLanguage  string                     `json:"outputLanguage,omitempty"`
	Language        string                     `json:"language,omitempty"`
	StartDate       string                     `json:"startDate,omitempty"`
	EndDate         string                     `json:"endDate,omitempty"`
	DurationDays    *int                       `json:"durationDays,omitempty"`
	DateFlexibility string                     `json:"dateFlexibility,omitempty"`
	Budget          *BudgetOverride            `json:"budget,omitempty"`
	Travelers       *TravelerOverride          `json:"travelers,omitempty"`
	Pace            string                     `json:"pace,omitempty"`
	Walking         *WalkingOverride           `json:"walking,omitempty"`
	Transport       *TransportOverride         `json:"transport,omitempty"`
	Route           *aggregate.TripRoute       `json:"route,omitempty"`
	TripStyles      []string                   `json:"tripStyles,omitempty"`
	Accommodation   *AccommodationOverride     `json:"accommodation,omitempty"`
	Interests       []string                   `json:"interests,omitempty"`
	Avoid           []string                   `json:"avoid,omitempty"`
	MustHave        []string                   `json:"mustHave,omitempty"`
	Accessibility   *Accessibility             `json:"accessibility,omitempty"`
	Food            *Food                      `json:"food,omitempty"`
	Prompt          *Prompt                    `json:"prompt,omitempty"`
	Raw             map[string]json.RawMessage `json:"-"`
}

type BudgetOverride struct {
	Amount     *float64 `json:"amount,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	Strictness string   `json:"strictness,omitempty"`
}

type TravelerOverride struct {
	Count *int32 `json:"count,omitempty"`
	Type  string `json:"type,omitempty"`
}

type WalkingOverride struct {
	MaxKmPerDay    *float64 `json:"maxKmPerDay,omitempty"`
	AllowLongHikes *bool    `json:"allowLongHikes,omitempty"`
}

type TransportOverride struct {
	PreferredModes         []string `json:"preferredModes,omitempty"`
	AllowedModes           []string `json:"allowedModes,omitempty"`
	AvoidModes             []string `json:"avoidModes,omitempty"`
	DisallowedModes        []string `json:"disallowedModes,omitempty"`
	CarAvailable           *bool    `json:"carAvailable,omitempty"`
	MaxTransferHoursPerDay *int     `json:"maxTransferHoursPerDay,omitempty"`
}

type AccommodationOverride struct {
	PreferredTypes []string `json:"preferredTypes,omitempty"`
	AvoidTypes     []string `json:"avoidTypes,omitempty"`
	CampingAllowed *bool    `json:"campingAllowed,omitempty"`
}
