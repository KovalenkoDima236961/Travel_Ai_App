package reminders

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
)

const internalServiceTokenHeader = "X-Internal-Service-Token"

type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

type ProcessInput struct {
	Now   time.Time `json:"now"`
	Limit int       `json:"limit"`
}

type ProcessResult struct {
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
}

func NewClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("internal service token is required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: normalized,
		token:   strings.TrimSpace(token),
		client:  &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) ProcessDue(ctx context.Context, input ProcessInput) (*ProcessResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal process due reminders: %w", err)
	}
	endpoint, err := url.JoinPath(c.baseURL, "/internal/reminders/process-due")
	if err != nil {
		return nil, fmt.Errorf("build process due reminders endpoint: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create process due reminders request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call trip reminders endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("trip reminders endpoint returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}
	var result ProcessResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode process due reminders response: %w", err)
	}
	return &result, nil
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("TRIP_SERVICE_URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid TRIP_SERVICE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid TRIP_SERVICE_URL: scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid TRIP_SERVICE_URL: host is required")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func readErrorBody(body io.Reader) string {
	limited, err := io.ReadAll(io.LimitReader(body, 2048))
	if err != nil {
		return "response body could not be read"
	}
	if message := strings.TrimSpace(string(limited)); message != "" {
		return message
	}
	return "empty response body"
}
