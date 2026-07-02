package priceclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const internalServiceTokenHeader = "X-Internal-Service-Token"

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
		timeout = 8
	}
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(cfg.Token),
		client:  &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}, nil
}

func (c *Client) EstimatePrice(ctx context.Context, input PriceEstimateInput) (*PriceEstimateResult, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode price estimate request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/prices/estimate", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build price estimate request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set(internalServiceTokenHeader, c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, &Error{Code: "price_provider_unavailable"}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var body struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		code := strings.TrimSpace(body.Error)
		if code == "" {
			code = "price_provider_unavailable"
		}
		return nil, &Error{StatusCode: resp.StatusCode, Code: code}
	}

	var out PriceEstimateResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, &Error{StatusCode: resp.StatusCode, Code: "price_response_invalid"}
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
