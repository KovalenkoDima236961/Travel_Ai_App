package placecontext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

const maxPlaceContextErrorBodyBytes = 2 * 1024

// Error is returned for place service failures and malformed payloads.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("place context error: status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("place context error: %s", e.Message)
}

// Client calls External Integrations Service place endpoints.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient constructs a client with a validated base URL and caller-provided
// HTTP client.
func NewClient(baseURL string, client *http.Client) (*Client, error) {
	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("place context http client is required")
	}
	return &Client{baseURL: normalized, client: client}, nil
}

// SearchPlaces searches External Integrations Service for normalized places.
func (c *Client) SearchPlaces(ctx context.Context, query string, destination string) ([]aggregate.PlaceRef, error) {
	endpoint, err := url.JoinPath(c.baseURL, "/places/search")
	if err != nil {
		return nil, fmt.Errorf("build place search endpoint: %w", err)
	}

	values := url.Values{}
	values.Set("query", query)
	values.Set("destination", destination)
	endpoint = endpoint + "?" + values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create place search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, &Error{Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &Error{StatusCode: resp.StatusCode, Message: readErrorBody(resp.Body)}
	}

	var payload SearchPlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, &Error{StatusCode: resp.StatusCode, Message: err.Error()}
	}
	return payload.Items, nil
}

// GetPlaceDetails loads one normalized place by provider place ID.
func (c *Client) GetPlaceDetails(ctx context.Context, placeID string) (*aggregate.PlaceRef, error) {
	endpoint, err := url.JoinPath(c.baseURL, "/places", placeID)
	if err != nil {
		return nil, fmt.Errorf("build place details endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create place details request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, &Error{Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &Error{StatusCode: resp.StatusCode, Message: readErrorBody(resp.Body)}
	}

	var place aggregate.PlaceRef
	if err := json.NewDecoder(resp.Body).Decode(&place); err != nil {
		return nil, &Error{StatusCode: resp.StatusCode, Message: err.Error()}
	}
	return &place, nil
}

func readErrorBody(body io.Reader) string {
	limited, err := io.ReadAll(io.LimitReader(body, maxPlaceContextErrorBodyBytes))
	if err != nil {
		return "response body could not be read"
	}
	if message := strings.TrimSpace(string(limited)); message != "" {
		return message
	}
	return "empty response body"
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("EXTERNAL_INTEGRATIONS_SERVICE_URL is required when place enrichment is enabled")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid EXTERNAL_INTEGRATIONS_SERVICE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid EXTERNAL_INTEGRATIONS_SERVICE_URL: scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid EXTERNAL_INTEGRATIONS_SERVICE_URL: host is required")
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}
