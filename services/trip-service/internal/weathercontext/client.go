package weathercontext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxWeatherContextErrorBodyBytes = 2 * 1024

// Error is returned for weather service failures and malformed payloads.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("weather context error: status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("weather context error: %s", e.Message)
}

// Client calls External Integrations Service weather endpoints.
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
		return nil, fmt.Errorf("weather context http client is required")
	}
	return &Client{baseURL: normalized, client: client}, nil
}

// GetForecast loads a daily forecast from External Integrations Service.
func (c *Client) GetForecast(ctx context.Context, destination string, startDate string, days int) (*WeatherForecast, error) {
	endpoint, err := url.JoinPath(c.baseURL, "/weather/forecast")
	if err != nil {
		return nil, fmt.Errorf("build weather forecast endpoint: %w", err)
	}

	values := url.Values{}
	values.Set("destination", destination)
	values.Set("startDate", startDate)
	values.Set("days", strconv.Itoa(days))
	endpoint = endpoint + "?" + values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create weather forecast request: %w", err)
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

	var forecast WeatherForecast
	if err := json.NewDecoder(resp.Body).Decode(&forecast); err != nil {
		return nil, &Error{StatusCode: resp.StatusCode, Message: err.Error()}
	}
	return &forecast, nil
}

func readErrorBody(body io.Reader) string {
	limited, err := io.ReadAll(io.LimitReader(body, maxWeatherContextErrorBodyBytes))
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
		return "", fmt.Errorf("EXTERNAL_INTEGRATIONS_SERVICE_URL is required when weather context is enabled")
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
