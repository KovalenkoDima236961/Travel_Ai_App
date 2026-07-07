package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

const maxNotificationErrorBodyBytes = 2 * 1024

// internalServiceTokenHeader is the header the Notification Service requires on
// internal endpoints. It mirrors the constant in Notification Service; kept
// local so Trip Service has no source dependency on that module.
const internalServiceTokenHeader = "X-Internal-Service-Token"

// Client calls the Notification Service internal batch endpoint. It is a thin
// HTTP wrapper: it never decides whether notifications are enabled or whether a
// failure should break the caller — that policy lives in the trip use case.
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient constructs a client with a validated base URL, the shared internal
// service token, and a caller-provided (timeout-bound) HTTP client.
func NewClient(baseURL, token string, client *http.Client) (*Client, error) {
	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("NOTIFICATION_SERVICE_TOKEN is required when notifications are enabled")
	}
	if client == nil {
		return nil, fmt.Errorf("notification http client is required")
	}
	return &Client{baseURL: normalized, token: token, client: observability.InstrumentHTTPClient(client)}, nil
}

// CreateNotifications creates a batch of notifications. An empty batch is a
// no-op (no HTTP call). A non-2xx response or transport error is returned to the
// caller; the caller decides whether to swallow it (fail-open).
//
// The internal service token is sent in a header and is never logged.
func (c *Client) CreateNotifications(ctx context.Context, notifications []NotificationCreateInput) error {
	if len(notifications) == 0 {
		return nil
	}

	payload := batchRequest{Notifications: make([]notificationPayload, 0, len(notifications))}
	for _, n := range notifications {
		payload.Notifications = append(payload.Notifications, toPayload(n))
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notifications batch: %w", err)
	}

	endpoint, err := url.JoinPath(c.baseURL, "/internal/notifications/batch")
	if err != nil {
		return fmt.Errorf("build notification service endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create notification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("call notification service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	// Decode the count best-effort; a malformed success body is not fatal.
	var decoded batchResponse
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return nil
}

func readErrorBody(body io.Reader) string {
	limited, err := io.ReadAll(io.LimitReader(body, maxNotificationErrorBodyBytes))
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
		return "", fmt.Errorf("NOTIFICATION_SERVICE_URL is required when notifications are enabled")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid NOTIFICATION_SERVICE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid NOTIFICATION_SERVICE_URL: scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid NOTIFICATION_SERVICE_URL: host is required")
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}
