package weather

import (
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
	openWeatherProviderName   = "openweathermap"
	openWeatherDefaultBaseURL = "https://api.openweathermap.org"
	openWeatherDefaultUnits   = "metric"
)

// OpenWeatherProvider produces daily forecasts from the OpenWeatherMap APIs. It
// geocodes the destination, fetches the 5 day / 3 hour forecast, and groups the
// 3-hour entries into one forecast per local day. Provider-specific shapes are
// isolated here; the rest of the service sees only the canonical entity types.
type OpenWeatherProvider struct {
	apiKey  string
	baseURL string
	units   string
	client  *http.Client
	log     *zap.Logger
}

// NewOpenWeatherProvider builds the provider. A missing API key is reported as
// an auth/config ProviderError so the selector can fall back to mock or fail
// startup.
func NewOpenWeatherProvider(cfg config.WeatherProviderConfig, log *zap.Logger) (*OpenWeatherProvider, error) {
	apiKey := strings.TrimSpace(cfg.OpenWeatherAPIKey)
	if apiKey == "" {
		return nil, &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorAuthConfig}
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.OpenWeatherBaseURL), "/")
	if baseURL == "" {
		baseURL = openWeatherDefaultBaseURL
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid OPENWEATHER_BASE_URL: %w", err)
	}

	units := strings.ToLower(strings.TrimSpace(cfg.OpenWeatherUnits))
	if units == "" {
		units = openWeatherDefaultUnits
	}

	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 8
	}
	if log == nil {
		log = zap.NewNop()
	}

	return &OpenWeatherProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		units:   units,
		client:  &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		log:     log,
	}, nil
}

// GetForecast resolves the destination, fetches the forecast, and normalises it.
// If the provider cannot cover every requested date it returns an error so the
// caller can fall back to the mock provider for the whole forecast, keeping the
// response exactly req.Days long with a single, honest provider label.
func (p *OpenWeatherProvider) GetForecast(ctx context.Context, req entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	start := time.Now()

	lat, lon, err := p.geocode(ctx, req.Destination)
	if err != nil {
		return nil, p.failure(req, start, err)
	}

	payload, err := p.fetchForecast(ctx, lat, lon)
	if err != nil {
		return nil, p.failure(req, start, err)
	}

	forecast, err := normalizeOpenWeather(req, p.units, payload)
	if err != nil {
		return nil, p.failure(req, start, err)
	}

	p.log.Info("weather provider request completed",
		zap.String("action", "weather_forecast"),
		zap.String("provider", openWeatherProviderName),
		zap.String("destination", req.Destination),
		zap.Int("days", len(forecast.Days)),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", false),
	)
	return forecast, nil
}

func (p *OpenWeatherProvider) failure(req entity.WeatherForecastRequest, start time.Time, err error) error {
	p.log.Warn("weather provider request failed",
		zap.String("action", "weather_forecast"),
		zap.String("provider", openWeatherProviderName),
		zap.String("destination", req.Destination),
		zap.Int("days", req.Days),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", false),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)
	return err
}

// geocode resolves a destination name to coordinates via the Geocoding API. An
// empty result is treated as a not-found provider error.
func (p *OpenWeatherProvider) geocode(ctx context.Context, destination string) (float64, float64, error) {
	endpoint, err := p.buildURL("/geo/1.0/direct", map[string]string{
		"q":     strings.TrimSpace(destination),
		"limit": "1",
	})
	if err != nil {
		return 0, 0, err
	}

	var results []owmGeoResult
	if err := p.getJSON(ctx, endpoint, &results); err != nil {
		return 0, 0, err
	}
	if len(results) == 0 {
		return 0, 0, &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorNotFound, Err: fmt.Errorf("destination not found")}
	}
	return results[0].Lat, results[0].Lon, nil
}

func (p *OpenWeatherProvider) fetchForecast(ctx context.Context, lat, lon float64) (*owmForecastResponse, error) {
	endpoint, err := p.buildURL("/data/2.5/forecast", map[string]string{
		"lat":   fmt.Sprintf("%.6f", lat),
		"lon":   fmt.Sprintf("%.6f", lon),
		"units": p.units,
	})
	if err != nil {
		return nil, err
	}

	var payload owmForecastResponse
	if err := p.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// buildURL appends the API key as a query parameter. The key is never logged.
func (p *OpenWeatherProvider) buildURL(path string, values map[string]string) (string, error) {
	parsed, err := url.Parse(p.baseURL + path)
	if err != nil {
		return "", &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorRequest, Err: err}
	}
	query := parsed.Query()
	for key, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			query.Set(key, value)
		}
	}
	query.Set("appid", p.apiKey)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (p *OpenWeatherProvider) getJSON(ctx context.Context, requestURL string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorRequest, Err: err}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorRequest, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return classifyOpenWeatherStatus(resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorResponse, Err: err}
	}
	return nil
}

func classifyOpenWeatherStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorAuthConfig, StatusCode: status}
	case status == http.StatusTooManyRequests:
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorRateLimit, StatusCode: status}
	case status == http.StatusNotFound:
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorNotFound, StatusCode: status}
	case status >= http.StatusInternalServerError:
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorUnavailable, StatusCode: status}
	default:
		return &ProviderError{Provider: openWeatherProviderName, Kind: providerErrorResponse, StatusCode: status}
	}
}

// dailyAggregate accumulates the 3-hour entries that fall on one local date.
type dailyAggregate struct {
	minC       float64
	maxC       float64
	precip     float64
	windKph    float64
	conditions map[string]int
	count      int
}

// normalizeOpenWeather groups 3-hour entries by local date (using the city's UTC
// offset) and builds a forecast for each requested date. If any requested date
// has no data, it returns a coverage error so the caller falls back to mock.
func normalizeOpenWeather(req entity.WeatherForecastRequest, units string, payload *owmForecastResponse) (*entity.WeatherForecast, error) {
	offset := time.Duration(payload.City.Timezone) * time.Second
	byDate := make(map[string]*dailyAggregate, req.Days+1)

	for _, item := range payload.List {
		localDate := time.Unix(item.Dt, 0).UTC().Add(offset).Format("2006-01-02")
		agg := byDate[localDate]
		if agg == nil {
			agg = &dailyAggregate{conditions: make(map[string]int)}
			byDate[localDate] = agg
		}

		minC := convertTemperatureToCelsius(item.Main.TempMin, units)
		maxC := convertTemperatureToCelsius(item.Main.TempMax, units)
		windKph := convertWindToKph(item.Wind.Speed, units)

		if agg.count == 0 {
			agg.minC = minC
			agg.maxC = maxC
		} else {
			agg.minC = math.Min(agg.minC, minC)
			agg.maxC = math.Max(agg.maxC, maxC)
		}
		if item.Pop > agg.precip {
			agg.precip = item.Pop
		}
		if windKph > agg.windKph {
			agg.windKph = windKph
		}
		if len(item.Weather) > 0 {
			agg.conditions[strings.TrimSpace(item.Weather[0].Main)]++
		}
		agg.count++
	}

	days := make([]entity.WeatherDay, 0, req.Days)
	for i := 0; i < req.Days; i++ {
		date := req.StartDate.AddDate(0, 0, i).Format("2006-01-02")
		agg, ok := byDate[date]
		if !ok || agg.count == 0 {
			return nil, &ProviderError{
				Provider: openWeatherProviderName,
				Kind:     providerErrorResponse,
				Err:      fmt.Errorf("no forecast coverage for %s", date),
			}
		}

		condition, summary := mapOpenWeatherCondition(dominantCondition(agg.conditions))
		day := entity.WeatherDay{
			Date:                date,
			Condition:           condition,
			TemperatureMinC:     round1(agg.minC),
			TemperatureMaxC:     round1(agg.maxC),
			PrecipitationChance: int(math.Round(agg.precip * 100)),
			WindSpeedKph:        round1(agg.windKph),
			Summary:             summary,
		}
		day.Warnings = weatherWarnings(day)
		days = append(days, day)
	}

	return &entity.WeatherForecast{
		Destination: req.Destination,
		Provider:    openWeatherProviderName,
		Days:        days,
	}, nil
}

func convertTemperatureToCelsius(value float64, units string) float64 {
	switch units {
	case "imperial":
		return (value - 32) * 5 / 9
	case "standard": // Kelvin
		return value - 273.15
	default: // metric is already Celsius
		return value
	}
}

func convertWindToKph(value float64, units string) float64 {
	switch units {
	case "imperial": // miles per hour
		return value * 1.609344
	default: // metric and standard report metres per second
		return value * 3.6
	}
}

// dominantCondition picks the most frequent OpenWeather condition, breaking ties
// in favour of the more severe condition so a stormy afternoon is not hidden by
// a clear morning.
func dominantCondition(conditions map[string]int) string {
	best := ""
	bestCount := -1
	for condition, count := range conditions {
		if count > bestCount || (count == bestCount && conditionSeverity(condition) > conditionSeverity(best)) {
			best = condition
			bestCount = count
		}
	}
	return best
}

func conditionSeverity(main string) int {
	switch strings.ToLower(strings.TrimSpace(main)) {
	case "tornado", "squall":
		return 7
	case "thunderstorm":
		return 6
	case "snow":
		return 5
	case "rain", "drizzle":
		return 4
	case "clouds":
		return 2
	case "clear":
		return 1
	default: // mist, fog, haze, smoke, dust, etc.
		return 3
	}
}

// mapOpenWeatherCondition maps an OpenWeather "main" condition to the simple
// condition/summary vocabulary the mock provider and UI already use.
func mapOpenWeatherCondition(main string) (condition string, summary string) {
	switch strings.ToLower(strings.TrimSpace(main)) {
	case "clear":
		return "sunny", "Clear and sunny"
	case "clouds":
		return "partly_cloudy", "Partly cloudy"
	case "drizzle":
		return "light_rain", "Light rain likely"
	case "rain":
		return "rain", "Rain likely"
	case "thunderstorm":
		return "storm", "Thunderstorms likely"
	case "snow":
		return "snow", "Snow likely"
	case "mist", "fog", "haze", "smoke", "dust", "sand", "ash":
		return "fog", "Reduced visibility"
	case "squall", "tornado":
		return "windy", "Severe wind"
	case "":
		return "partly_cloudy", "Mixed conditions"
	default:
		return "partly_cloudy", "Mixed conditions"
	}
}

type owmGeoResult struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
}

type owmForecastResponse struct {
	List []owmForecastEntry `json:"list"`
	City owmCity            `json:"city"`
}

type owmForecastEntry struct {
	Dt      int64        `json:"dt"`
	Main    owmMain      `json:"main"`
	Weather []owmWeather `json:"weather"`
	Wind    owmWind      `json:"wind"`
	Pop     float64      `json:"pop"`
	DtTxt   string       `json:"dt_txt"`
}

type owmMain struct {
	Temp    float64 `json:"temp"`
	TempMin float64 `json:"temp_min"`
	TempMax float64 `json:"temp_max"`
}

type owmWeather struct {
	Main        string `json:"main"`
	Description string `json:"description"`
}

type owmWind struct {
	Speed float64 `json:"speed"`
}

type owmCity struct {
	Timezone int `json:"timezone"`
}
