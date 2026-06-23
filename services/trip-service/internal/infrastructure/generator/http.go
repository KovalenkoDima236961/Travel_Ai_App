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
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
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
}

type aiPlanningRegenerateDayResponse struct {
	Day aggregate.ItineraryDay `json:"day"`
}

type aiPlanningRegenerateItemResponse struct {
	Item aggregate.ItineraryItem `json:"item"`
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
		client:  client,
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
		Trip:             newAIPlanningTripRequest(trip),
		CurrentItinerary: input.CurrentItinerary,
		DayNumber:        input.DayNumber,
		Instruction:      input.Instruction,
		UserProfile:      input.UserProfile,
		UserPreferences:  input.UserPreferences,
		WeatherForecast:  input.WeatherForecast,
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
		Trip:             newAIPlanningTripRequest(trip),
		CurrentItinerary: input.CurrentItinerary,
		DayNumber:        input.DayNumber,
		ItemIndex:        input.ItemIndex,
		Instruction:      input.Instruction,
		UserProfile:      input.UserProfile,
		UserPreferences:  input.UserPreferences,
		WeatherForecast:  input.WeatherForecast,
	}

	var result aiPlanningRegenerateItemResponse
	if err := g.postJSON(ctx, trip.ID, "regenerate-item", payload, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

func newAIPlanningGenerateRequest(input application.GenerateItineraryInput) aiPlanningGenerateRequest {
	trip := input.Trip
	var startDate *string
	if trip.StartDate != nil {
		formatted := trip.StartDate.Format("2006-01-02")
		startDate = &formatted
	}

	currency := strings.TrimSpace(trip.BudgetCurrency)
	if currency == "" {
		currency = defaultCurrency
	}
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
	}
}

func newAIPlanningTripRequest(trip entity.Trip) aiPlanningTripRequest {
	var startDate *string
	if trip.StartDate != nil {
		formatted := trip.StartDate.Format("2006-01-02")
		startDate = &formatted
	}

	currency := strings.TrimSpace(trip.BudgetCurrency)
	if currency == "" {
		currency = defaultCurrency
	}
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
