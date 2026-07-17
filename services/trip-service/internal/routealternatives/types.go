package routealternatives

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
)

const (
	SourcePreTrip             = "pre_trip"
	SourceExistingTrip        = "existing_trip"
	SourceDiscoveryRefinement = "discovery_refinement"
	SourceRouteRefinement     = "route_refinement"

	StatusCompleted   = "completed"
	StatusFailed      = "failed"
	StatusCreatedTrip = "created_trip"
	StatusApplied     = "applied"
	StatusArchived    = "archived"
)

type BudgetEstimate struct {
	Amount     *float64 `json:"amount,omitempty"`
	Currency   string   `json:"currency"`
	Confidence string   `json:"confidence,omitempty"`
}

type TransportInput struct {
	PreferredModes         []string `json:"preferredModes,omitempty"`
	AvoidModes             []string `json:"avoidModes,omitempty"`
	CarAvailable           bool     `json:"carAvailable"`
	MaxTransferHoursPerDay *int     `json:"maxTransferHoursPerDay,omitempty"`
}

type SuggestInput struct {
	Origin          *aggregate.RoutePlace `json:"origin,omitempty"`
	Prompt          string                `json:"prompt,omitempty"`
	DurationDays    int                   `json:"durationDays,omitempty"`
	StartDate       string                `json:"startDate,omitempty"`
	Budget          *BudgetEstimate       `json:"budget,omitempty"`
	Travelers       int32                 `json:"travelers,omitempty"`
	WorkspaceID     *uuid.UUID            `json:"workspaceId,omitempty"`
	Transport       TransportInput        `json:"transport,omitempty"`
	TripStyles      []string              `json:"tripStyles,omitempty"`
	OutputLanguage  string                `json:"outputLanguage,omitempty"`
	SuggestionCount int                   `json:"suggestionCount,omitempty"`
}

type ExistingTripSuggestInput struct {
	Prompt                    string `json:"prompt,omitempty"`
	SuggestionCount           int    `json:"suggestionCount,omitempty"`
	UseCurrentRouteAsBaseline bool   `json:"useCurrentRouteAsBaseline"`
	OutputLanguage            string `json:"outputLanguage,omitempty"`
}

type RefineInput struct {
	Instruction           string `json:"instruction"`
	SelectedAlternativeID string `json:"selectedAlternativeId,omitempty"`
}

type CreateTripInput struct {
	Title                 string          `json:"title"`
	StartDate             string          `json:"startDate,omitempty"`
	Budget                *BudgetEstimate `json:"budget,omitempty"`
	Travelers             *int32          `json:"travelers,omitempty"`
	WorkspaceID           *uuid.UUID      `json:"workspaceId,omitempty"`
	AutoGenerateItinerary bool            `json:"autoGenerateItinerary"`
}

type ApplyInput struct {
	ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
	RegenerateItinerary       bool `json:"regenerateItinerary"`
}

type CreatePollInput struct {
	Title          string   `json:"title"`
	AlternativeIDs []string `json:"alternativeIds"`
}

type AIRequest struct {
	Origin              *aggregate.RoutePlace                    `json:"origin,omitempty"`
	Prompt              string                                   `json:"prompt,omitempty"`
	DurationDays        int                                      `json:"durationDays,omitempty"`
	StartDate           string                                   `json:"startDate,omitempty"`
	Budget              *BudgetEstimate                          `json:"budget,omitempty"`
	Travelers           int32                                    `json:"travelers"`
	OutputLanguage      string                                   `json:"outputLanguage"`
	PlanningConstraints *planningconstraints.PlanningConstraints `json:"planningConstraints,omitempty"`
	CurrentRoute        *aggregate.TripRoute                     `json:"currentRoute,omitempty"`
	Refinement          Refinement                               `json:"refinement"`
	SuggestionCount     int                                      `json:"suggestionCount"`
}

type Refinement struct {
	PreviousAlternatives  []Alternative `json:"previousAlternatives"`
	Instruction           string        `json:"instruction,omitempty"`
	SelectedAlternativeID string        `json:"selectedAlternativeId,omitempty"`
}

type Response struct {
	SessionTitle      string            `json:"sessionTitle"`
	Alternatives      []Alternative     `json:"alternatives"`
	ComparisonSummary ComparisonSummary `json:"comparisonSummary"`
	FollowUpQuestions []string          `json:"followUpQuestions"`
	Warnings          []string          `json:"warnings"`
}

type Alternative struct {
	ID                       string              `json:"id"`
	Title                    string              `json:"title"`
	Summary                  string              `json:"summary"`
	Route                    aggregate.TripRoute `json:"route"`
	Scores                   Scores              `json:"scores"`
	EstimatedBudget          *BudgetEstimate     `json:"estimatedBudget,omitempty"`
	EstimatedTransferMinutes *int                `json:"estimatedTransferMinutes,omitempty"`
	EstimatedTransferCost    *BudgetEstimate     `json:"estimatedTransferCost,omitempty"`
	Difficulty               string              `json:"difficulty"`
	BestFor                  []string            `json:"bestFor"`
	Pros                     []string            `json:"pros"`
	Cons                     []string            `json:"cons"`
	Warnings                 []string            `json:"warnings"`
	SuggestedItineraryPrompt string              `json:"suggestedItineraryPrompt,omitempty"`
	PersonalizationFit       *PersonalizationFit `json:"personalizationFit,omitempty"`
}

type PersonalizationFit struct {
	Score    int      `json:"score"`
	Reasons  []string `json:"reasons"`
	Concerns []string `json:"concerns"`
}

type Scores struct {
	OverallFit          int `json:"overallFit"`
	BudgetFit           int `json:"budgetFit"`
	TimeEfficiency      int `json:"timeEfficiency"`
	Relaxation          int `json:"relaxation"`
	Nature              int `json:"nature"`
	Culture             int `json:"culture"`
	TransportSimplicity int `json:"transportSimplicity"`
	PolicyCompliance    int `json:"policyCompliance"`
}

type ComparisonSummary struct {
	CheapestAlternativeID    string `json:"cheapestAlternativeId,omitempty"`
	MostRelaxedAlternativeID string `json:"mostRelaxedAlternativeId,omitempty"`
	BestNatureAlternativeID  string `json:"bestNatureAlternativeId,omitempty"`
	BestOverallAlternativeID string `json:"bestOverallAlternativeId,omitempty"`
}

type Session struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	TripID                *uuid.UUID
	WorkspaceID           *uuid.UUID
	Source                string
	Prompt                string
	OutputLanguage        string
	Status                string
	RequestJSON           json.RawMessage
	ResponseJSON          json.RawMessage
	SelectedAlternativeID string
	CreatedTripID         *uuid.UUID
	AppliedToTripID       *uuid.UUID
	ParentSessionID       *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type SessionView struct {
	ID                    uuid.UUID         `json:"id"`
	UserID                uuid.UUID         `json:"userId"`
	TripID                *uuid.UUID        `json:"tripId,omitempty"`
	WorkspaceID           *uuid.UUID        `json:"workspaceId,omitempty"`
	Source                string            `json:"source"`
	Prompt                string            `json:"prompt,omitempty"`
	OutputLanguage        string            `json:"outputLanguage"`
	Status                string            `json:"status"`
	SelectedAlternativeID string            `json:"selectedAlternativeId,omitempty"`
	CreatedTripID         *uuid.UUID        `json:"createdTripId,omitempty"`
	AppliedToTripID       *uuid.UUID        `json:"appliedToTripId,omitempty"`
	ParentSessionID       *uuid.UUID        `json:"parentSessionId,omitempty"`
	SessionTitle          string            `json:"sessionTitle"`
	Alternatives          []Alternative     `json:"alternatives"`
	ComparisonSummary     ComparisonSummary `json:"comparisonSummary"`
	FollowUpQuestions     []string          `json:"followUpQuestions"`
	Warnings              []string          `json:"warnings"`
	CreatedAt             time.Time         `json:"createdAt"`
	UpdatedAt             time.Time         `json:"updatedAt"`
}

type ListSessionsResult struct {
	Items []SessionView `json:"items"`
	Limit int           `json:"limit"`
}

func NewSessionView(session *Session) (SessionView, error) {
	var response Response
	if len(session.ResponseJSON) > 0 {
		if err := json.Unmarshal(session.ResponseJSON, &response); err != nil {
			return SessionView{}, err
		}
	}
	return SessionView{
		ID:                    session.ID,
		UserID:                session.UserID,
		TripID:                session.TripID,
		WorkspaceID:           session.WorkspaceID,
		Source:                session.Source,
		Prompt:                session.Prompt,
		OutputLanguage:        session.OutputLanguage,
		Status:                session.Status,
		SelectedAlternativeID: session.SelectedAlternativeID,
		CreatedTripID:         session.CreatedTripID,
		AppliedToTripID:       session.AppliedToTripID,
		ParentSessionID:       session.ParentSessionID,
		SessionTitle:          response.SessionTitle,
		Alternatives:          response.Alternatives,
		ComparisonSummary:     response.ComparisonSummary,
		FollowUpQuestions:     response.FollowUpQuestions,
		Warnings:              response.Warnings,
		CreatedAt:             session.CreatedAt,
		UpdatedAt:             session.UpdatedAt,
	}, nil
}

func FindAlternative(response Response, alternativeID string) (*Alternative, bool) {
	for i := range response.Alternatives {
		if response.Alternatives[i].ID == alternativeID {
			return &response.Alternatives[i], true
		}
	}
	return nil, false
}

func DecodeResponse(raw json.RawMessage) (Response, error) {
	var response Response
	if err := json.Unmarshal(raw, &response); err != nil {
		return Response{}, err
	}
	return response, nil
}

func NormalizeAndScore(response *Response, budget *BudgetEstimate, constraints *planningconstraints.PlanningConstraints) {
	if response == nil {
		return
	}
	seen := map[string]int{}
	for index := range response.Alternatives {
		alt := &response.Alternatives[index]
		alt.ID = normalizeID(alt.ID, alt.Title, index)
		if count := seen[alt.ID]; count > 0 {
			alt.ID = alt.ID + "-" + string(rune('a'+count))
		}
		seen[alt.ID]++
		NormalizeAlternative(alt, budget, constraints)
	}
	response.ComparisonSummary = BuildComparison(response.Alternatives)
	if len(response.Warnings) == 0 {
		response.Warnings = []string{"Route estimates are approximate and do not include live ticket prices."}
	}
}

func NormalizeAlternative(alt *Alternative, budget *BudgetEstimate, constraints *planningconstraints.PlanningConstraints) {
	if alt == nil {
		return
	}
	ensureRouteLegs(alt)
	totalMinutes := 0
	totalCost := 0.0
	for index := range alt.Route.Legs {
		leg := &alt.Route.Legs[index]
		if leg.Mode == "" {
			leg.Mode = preferredMode(alt.Route.Preferences)
		}
		if leg.EstimatedDurationMinutes == nil {
			value := fallbackDurationMinutes(leg.Mode, leg.EstimatedDistanceKm)
			leg.EstimatedDurationMinutes = &value
			leg.Warnings = appendUnique(leg.Warnings, "Transfer duration is an approximate fallback estimate.")
		}
		if leg.EstimatedDistanceKm == nil {
			value := float64(*leg.EstimatedDurationMinutes) * 1.4
			leg.EstimatedDistanceKm = &value
		}
		if leg.EstimatedCost == nil {
			amount := fallbackCost(leg.Mode, leg.EstimatedDistanceKm)
			leg.EstimatedCost = &aggregate.EstimatedCost{
				Amount:     &amount,
				Currency:   estimateCurrency(alt, budget),
				Category:   "transport",
				Confidence: "low",
				Source:     "mock",
				Note:       "Approximate fallback route estimate.",
			}
			leg.Warnings = appendUnique(leg.Warnings, "Transfer cost is an approximate fallback estimate.")
		}
		totalMinutes += *leg.EstimatedDurationMinutes
		if leg.EstimatedCost != nil && leg.EstimatedCost.Amount != nil {
			totalCost += *leg.EstimatedCost.Amount
		}
	}
	alt.EstimatedTransferMinutes = &totalMinutes
	alt.EstimatedTransferCost = &BudgetEstimate{
		Amount:     roundPtr(totalCost),
		Currency:   estimateCurrency(alt, budget),
		Confidence: "medium",
	}
	if alt.EstimatedBudget == nil || alt.EstimatedBudget.Amount == nil {
		base := totalCost + float64(maxInt(1, len(alt.Route.Stops)))*120
		alt.EstimatedBudget = &BudgetEstimate{
			Amount:     roundPtr(base),
			Currency:   estimateCurrency(alt, budget),
			Confidence: "low",
		}
	}
	alt.Scores = scoreAlternative(*alt, budget, constraints)
	alt.Difficulty = difficultyFor(alt, constraints)
	alt.Pros = capStrings(alt.Pros, 8)
	alt.Cons = capStrings(alt.Cons, 8)
	alt.Warnings = capStrings(alt.Warnings, 8)
	if len(alt.Warnings) == 0 {
		alt.Warnings = []string{"Estimates are approximate and not live schedules or prices."}
	}
}

func BuildComparison(alternatives []Alternative) ComparisonSummary {
	var out ComparisonSummary
	if len(alternatives) == 0 {
		return out
	}
	cheapest := 0
	relaxed := 0
	nature := 0
	overall := 0
	for i := range alternatives {
		if amountOf(alternatives[i].EstimatedBudget) < amountOf(alternatives[cheapest].EstimatedBudget) {
			cheapest = i
		}
		if alternatives[i].Scores.Relaxation > alternatives[relaxed].Scores.Relaxation {
			relaxed = i
		}
		if alternatives[i].Scores.Nature > alternatives[nature].Scores.Nature {
			nature = i
		}
		if alternatives[i].Scores.OverallFit > alternatives[overall].Scores.OverallFit {
			overall = i
		}
	}
	out.CheapestAlternativeID = alternatives[cheapest].ID
	out.MostRelaxedAlternativeID = alternatives[relaxed].ID
	out.BestNatureAlternativeID = alternatives[nature].ID
	out.BestOverallAlternativeID = alternatives[overall].ID
	return out
}

func scoreAlternative(alt Alternative, budget *BudgetEstimate, constraints *planningconstraints.PlanningConstraints) Scores {
	budgetFit := budgetFitScore(alt.EstimatedBudget, budget)
	relaxation := relaxationScore(alt)
	timeEfficiency := clamp(100 - transferMinutes(alt)/20 - len(alt.Route.Stops)*3)
	transport := transportSimplicityScore(alt, constraints)
	policy := policyComplianceScore(constraints)
	nature := clamp(defaultScore(alt.Scores.Nature, styleScore(alt, "nature", "hiking", "camping")))
	culture := clamp(defaultScore(alt.Scores.Culture, styleScore(alt, "culture", "city_break")))
	overall := clamp(int(math.Round(
		float64(policy)*0.25 +
			float64(budgetFit)*0.20 +
			float64(relaxation+timeEfficiency)/2*0.20 +
			float64(transport)*0.20 +
			float64(defaultScore(alt.Scores.OverallFit, 75))*0.15,
	)))
	return Scores{
		OverallFit:          overall,
		BudgetFit:           budgetFit,
		TimeEfficiency:      timeEfficiency,
		Relaxation:          relaxation,
		Nature:              nature,
		Culture:             culture,
		TransportSimplicity: transport,
		PolicyCompliance:    policy,
	}
}

func budgetFitScore(estimated, budget *BudgetEstimate) int {
	if budget == nil || budget.Amount == nil || *budget.Amount <= 0 {
		return 70
	}
	if estimated == nil || estimated.Amount == nil {
		return 60
	}
	ratio := *estimated.Amount / *budget.Amount
	switch {
	case ratio <= 1:
		return 95
	case ratio <= 1.1:
		return 70
	case ratio <= 1.3:
		return 50
	default:
		return 25
	}
}

func relaxationScore(alt Alternative) int {
	minutes := transferMinutes(alt)
	stops := len(alt.Route.Stops)
	return clamp(100 - stops*8 - minutes/45)
}

func transportSimplicityScore(alt Alternative, constraints *planningconstraints.PlanningConstraints) int {
	modes := map[string]struct{}{}
	score := 92
	avoid := map[string]struct{}{}
	preferred := map[string]struct{}{}
	if constraints != nil {
		for _, mode := range constraints.Transport.AvoidModes {
			avoid[normalizeToken(mode)] = struct{}{}
		}
		for _, mode := range constraints.Transport.DisallowedModes {
			avoid[normalizeToken(mode)] = struct{}{}
		}
		for _, mode := range constraints.Transport.PreferredModes {
			preferred[normalizeToken(mode)] = struct{}{}
		}
	}
	for _, leg := range alt.Route.Legs {
		mode := normalizeToken(leg.Mode)
		modes[mode] = struct{}{}
		if _, ok := avoid[mode]; ok {
			score -= 35
		}
		if len(preferred) > 0 {
			if _, ok := preferred[mode]; ok {
				score += 2
			} else {
				score -= 8
			}
		}
		if mode == aggregate.TransportModeFlight || mode == aggregate.TransportModeFerry || mode == aggregate.TransportModeBoat {
			score -= 5
		}
	}
	score -= maxInt(0, len(modes)-1) * 8
	return clamp(score)
}

func policyComplianceScore(constraints *planningconstraints.PlanningConstraints) int {
	if constraints == nil {
		return 100
	}
	if len(constraints.Blockers) > 0 {
		return 35
	}
	if len(constraints.Warnings) > 0 {
		return 82
	}
	return 100
}

func difficultyFor(alt *Alternative, constraints *planningconstraints.PlanningConstraints) string {
	days := len(alt.Route.Stops)
	if constraints != nil && constraints.Dates.DurationDays > 0 {
		days = constraints.Dates.DurationDays
	}
	if days < 1 {
		days = 1
	}
	minutesPerDay := float64(transferMinutes(*alt)) / float64(days)
	stopsPerDay := float64(len(alt.Route.Stops)) / float64(days)
	switch {
	case stopsPerDay <= 0.4 && minutesPerDay <= 90:
		return "relaxed"
	case stopsPerDay > 0.8 || minutesPerDay > 180:
		return "rushed"
	case stopsPerDay > 0.6 || minutesPerDay > 130:
		return "intense"
	default:
		return "balanced"
	}
}

func ensureRouteLegs(alt *Alternative) {
	if len(alt.Route.Legs) >= len(alt.Route.Stops) {
		return
	}
	existing := map[string]struct{}{}
	for _, leg := range alt.Route.Legs {
		existing[leg.ToStopID] = struct{}{}
	}
	mode := preferredMode(alt.Route.Preferences)
	for i, stop := range alt.Route.Stops {
		if _, ok := existing[stop.ID]; ok {
			continue
		}
		fromID := "origin"
		fromName := "Origin"
		if alt.Route.Origin != nil && strings.TrimSpace(alt.Route.Origin.Name) != "" {
			fromName = alt.Route.Origin.Name
		}
		if i > 0 {
			fromID = alt.Route.Stops[i-1].ID
			fromName = alt.Route.Stops[i-1].Destination
		}
		alt.Route.Legs = append(alt.Route.Legs, aggregate.RouteLeg{
			ID:         "leg_" + intString(len(alt.Route.Legs)+1),
			FromStopID: fromID,
			ToStopID:   stop.ID,
			FromName:   fromName,
			ToName:     stop.Destination,
			Mode:       mode,
			Warnings:   []string{"Leg was added from route stop order using fallback estimates."},
		})
	}
}

func preferredMode(prefs aggregate.RoutePreferences) string {
	if len(prefs.PreferredModes) > 0 {
		return normalizeToken(prefs.PreferredModes[0])
	}
	return aggregate.TransportModeTrain
}

func fallbackDurationMinutes(mode string, distance *float64) int {
	km := 120.0
	if distance != nil && *distance > 0 {
		km = *distance
	}
	speed := 75.0
	switch normalizeToken(mode) {
	case aggregate.TransportModeFlight:
		speed = 600
	case aggregate.TransportModeCar, aggregate.TransportModeRentalCar:
		speed = 80
	case aggregate.TransportModeBus:
		speed = 65
	case aggregate.TransportModeFerry, aggregate.TransportModeBoat:
		speed = 35
	case aggregate.TransportModeBike:
		speed = 18
	case aggregate.TransportModeWalk, aggregate.TransportModeHiking:
		speed = 5
	}
	return maxInt(15, int(math.Round(km/speed*60)))
}

func fallbackCost(mode string, distance *float64) float64 {
	km := 100.0
	if distance != nil && *distance > 0 {
		km = *distance
	}
	switch normalizeToken(mode) {
	case aggregate.TransportModeBus:
		return round(km * 0.08)
	case aggregate.TransportModeFlight:
		return round(math.Max(50, km*0.15))
	case aggregate.TransportModeFerry, aggregate.TransportModeBoat:
		return round(km * 0.20)
	case aggregate.TransportModeCar, aggregate.TransportModeRentalCar:
		return round(km * 0.18)
	case aggregate.TransportModeBike, aggregate.TransportModeHiking, aggregate.TransportModeWalk:
		return 0
	default:
		return round(km * 0.12)
	}
}

func estimateCurrency(alt *Alternative, budget *BudgetEstimate) string {
	if alt.EstimatedBudget != nil && strings.TrimSpace(alt.EstimatedBudget.Currency) != "" {
		return strings.ToUpper(strings.TrimSpace(alt.EstimatedBudget.Currency))
	}
	if budget != nil && strings.TrimSpace(budget.Currency) != "" {
		return strings.ToUpper(strings.TrimSpace(budget.Currency))
	}
	return "EUR"
}

func transferMinutes(alt Alternative) int {
	if alt.EstimatedTransferMinutes != nil {
		return *alt.EstimatedTransferMinutes
	}
	total := 0
	for _, leg := range alt.Route.Legs {
		if leg.EstimatedDurationMinutes != nil {
			total += *leg.EstimatedDurationMinutes
		}
	}
	return total
}

func styleScore(alt Alternative, values ...string) int {
	text := strings.ToLower(strings.Join(append(append([]string{}, alt.BestFor...), alt.Route.Preferences.TripStyles...), " "))
	for _, value := range values {
		if strings.Contains(text, value) {
			return 88
		}
	}
	return 68
}

func defaultScore(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func amountOf(estimate *BudgetEstimate) float64 {
	if estimate == nil || estimate.Amount == nil {
		return math.MaxFloat64
	}
	return *estimate.Amount
}

func roundPtr(value float64) *float64 {
	out := round(value)
	return &out
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}

func clamp(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func capStrings(values []string, max int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
		if len(out) >= max {
			break
		}
	}
	return out
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func normalizeID(raw, title string, index int) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = title
	}
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	normalized := strings.Trim(b.String(), "-")
	if normalized == "" {
		return "route-option-" + intString(index+1)
	}
	return normalized
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
