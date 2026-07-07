package places

import (
	"context"
	"encoding/json"
	"errors"
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
	foursquareProviderName = "foursquare"
	foursquareSearchLimit  = "10"
	foursquareFields       = "fsq_id,name,location,geocodes,categories,rating,stats,website,link"
)

type FoursquarePlaceProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

func NewFoursquarePlaceProvider(cfg config.PlaceProviderConfig, log *zap.Logger) (*FoursquarePlaceProvider, error) {
	apiKey := strings.TrimSpace(cfg.FoursquareAPIKey)
	if apiKey == "" {
		return nil, &ProviderError{Provider: foursquareProviderName, Kind: providerErrorAuthConfig}
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.FoursquareBaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.foursquare.com/v3"
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid FOURSQUARE_BASE_URL: %w", err)
	}

	timeoutSeconds := cfg.FoursquareTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 8
	}
	if log == nil {
		log = zap.NewNop()
	}

	return &FoursquarePlaceProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
		log: log,
	}, nil
}

func (p *FoursquarePlaceProvider) SearchPlaces(ctx context.Context, query string, destination string) ([]entity.Place, error) {
	start := time.Now()
	query = strings.TrimSpace(query)
	destination = strings.TrimSpace(destination)

	reqURL, err := p.buildURL("/places/search", map[string]string{
		"query":  query,
		"near":   destination,
		"limit":  foursquareSearchLimit,
		"fields": foursquareFields,
	})
	if err != nil {
		return nil, err
	}

	var payload foursquareSearchResponse
	if err := p.getJSON(ctx, reqURL, &payload); err != nil {
		p.log.Warn("place provider request failed",
			zap.String("action", "place_search"),
			zap.String("provider", foursquareProviderName),
			zap.Int("queryLength", len(query)),
			zap.String("destination", destination),
			zap.Int64("durationMs", time.Since(start).Milliseconds()),
			zap.Bool("fallbackUsed", false),
			zap.String("errorType", providerErrorKind(err)),
			zap.Error(err),
		)
		return nil, err
	}

	places := make([]entity.Place, 0, len(payload.Results))
	for _, item := range payload.Results {
		places = append(places, normalizeFoursquarePlace(item))
	}

	p.log.Info("place provider request completed",
		zap.String("action", "place_search"),
		zap.String("provider", foursquareProviderName),
		zap.Int("queryLength", len(query)),
		zap.String("destination", destination),
		zap.Int("resultCount", len(places)),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", false),
	)

	return places, nil
}

func (p *FoursquarePlaceProvider) GetPlaceDetails(ctx context.Context, providerPlaceID string) (*entity.Place, error) {
	start := time.Now()
	providerPlaceID = strings.TrimSpace(providerPlaceID)
	if providerPlaceID == "" {
		return nil, nil
	}

	reqURL, err := p.buildURL("/places/"+url.PathEscape(providerPlaceID), map[string]string{
		"fields": foursquareFields,
	})
	if err != nil {
		return nil, err
	}

	var payload foursquarePlace
	if err := p.getJSON(ctx, reqURL, &payload); err != nil {
		var providerErr *ProviderError
		if errors.As(err, &providerErr) && providerErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		p.log.Warn("place provider request failed",
			zap.String("action", "place_details"),
			zap.String("provider", foursquareProviderName),
			zap.Int64("durationMs", time.Since(start).Milliseconds()),
			zap.Bool("fallbackUsed", false),
			zap.String("errorType", providerErrorKind(err)),
			zap.Error(err),
		)
		return nil, err
	}

	place := normalizeFoursquarePlace(payload)
	p.log.Info("place provider request completed",
		zap.String("action", "place_details"),
		zap.String("provider", foursquareProviderName),
		zap.Int("resultCount", 1),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", false),
	)

	return &place, nil
}

func (p *FoursquarePlaceProvider) buildURL(path string, values map[string]string) (string, error) {
	parsed, err := url.Parse(p.baseURL + path)
	if err != nil {
		return "", fmt.Errorf("build foursquare request URL: %w", err)
	}

	query := parsed.Query()
	for key, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (p *FoursquarePlaceProvider) getJSON(ctx context.Context, requestURL string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorRequest, Err: err}
	}
	req.Header.Set("Authorization", p.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorRequest, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return classifyFoursquareStatus(resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorResponse, Err: err}
	}
	return nil
}

func classifyFoursquareStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorAuthConfig, StatusCode: status}
	case status == http.StatusTooManyRequests:
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorRateLimit, StatusCode: status}
	case status == http.StatusNotFound:
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorResponse, StatusCode: status}
	case status >= http.StatusInternalServerError:
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorUnavailable, StatusCode: status}
	default:
		return &ProviderError{Provider: foursquareProviderName, Kind: providerErrorResponse, StatusCode: status}
	}
}

func normalizeFoursquarePlace(item foursquarePlace) entity.Place {
	place := entity.Place{
		Provider:        foursquareProviderName,
		ProviderPlaceID: strings.TrimSpace(item.FsqID),
		Name:            strings.TrimSpace(item.Name),
		Address:         foursquareAddress(item.Location),
		Rating:          normalizeFoursquareRating(item.Rating),
		RatingCount:     item.Stats.TotalRatings,
		Category:        firstFoursquareCategory(item.Categories),
		Website:         strings.TrimSpace(item.Website),
		OpeningHours:    []entity.OpeningHoursInterval{},
	}

	if item.Geocodes.Main.Latitude != nil && item.Geocodes.Main.Longitude != nil {
		lat := *item.Geocodes.Main.Latitude
		lng := *item.Geocodes.Main.Longitude
		place.Latitude = &lat
		place.Longitude = &lng
	}

	link := strings.TrimSpace(item.Link)
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		place.MapURL = link
	} else if place.Latitude != nil && place.Longitude != nil {
		place.MapURL = googleMapsURL(*place.Latitude, *place.Longitude)
	}

	return place
}

func foursquareAddress(location foursquareLocation) string {
	if formatted := strings.TrimSpace(location.FormattedAddress); formatted != "" {
		return formatted
	}

	parts := []string{
		location.Address,
		location.AddressExtended,
		location.Locality,
		location.Region,
		location.Postcode,
		location.Country,
	}
	seen := make(map[string]struct{}, len(parts))
	joined := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		joined = append(joined, part)
	}
	return strings.Join(joined, ", ")
}

func normalizeFoursquareRating(rating *float64) *float64 {
	if rating == nil {
		return nil
	}

	value := *rating
	if value > 5 {
		value = value / 2
	}
	if value < 0 {
		value = 0
	}
	if value > 5 {
		value = 5
	}
	value = math.Round(value*10) / 10
	return &value
}

func firstFoursquareCategory(categories []foursquareCategory) string {
	if len(categories) == 0 {
		return ""
	}
	return strings.TrimSpace(categories[0].Name)
}

func googleMapsURL(lat, lng float64) string {
	return fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%.6f,%.6f", lat, lng)
}

type foursquareSearchResponse struct {
	Results []foursquarePlace `json:"results"`
}

type foursquarePlace struct {
	FsqID      string               `json:"fsq_id"`
	Name       string               `json:"name"`
	Location   foursquareLocation   `json:"location"`
	Geocodes   foursquareGeocodes   `json:"geocodes"`
	Categories []foursquareCategory `json:"categories"`
	Rating     *float64             `json:"rating"`
	Stats      foursquareStats      `json:"stats"`
	Website    string               `json:"website"`
	Link       string               `json:"link"`
}

type foursquareLocation struct {
	Address          string `json:"address"`
	AddressExtended  string `json:"address_extended"`
	Locality         string `json:"locality"`
	Region           string `json:"region"`
	Postcode         string `json:"postcode"`
	Country          string `json:"country"`
	FormattedAddress string `json:"formatted_address"`
}

type foursquareGeocodes struct {
	Main foursquareCoordinates `json:"main"`
}

type foursquareCoordinates struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type foursquareCategory struct {
	Name string `json:"name"`
}

type foursquareStats struct {
	TotalRatings *int `json:"total_ratings"`
}
