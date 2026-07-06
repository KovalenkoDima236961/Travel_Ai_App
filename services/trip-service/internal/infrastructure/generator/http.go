package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/templateadaptation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

const maxAIPlanningErrorBodyBytes = 4 * 1024

// AIPlanningHTTPGenerator calls AI Planning Service v1 over HTTP.
type AIPlanningHTTPGenerator struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

type aiPlanningGenerateRequest struct {
	TripID          string                          `json:"tripId"`
	Destination     string                          `json:"destination"`
	StartDate       *string                         `json:"startDate,omitempty"`
	Days            int32                           `json:"days"`
	BudgetAmount    *float64                        `json:"budgetAmount,omitempty"`
	BudgetCurrency  string                          `json:"budgetCurrency"`
	Travelers       int32                           `json:"travelers"`
	Interests       []string                        `json:"interests"`
	Pace            string                          `json:"pace"`
	UserProfile     *usercontext.UserProfile        `json:"userProfile,omitempty"`
	UserPreferences *usercontext.UserPreferences    `json:"userPreferences,omitempty"`
	WeatherForecast *weathercontext.WeatherForecast `json:"weatherForecast,omitempty"`
	Accommodation   *aggregate.Accommodation        `json:"accommodation,omitempty"`
}

type aiPlanningTripRequest struct {
	ID             string   `json:"id"`
	Destination    string   `json:"destination"`
	StartDate      *string  `json:"startDate,omitempty"`
	Days           int32    `json:"days"`
	BudgetAmount   *float64 `json:"budgetAmount,omitempty"`
	BudgetCurrency string   `json:"budgetCurrency"`
	Travelers      int32    `json:"travelers"`
	Interests      []string `json:"interests"`
	Pace           string   `json:"pace"`
}

type aiPlanningRegenerateDayRequest struct {
	Trip             aiPlanningTripRequest           `json:"trip"`
	CurrentItinerary aggregate.Itinerary             `json:"currentItinerary"`
	DayNumber        int                             `json:"dayNumber"`
	Instruction      string                          `json:"instruction,omitempty"`
	UserProfile      *usercontext.UserProfile        `json:"userProfile,omitempty"`
	UserPreferences  *usercontext.UserPreferences    `json:"userPreferences,omitempty"`
	WeatherForecast  *weathercontext.WeatherForecast `json:"weatherForecast,omitempty"`
	Accommodation    *aggregate.Accommodation        `json:"accommodation,omitempty"`
}

type aiPlanningRegenerateItemRequest struct {
	Trip             aiPlanningTripRequest           `json:"trip"`
	CurrentItinerary aggregate.Itinerary             `json:"currentItinerary"`
	DayNumber        int                             `json:"dayNumber"`
	ItemIndex        int                             `json:"itemIndex"`
	Instruction      string                          `json:"instruction,omitempty"`
	UserProfile      *usercontext.UserProfile        `json:"userProfile,omitempty"`
	UserPreferences  *usercontext.UserPreferences    `json:"userPreferences,omitempty"`
	WeatherForecast  *weathercontext.WeatherForecast `json:"weatherForecast,omitempty"`
	Accommodation    *aggregate.Accommodation        `json:"accommodation,omitempty"`
}

type aiPlanningRegenerateDayResponse struct {
	Day aggregate.ItineraryDay `json:"day"`
}

type aiPlanningRegenerateItemResponse struct {
	Item aggregate.ItineraryItem `json:"item"`
}

type aiPlanningOptimizeBudgetDayRequest struct {
	Trip             aiPlanningTripRequest            `json:"trip"`
	CurrentItinerary aggregate.Itinerary              `json:"currentItinerary"`
	DayNumber        int                              `json:"dayNumber"`
	CurrentDay       aggregate.ItineraryDay           `json:"currentDay"`
	BudgetContext    budgetoptimization.BudgetContext `json:"budgetContext"`
	Constraints      budgetoptimization.Constraints   `json:"constraints"`
	Instruction      string                           `json:"instruction,omitempty"`
	UserProfile      *usercontext.UserProfile         `json:"userProfile,omitempty"`
	UserPreferences  *usercontext.UserPreferences     `json:"userPreferences,omitempty"`
	WeatherForecast  *weathercontext.WeatherForecast  `json:"weatherForecast,omitempty"`
	Accommodation    *aggregate.Accommodation         `json:"accommodation,omitempty"`
}

type aiPlanningAdaptTemplateRequest struct {
	Template    templateadaptation.Template    `json:"template"`
	Target      aiPlanningAdaptTarget          `json:"target"`
	Constraints templateadaptation.Constraints `json:"constraints"`
	Context     *aiPlanningAdaptContext        `json:"context,omitempty"`
}

type aiPlanningAdaptTarget struct {
	Destination  string                    `json:"destination"`
	StartDate    string                    `json:"startDate"`
	DurationDays int                       `json:"durationDays"`
	Budget       *templateadaptation.Money `json:"budget,omitempty"`
	Travelers    int                       `json:"travelers"`
	Pace         string                    `json:"pace"`
	Interests    []string                  `json:"interests"`
	Avoid        []string                  `json:"avoid"`
}

type aiPlanningAdaptContext struct {
	UserProfile     *usercontext.UserProfile     `json:"userProfile,omitempty"`
	UserPreferences *usercontext.UserPreferences `json:"userPreferences,omitempty"`
}

type aiPlanningAdaptResponse struct {
	Itinerary         aiPlanningAdaptedItinerary  `json:"itinerary"`
	AdaptationSummary aiPlanningAdaptationSummary `json:"adaptationSummary"`
}

type aiPlanningAdaptedItinerary struct {
	Title       string                 `json:"title"`
	Destination string                 `json:"destination"`
	StartDate   string                 `json:"startDate"`
	Days        []aiPlanningAdaptedDay `json:"days"`
}

type aiPlanningAdaptedDay struct {
	Date  string                  `json:"date"`
	Title string                  `json:"title"`
	Items []aiPlanningAdaptedItem `json:"items"`
}

type aiPlanningAdaptedItem struct {
	Name          string                            `json:"name"`
	Type          string                            `json:"type"`
	Description   string                            `json:"description"`
	Time          string                            `json:"time"`
	StartTime     string                            `json:"startTime"`
	EndTime       string                            `json:"endTime"`
	Place         *templateadaptation.TemplatePlace `json:"place"`
	EstimatedCost *aggregate.EstimatedCost          `json:"estimatedCost"`
	Notes         string                            `json:"notes"`
}

type aiPlanningAdaptationSummary struct {
	SourceDurationDays int      `json:"sourceDurationDays"`
	TargetDurationDays int      `json:"targetDurationDays"`
	PreservedStructure bool     `json:"preservedStructure"`
	ChangedDestination bool     `json:"changedDestination"`
	FallbackUsed       bool     `json:"fallbackUsed"`
	MajorChanges       []string `json:"majorChanges"`
	Warnings           []string `json:"warnings"`
}

// AdaptTemplate calls AI Planning Service /adapt-template and maps the adapted
// itinerary into the internal aggregate. The returned itinerary is still a
// draft: prices are estimates and availability is unchecked.
func (g *AIPlanningHTTPGenerator) AdaptTemplate(ctx context.Context, input templateadaptation.AdaptInput) (*templateadaptation.AdaptResult, error) {
	payload := aiPlanningAdaptTemplateRequest{
		Template: input.Template,
		Target: aiPlanningAdaptTarget{
			Destination:  input.Target.Destination,
			StartDate:    input.Target.StartDate,
			DurationDays: input.Target.DurationDays,
			Budget:       input.Target.Budget,
			Travelers:    input.Target.Travelers,
			Pace:         input.Target.Pace,
			Interests:    nonNilStrings(input.Target.Interests),
			Avoid:        nonNilStrings(input.Target.Avoid),
		},
		Constraints: input.Constraints,
	}
	if input.UserProfile != nil || input.UserPreferences != nil {
		payload.Context = &aiPlanningAdaptContext{
			UserProfile:     input.UserProfile,
			UserPreferences: input.UserPreferences,
		}
	}

	var result aiPlanningAdaptResponse
	if err := g.postJSON(ctx, input.TripID, "adapt-template", payload, &result); err != nil {
		return nil, err
	}
	if len(result.Itinerary.Days) == 0 {
		return nil, fmt.Errorf("ai planning service returned empty adapted itinerary days")
	}
	return mapAdaptResponse(result, input), nil
}

func mapAdaptResponse(result aiPlanningAdaptResponse, input templateadaptation.AdaptInput) *templateadaptation.AdaptResult {
	days := make([]aggregate.ItineraryDay, 0, len(result.Itinerary.Days))
	for index, day := range result.Itinerary.Days {
		items := make([]aggregate.ItineraryItem, 0, len(day.Items))
		for _, item := range day.Items {
			items = append(items, mapAdaptedItem(item))
		}
		title := strings.TrimSpace(day.Title)
		if title == "" {
			title = fmt.Sprintf("Day %d", index+1)
		}
		days = append(days, aggregate.ItineraryDay{
			Day:   index + 1,
			Title: title,
			Items: items,
		})
	}

	currency := defaultCurrency
	if input.Target.Budget != nil && input.Target.Budget.Currency != "" {
		currency = input.Target.Budget.Currency
	}
	itinerary := aggregate.Itinerary{
		Destination: input.Target.Destination,
		Summary:     strings.TrimSpace(result.Itinerary.Title),
		Travelers:   int32(input.Target.Travelers),
		Pace:        input.Target.Pace,
		Currency:    currency,
		Days:        days,
		GeneratedAt: time.Now().UTC(),
		Source:      "ai_template_adaptation",
	}
	summary := templateadaptation.Summary{
		SourceDurationDays: result.AdaptationSummary.SourceDurationDays,
		TargetDurationDays: result.AdaptationSummary.TargetDurationDays,
		PreservedStructure: result.AdaptationSummary.PreservedStructure,
		ChangedDestination: result.AdaptationSummary.ChangedDestination,
		FallbackUsed:       result.AdaptationSummary.FallbackUsed,
		MajorChanges:       nonNilStrings(result.AdaptationSummary.MajorChanges),
		Warnings:           nonNilStrings(result.AdaptationSummary.Warnings),
	}
	return &templateadaptation.AdaptResult{Itinerary: itinerary, Summary: summary}
}

func mapAdaptedItem(item aiPlanningAdaptedItem) aggregate.ItineraryItem {
	timeValue := strings.TrimSpace(item.Time)
	if timeValue == "" {
		timeValue = strings.TrimSpace(item.StartTime)
		if end := strings.TrimSpace(item.EndTime); end != "" {
			timeValue = strings.TrimSpace(timeValue + " - " + end)
		}
	}
	note := strings.TrimSpace(item.Notes)
	if note == "" {
		note = strings.TrimSpace(item.Description)
	}
	var place *aggregate.PlaceRef
	if item.Place != nil && strings.TrimSpace(item.Place.Name) != "" {
		place = &aggregate.PlaceRef{
			Name:     strings.TrimSpace(item.Place.Name),
			Category: strings.TrimSpace(item.Place.Category),
		}
	}
	return aggregate.ItineraryItem{
		Time:          timeValue,
		Type:          strings.TrimSpace(item.Type),
		Name:          strings.TrimSpace(item.Name),
		Note:          note,
		EstimatedCost: item.EstimatedCost,
		Place:         place,
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// NewAIPlanningHTTPGenerator constructs an HTTP generator with a validated base
// URL and a caller-provided client.
func NewAIPlanningHTTPGenerator(baseURL string, client *http.Client, logger *zap.Logger) (*AIPlanningHTTPGenerator, error) {
	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("ai planning http client is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AIPlanningHTTPGenerator{
		baseURL: normalizedBaseURL,
		client:  observability.InstrumentHTTPClient(client),
		logger:  logger,
	}, nil
}

// Generate sends the trip to AI Planning Service v1 and returns the generated
// itinerary.
func (g *AIPlanningHTTPGenerator) Generate(ctx context.Context, input application.GenerateItineraryInput) (*aggregate.Itinerary, error) {
	trip := input.Trip
	payload := newAIPlanningGenerateRequest(input)

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		g.logger.Error("failed to encode ai planning request",
			zap.String("trip_id", trip.ID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("encode ai planning request: %w", err)
	}

	endpoint, err := url.JoinPath(g.baseURL, "generate-itinerary")
	if err != nil {
		return nil, fmt.Errorf("build ai planning endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("create ai planning request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		g.logger.Error("ai planning request failed",
			zap.String("trip_id", trip.ID.String()),
			zap.String("url", endpoint),
			zap.Error(err),
		)
		return nil, fmt.Errorf("call ai planning service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		limitedBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxAIPlanningErrorBodyBytes))
		if readErr != nil {
			g.logger.Error("failed to read ai planning error response",
				zap.String("trip_id", trip.ID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Error(readErr),
			)
			return nil, fmt.Errorf("ai planning service returned status %d and error body could not be read: %w", resp.StatusCode, readErr)
		}

		responseBody := strings.TrimSpace(string(limitedBody))
		err := fmt.Errorf("ai planning service returned status %d: %s", resp.StatusCode, responseBody)
		g.logger.Error("ai planning service returned non-2xx response",
			zap.String("trip_id", trip.ID.String()),
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", responseBody),
			zap.Error(err),
		)
		return nil, err
	}

	var itinerary aggregate.Itinerary
	if err := json.NewDecoder(resp.Body).Decode(&itinerary); err != nil {
		g.logger.Error("failed to decode ai planning response",
			zap.String("trip_id", trip.ID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("decode ai planning response: %w", err)
	}
	if len(itinerary.Days) == 0 {
		err := fmt.Errorf("ai planning service returned empty itinerary days")
		g.logger.Error("invalid ai planning response",
			zap.String("trip_id", trip.ID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	enrichItineraryDefaults(&itinerary, trip)
	return &itinerary, nil
}

// RegenerateDay calls AI Planning Service v1 to replace a single itinerary day.
func (g *AIPlanningHTTPGenerator) RegenerateDay(ctx context.Context, input application.RegenerateDayInput) (*aggregate.ItineraryDay, error) {
	trip := input.Trip
	payload := aiPlanningRegenerateDayRequest{
		Trip:             newAIPlanningTripRequest(trip, input.UserProfile),
		CurrentItinerary: input.CurrentItinerary,
		DayNumber:        input.DayNumber,
		Instruction:      input.Instruction,
		UserProfile:      input.UserProfile,
		UserPreferences:  input.UserPreferences,
		WeatherForecast:  input.WeatherForecast,
		Accommodation:    trip.Accommodation,
	}

	var result aiPlanningRegenerateDayResponse
	if err := g.postJSON(ctx, trip.ID, "regenerate-day", payload, &result); err != nil {
		return nil, err
	}
	return &result.Day, nil
}

// RegenerateItem calls AI Planning Service v1 to replace a single itinerary item.
func (g *AIPlanningHTTPGenerator) RegenerateItem(ctx context.Context, input application.RegenerateItemInput) (*aggregate.ItineraryItem, error) {
	trip := input.Trip
	payload := aiPlanningRegenerateItemRequest{
		Trip:             newAIPlanningTripRequest(trip, input.UserProfile),
		CurrentItinerary: input.CurrentItinerary,
		DayNumber:        input.DayNumber,
		ItemIndex:        input.ItemIndex,
		Instruction:      input.Instruction,
		UserProfile:      input.UserProfile,
		UserPreferences:  input.UserPreferences,
		WeatherForecast:  input.WeatherForecast,
		Accommodation:    trip.Accommodation,
	}

	var result aiPlanningRegenerateItemResponse
	if err := g.postJSON(ctx, trip.ID, "regenerate-item", payload, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// OptimizeBudgetDay calls AI Planning Service v1 to produce a reviewable cheaper-day proposal.
func (g *AIPlanningHTTPGenerator) OptimizeBudgetDay(ctx context.Context, input budgetoptimization.OptimizeDayInput) (*budgetoptimization.ProposalContent, error) {
	trip := input.Trip
	payload := aiPlanningOptimizeBudgetDayRequest{
		Trip:             newAIPlanningTripRequest(trip, input.UserProfile),
		CurrentItinerary: input.CurrentItinerary,
		DayNumber:        input.DayNumber,
		CurrentDay:       input.CurrentDay,
		BudgetContext:    input.BudgetContext,
		Constraints:      input.Constraints,
		Instruction:      input.Instruction,
		UserProfile:      input.UserProfile,
		UserPreferences:  input.UserPreferences,
		WeatherForecast:  input.WeatherForecast,
		Accommodation:    trip.Accommodation,
	}

	var result budgetoptimization.ProposalContent
	if err := g.postJSON(ctx, trip.ID, "optimize-budget/day", payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func newAIPlanningGenerateRequest(input application.GenerateItineraryInput) aiPlanningGenerateRequest {
	trip := input.Trip
	var startDate *string
	if trip.StartDate != nil {
		formatted := trip.StartDate.Format("2006-01-02")
		startDate = &formatted
	}

	currency := resolveRequestCurrency(trip.BudgetCurrency, input.UserProfile)
	pace := strings.TrimSpace(trip.Pace)
	if pace == "" {
		pace = defaultPace
	}
	interests := trip.Interests
	if interests == nil {
		interests = []string{}
	}

	return aiPlanningGenerateRequest{
		TripID:          trip.ID.String(),
		Destination:     trip.Destination,
		StartDate:       startDate,
		Days:            trip.Days,
		BudgetAmount:    trip.BudgetAmount,
		BudgetCurrency:  currency,
		Travelers:       trip.Travelers,
		Interests:       interests,
		Pace:            pace,
		UserProfile:     input.UserProfile,
		UserPreferences: input.UserPreferences,
		WeatherForecast: input.WeatherForecast,
		Accommodation:   trip.Accommodation,
	}
}

// resolveRequestCurrency chooses the currency sent to AI Planning Service: the
// trip budget currency, then the user's preferred currency, then the default.
func resolveRequestCurrency(budgetCurrency string, profile *usercontext.UserProfile) string {
	if c := strings.ToUpper(strings.TrimSpace(budgetCurrency)); c != "" {
		return c
	}
	if profile != nil {
		if c := strings.ToUpper(strings.TrimSpace(profile.PreferredCurrency)); c != "" {
			return c
		}
	}
	return defaultCurrency
}

func newAIPlanningTripRequest(trip entity.Trip, profile *usercontext.UserProfile) aiPlanningTripRequest {
	var startDate *string
	if trip.StartDate != nil {
		formatted := trip.StartDate.Format("2006-01-02")
		startDate = &formatted
	}

	currency := resolveRequestCurrency(trip.BudgetCurrency, profile)
	pace := strings.TrimSpace(trip.Pace)
	if pace == "" {
		pace = defaultPace
	}
	interests := trip.Interests
	if interests == nil {
		interests = []string{}
	}

	return aiPlanningTripRequest{
		ID:             trip.ID.String(),
		Destination:    trip.Destination,
		StartDate:      startDate,
		Days:           trip.Days,
		BudgetAmount:   trip.BudgetAmount,
		BudgetCurrency: currency,
		Travelers:      trip.Travelers,
		Interests:      interests,
		Pace:           pace,
	}
}

func (g *AIPlanningHTTPGenerator) postJSON(ctx context.Context, tripID uuid.UUID, path string, payload, out any) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		g.logger.Error("failed to encode ai planning request",
			zap.String("trip_id", tripID.String()),
			zap.String("path", path),
			zap.Error(err),
		)
		return fmt.Errorf("encode ai planning request: %w", err)
	}

	endpoint, err := url.JoinPath(g.baseURL, path)
	if err != nil {
		return fmt.Errorf("build ai planning endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return fmt.Errorf("create ai planning request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		g.logger.Error("ai planning request failed",
			zap.String("trip_id", tripID.String()),
			zap.String("url", endpoint),
			zap.Error(err),
		)
		return fmt.Errorf("call ai planning service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		limitedBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxAIPlanningErrorBodyBytes))
		if readErr != nil {
			g.logger.Error("failed to read ai planning error response",
				zap.String("trip_id", tripID.String()),
				zap.Int("status_code", resp.StatusCode),
				zap.Error(readErr),
			)
			return fmt.Errorf("ai planning service returned status %d and error body could not be read: %w", resp.StatusCode, readErr)
		}

		responseBody := strings.TrimSpace(string(limitedBody))
		err := fmt.Errorf("ai planning service returned status %d: %s", resp.StatusCode, responseBody)
		g.logger.Error("ai planning service returned non-2xx response",
			zap.String("trip_id", tripID.String()),
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", responseBody),
			zap.Error(err),
		)
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		g.logger.Error("failed to decode ai planning response",
			zap.String("trip_id", tripID.String()),
			zap.String("path", path),
			zap.Error(err),
		)
		return fmt.Errorf("decode ai planning response: %w", err)
	}

	return nil
}

func enrichItineraryDefaults(itinerary *aggregate.Itinerary, trip entity.Trip) {
	if itinerary.Destination == "" {
		itinerary.Destination = trip.Destination
	}
	if itinerary.Travelers == 0 {
		itinerary.Travelers = trip.Travelers
	}
	if itinerary.Pace == "" {
		itinerary.Pace = trip.Pace
	}
	if itinerary.Pace == "" {
		itinerary.Pace = defaultPace
	}
	if itinerary.Currency == "" {
		itinerary.Currency = trip.BudgetCurrency
	}
	if itinerary.Currency == "" {
		itinerary.Currency = defaultCurrency
	}
	if itinerary.TotalBudget == nil {
		itinerary.TotalBudget = trip.BudgetAmount
	}
	if itinerary.GeneratedAt.IsZero() {
		itinerary.GeneratedAt = time.Now().UTC()
	}
	if itinerary.Source == "" {
		itinerary.Source = "ai-planning-service-http"
	}
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("AI_PLANNING_SERVICE_URL is required when ITINERARY_GENERATOR_MODE=http")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid AI_PLANNING_SERVICE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid AI_PLANNING_SERVICE_URL: scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid AI_PLANNING_SERVICE_URL: host is required")
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}
