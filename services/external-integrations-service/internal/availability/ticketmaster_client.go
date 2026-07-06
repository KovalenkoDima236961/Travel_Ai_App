package availability

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// ticketmasterClient is a thin HTTP wrapper around the Discovery API Event
// Search endpoint. It owns request construction, auth injection, and error
// classification; it never logs the API key and never returns raw provider
// payloads to callers.
type ticketmasterClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
	log     *zap.Logger
}

func newTicketmasterClient(apiKey, baseURL string, timeout time.Duration, log *zap.Logger) *ticketmasterClient {
	if log == nil {
		log = zap.NewNop()
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &ticketmasterClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
		log:     log,
	}
}

// searchEvents performs a GET /events.json with the given query parameters. The
// apikey is appended here so callers never handle it.
func (c *ticketmasterClient) searchEvents(ctx context.Context, params url.Values) (*tmEventsResponse, error) {
	endpoint, err := c.buildURL("/events.json", params)
	if err != nil {
		return nil, &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorRequest, Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorRequest, Err: err}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
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

// buildURL appends the API key as a query parameter. The key is never logged.
func (c *ticketmasterClient) buildURL(path string, values url.Values) (string, error) {
	parsed, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", err
	}
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
