package transportclient

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
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/providerlimit"
)

const internalServiceTokenHeader = "X-Internal-Service-Token"
const maxTransportClientErrorBodyBytes = 2 * 1024

type Config struct {
	BaseURL        string
	Token          string
	TimeoutSeconds int
}

type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("external integrations service url is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid external integrations service url")
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(cfg.Token),
		client: observability.InstrumentHTTPClient(&http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		}),
	}, nil
}

func (c *Client) SearchTransportOptions(ctx context.Context, input TransportSearchRequest) (*TransportSearchResponse, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode transport search request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/transport/search", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build transport search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set(internalServiceTokenHeader, c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, &Error{Code: "transport_provider_unavailable"}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxTransportClientErrorBodyBytes))
		if limitErr := providerlimit.Parse(resp.StatusCode, raw); limitErr != nil {
			return nil, limitErr
		}
		var body struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &body)
		code := strings.TrimSpace(body.Error)
		if code == "" {
			code = "transport_provider_unavailable"
		}
		return nil, &Error{StatusCode: resp.StatusCode, Code: code}
	}

	var out TransportSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, &Error{StatusCode: resp.StatusCode, Code: "transport_response_invalid"}
	}
	if out.Options == nil {
		out.Options = []TransportOption{}
	}
	return &out, nil
}

type Error struct {
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	return e.Code
}
