package digests

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
	baseURL, token string
	client         *http.Client
}
type ProcessInput struct {
	Now   time.Time `json:"now"`
	Limit int       `json:"limit"`
}
type ProcessResult struct {
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
	Retrying  int `json:"retrying"`
}

func NewClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid NOTIFICATION_SERVICE_URL")
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("internal service token is required")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{baseURL: strings.TrimRight(parsed.String(), "/"), token: strings.TrimSpace(token), client: &http.Client{Timeout: timeout}}, nil
}
func (c *Client) ProcessDue(ctx context.Context, input ProcessInput) (*ProcessResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.JoinPath(c.baseURL, "/internal/notifications/process-digests")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call notification digest endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("notification digest endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var result ProcessResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
