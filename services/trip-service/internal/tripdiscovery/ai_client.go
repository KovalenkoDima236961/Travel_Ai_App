package tripdiscovery

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

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

const maxErrorBody = 4 * 1024

type AIClient interface {
	SuggestDestinations(context.Context, AIRequest) (*SuggestionResponse, error)
}

type HTTPAIClient struct {
	baseURL *url.URL
	client  *http.Client
}

func NewHTTPAIClient(baseURL string, timeout time.Duration) (*HTTPAIClient, error) {
	normalized, err := normalizeAIPlanningBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &HTTPAIClient{
		baseURL: normalized,
		client:  observability.InstrumentHTTPClient(&http.Client{Timeout: timeout}),
	}, nil
}

func normalizeAIPlanningBaseURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("AI_PLANNING_SERVICE_URL is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse AI_PLANNING_SERVICE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("AI_PLANNING_SERVICE_URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("AI_PLANNING_SERVICE_URL must include a host")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("AI_PLANNING_SERVICE_URL must not include user info")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("AI_PLANNING_SERVICE_URL must not include query or fragment")
	}
	return parsed, nil
}

func (c *HTTPAIClient) endpoint(path string) string {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint.RawPath = ""
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	return endpoint.String()
}

func (c *HTTPAIClient) SuggestDestinations(
	ctx context.Context,
	input AIRequest,
) (*SuggestionResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal destination suggestion request: %w", err)
	}
	req, err := http.NewRequestWithContext( // #nosec G704 -- endpoint is built from a validated service base URL.
		ctx,
		http.MethodPost,
		c.endpoint("suggest-destinations"),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build destination suggestion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req) // #nosec G704 -- request URL is built from a validated service base URL.
	if err != nil {
		return nil, fmt.Errorf("call destination suggestion endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return nil, fmt.Errorf(
			"destination suggestion endpoint returned HTTP %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(raw)),
		)
	}
	var result SuggestionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode destination suggestion response: %w", err)
	}
	if len(result.Suggestions) == 0 {
		return nil, fmt.Errorf("destination suggestion endpoint returned no suggestions")
	}
	return &result, nil
}
