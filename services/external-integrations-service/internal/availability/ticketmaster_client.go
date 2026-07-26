package availability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ticketmasterClient is a thin HTTP wrapper around the Discovery API Event
// Search endpoint. It owns request construction, auth injection, and error
// classification; it never logs the API key and never returns raw provider
// payloads to callers.
type ticketmasterClient struct {
	apiKey  string
	baseURL *url.URL
	http    *http.Client
	log     *zap.Logger
}

func newTicketmasterClient(apiKey, baseURL string, timeout time.Duration, log *zap.Logger) (*ticketmasterClient, error) {
	if log == nil {
		log = zap.NewNop()
	}
	normalizedBaseURL, err := normalizeTicketmasterBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &ticketmasterClient{
		apiKey:  apiKey,
		baseURL: normalizedBaseURL,
		http:    &http.Client{Timeout: timeout},
		log:     log,
	}, nil
}

// searchEvents performs a GET /events.json with the given query parameters. The
// apikey is appended here so callers never handle it.
func (c *ticketmasterClient) searchEvents(ctx context.Context, params url.Values) (*tmEventsResponse, error) {
	endpoint, err := c.buildURL("/events.json", params)
	if err != nil {
		return nil, &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorRequest, Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil) // #nosec G704 -- endpoint is built from a validated Ticketmaster base URL and fixed path.
	if err != nil {
		return nil, &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorRequest, Err: err}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req) // #nosec G704 -- request URL is built from a validated Ticketmaster base URL and fixed path.
	if err != nil {
		return nil, classifyTicketmasterTransportError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// 404 means "no events matched" rather than a hard failure; surface it as
		// an empty result so the provider can report unknown/no-match cleanly.
		if resp.StatusCode == http.StatusNotFound {
			return &tmEventsResponse{}, nil
		}
		return nil, classifyTicketmasterStatus(resp.StatusCode)
	}

	var payload tmEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorBadResponse, Err: err}
	}
	return &payload, nil
}

func normalizeTicketmasterBaseURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("TICKETMASTER_BASE_URL is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse TICKETMASTER_BASE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("TICKETMASTER_BASE_URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("TICKETMASTER_BASE_URL must include a host")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("TICKETMASTER_BASE_URL must not include user info")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("TICKETMASTER_BASE_URL must not include query or fragment")
	}
	return parsed, nil
}

// buildURL appends the API key as a query parameter. The key is never logged.
func (c *ticketmasterClient) buildURL(path string, values url.Values) (string, error) {
	parsed := *c.baseURL
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(path, "/")
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	query := parsed.Query()
	for key, list := range values {
		for _, value := range list {
			if value != "" {
				query.Add(key, value)
			}
		}
	}
	query.Set("apikey", c.apiKey)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func classifyTicketmasterTransportError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorTimeout, Err: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorTimeout, Err: err}
	}
	return &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorRequest, Err: err}
}

func classifyTicketmasterStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorAuthConfig, StatusCode: status}
	case status == http.StatusTooManyRequests:
		return &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorRateLimit, StatusCode: status}
	case status >= http.StatusInternalServerError:
		return &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorUnavailable, StatusCode: status}
	default:
		return &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorBadResponse, StatusCode: status}
	}
}
