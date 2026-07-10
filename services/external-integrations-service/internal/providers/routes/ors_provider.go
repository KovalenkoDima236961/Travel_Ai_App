package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const (
	orsProviderName   = "ors"
	orsDefaultBaseURL = "https://api.openrouteservice.org"
)

// OpenRouteServiceProvider estimates routes via the OpenRouteService Directions
// v2 API. Provider-specific request/response shapes are isolated here; the rest
// of the service only sees the canonical entity types.
type OpenRouteServiceProvider struct {
	apiKey   string
	baseURL  string
	profiles map[string]string
	client   *http.Client
	log      *zap.Logger
}

// NewOpenRouteServiceProvider builds the ORS provider. A missing API key is
// reported as an auth/config ProviderError so the selector can decide whether to
// fall back to mock or fail startup.
func NewOpenRouteServiceProvider(cfg config.RouteProviderConfig, log *zap.Logger) (*OpenRouteServiceProvider, error) {
	apiKey := strings.TrimSpace(cfg.ORSAPIKey)
	if apiKey == "" {
		return nil, &ProviderError{Provider: orsProviderName, Kind: providerErrorAuthConfig}
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.ORSBaseURL), "/")
	if baseURL == "" {
		baseURL = orsDefaultBaseURL
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid ORS_BASE_URL: %w", err)
	}

	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 8
	}
	if log == nil {
		log = zap.NewNop()
	}

	return &OpenRouteServiceProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		profiles: map[string]string{
			entity.RouteModeWalk:      firstNonEmpty(cfg.ORSProfileWalking, "foot-walking"),
			entity.RouteModeHiking:    firstNonEmpty(cfg.ORSProfileWalking, "foot-walking"),
			entity.RouteModeCar:       firstNonEmpty(cfg.ORSProfileDriving, "driving-car"),
			entity.RouteModeRentalCar: firstNonEmpty(cfg.ORSProfileDriving, "driving-car"),
			entity.RouteModeBike:      firstNonEmpty(cfg.ORSProfileCycling, "cycling-regular"),
		},
		client: &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		log:    log,
	}, nil
}

// EstimateRoute maps the canonical request to an ORS Directions call and
// normalises the response. Coordinates are sent as [longitude, latitude] pairs,
// the order ORS requires.
func (p *OpenRouteServiceProvider) EstimateRoute(ctx context.Context, req entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	start := time.Now()
	req = entity.NormalizeRouteEstimateRequest(req)
	mode := strings.ToLower(strings.TrimSpace(req.Mode))

	profile, ok := p.profiles[mode]
	if !ok {
		return nil, &ProviderError{Provider: orsProviderName, Kind: providerErrorRequest, Err: fmt.Errorf("unsupported mode %q", mode)}
	}

	coordinates := make([][]float64, 0, len(req.Stops))
	for _, stop := range req.Stops {
		coordinates = append(coordinates, []float64{stop.Longitude, stop.Latitude})
	}

	body, err := json.Marshal(orsDirectionsRequest{Coordinates: coordinates})
	if err != nil {
		return nil, &ProviderError{Provider: orsProviderName, Kind: providerErrorRequest, Err: err}
	}

	reqURL := p.baseURL + "/v2/directions/" + url.PathEscape(profile)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, &ProviderError{Provider: orsProviderName, Kind: providerErrorRequest, Err: err}
	}
	httpReq.Header.Set("Authorization", p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, p.failure(req, mode, start, &ProviderError{Provider: orsProviderName, Kind: providerErrorRequest, Err: err})
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, p.failure(req, mode, start, classifyORSStatus(resp.StatusCode))
	}

	var payload orsDirectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, p.failure(req, mode, start, &ProviderError{Provider: orsProviderName, Kind: providerErrorResponse, Err: err})
	}

	estimate, err := normalizeORSRoute(req, mode, payload)
	if err != nil {
		return nil, p.failure(req, mode, start, err)
	}

	p.log.Info("route provider request completed",
		zap.String("action", "route_estimate"),
		zap.String("provider", orsProviderName),
		zap.String("mode", mode),
		zap.Int("stopCount", len(req.Stops)),
		zap.Float64("distanceKm", estimate.DistanceKm),
		zap.Int("durationMinutes", estimate.DurationMinutes),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", false),
	)

	return estimate, nil
}

// failure logs a structured warning (never the API key or raw provider body) and
// returns the classified error unchanged.
func (p *OpenRouteServiceProvider) failure(req entity.RouteEstimateRequest, mode string, start time.Time, err error) error {
	p.log.Warn("route provider request failed",
		zap.String("action", "route_estimate"),
		zap.String("provider", orsProviderName),
		zap.String("mode", mode),
		zap.Int("stopCount", len(req.Stops)),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", false),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)
	return err
}

// normalizeORSRoute converts the first ORS route into the canonical estimate.
// Per-leg ORS segments map one-to-one to consecutive stop pairs; the total is
// the sum of the rounded segments so the response stays internally consistent.
func normalizeORSRoute(req entity.RouteEstimateRequest, mode string, payload orsDirectionsResponse) (*entity.RouteEstimate, error) {
	if len(payload.Routes) == 0 {
		return nil, &ProviderError{Provider: orsProviderName, Kind: providerErrorResponse, Err: fmt.Errorf("no routes returned")}
	}
	route := payload.Routes[0]

	segments := make([]entity.RouteSegment, 0, len(req.Stops)-1)
	var totalDistanceKm float64
	var totalDurationMinutes int

	for i := 1; i < len(req.Stops); i++ {
		var distanceMeters, durationSeconds float64
		if idx := i - 1; idx < len(route.Segments) {
			distanceMeters = route.Segments[idx].Distance
			durationSeconds = route.Segments[idx].Duration
		}

		distanceKm := round2(distanceMeters / 1000)
		durationMinutes := int(math.Round(durationSeconds / 60))

		segments = append(segments, entity.RouteSegment{
			FromName:                 req.Stops[i-1].Name,
			ToName:                   req.Stops[i].Name,
			DistanceKm:               distanceKm,
			EstimatedDistanceKm:      distanceKm,
			DurationMinutes:          durationMinutes,
			EstimatedDurationMinutes: durationMinutes,
			EstimatedCost:            estimatedCostForMode(distanceKm, mode, req.Currency),
		})
		totalDistanceKm += distanceKm
		totalDurationMinutes += durationMinutes
	}

	estimate := &entity.RouteEstimate{
		Mode:                     mode,
		Provider:                 orsProviderName,
		DistanceKm:               round2(totalDistanceKm),
		EstimatedDistanceKm:      round2(totalDistanceKm),
		DurationMinutes:          totalDurationMinutes,
		EstimatedDurationMinutes: totalDurationMinutes,
		EstimatedCost:            estimatedCostForMode(totalDistanceKm, mode, req.Currency),
		Segments:                 segments,
	}
	if mode == entity.RouteModeHiking {
		estimate.Warnings = warningsForMode(mode)
	}
	if geometry := strings.TrimSpace(route.Geometry); geometry != "" {
		estimate.RouteGeometry = geometry
	}
	return estimate, nil
}

func classifyORSStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &ProviderError{Provider: orsProviderName, Kind: providerErrorAuthConfig, StatusCode: status}
	case status == http.StatusTooManyRequests:
		return &ProviderError{Provider: orsProviderName, Kind: providerErrorRateLimit, StatusCode: status}
	case status >= http.StatusInternalServerError:
		return &ProviderError{Provider: orsProviderName, Kind: providerErrorUnavailable, StatusCode: status}
	default:
		return &ProviderError{Provider: orsProviderName, Kind: providerErrorResponse, StatusCode: status}
	}
}

func firstNonEmpty(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

type orsDirectionsRequest struct {
	Coordinates [][]float64 `json:"coordinates"`
}

type orsDirectionsResponse struct {
	Routes []orsRoute `json:"routes"`
}

type orsRoute struct {
	Summary  orsSummary   `json:"summary"`
	Segments []orsSegment `json:"segments"`
	Geometry string       `json:"geometry"`
}

type orsSummary struct {
	Distance float64 `json:"distance"`
	Duration float64 `json:"duration"`
}

type orsSegment struct {
	Distance float64 `json:"distance"`
	Duration float64 `json:"duration"`
}
