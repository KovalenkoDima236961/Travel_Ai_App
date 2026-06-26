package calendarclient

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
		timeout = 30
	}
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(cfg.Token),
		client:  &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}, nil
}

func (c *Client) GetGoogleCalendarStatus(ctx context.Context, accessToken string) (*ConnectionStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/calendar/google/status", nil)
	if err != nil {
		return nil, fmt.Errorf("build calendar status request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	var out ConnectionStatus
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SyncGoogleCalendarEvents(ctx context.Context, input SyncRequest) (*SyncResult, error) {
	var out SyncResult
	if err := c.postInternal(ctx, "/internal/calendar/google/events/sync", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteGoogleCalendarEvents(ctx context.Context, input DeleteRequest) (*DeleteResult, error) {
	var out DeleteResult
	if err := c.postInternal(ctx, "/internal/calendar/google/events/delete", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) postInternal(ctx context.Context, path string, input any, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode calendar request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build calendar request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)
	return c.do(req, output)
}

func (c *Client) do(req *http.Request, output any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("calendar service request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var body struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		code := strings.TrimSpace(body.Error)
		if code == "" {
			code = "calendar_service_error"
		}
		return &Error{StatusCode: resp.StatusCode, Code: code}
	}
	if output != nil {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
			return fmt.Errorf("decode calendar service response: %w", err)
		}
	}
	return nil
}

type Error struct {
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	return e.Code
}
