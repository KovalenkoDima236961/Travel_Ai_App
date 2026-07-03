package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/observability"
)

const internalServiceTokenHeader = "X-Internal-Service-Token"

type Config struct {
	BaseURL        string
	Token          string
	TimeoutSeconds int
}

type CreateInput struct {
	UserID      uuid.UUID
	ActorUserID *uuid.UUID
	Type        string
	Title       string
	Message     string
	EntityType  *string
	EntityID    *uuid.UUID
	Metadata    map[string]any
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(cfg Config) (*Client, error) {
	normalized, err := normalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("NOTIFICATION_SERVICE_TOKEN is required when notifications are enabled")
	}
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 3
	}
	return &Client{
		baseURL: normalized,
		token:   strings.TrimSpace(cfg.Token),
		httpClient: observability.InstrumentHTTPClient(&http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		}),
	}, nil
}

func (c *Client) CreateNotifications(ctx context.Context, notifications []CreateInput) error {
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
		return fmt.Errorf("build notification endpoint: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create notification request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call notification service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned HTTP %d", resp.StatusCode)
	}
	return nil
}

type batchRequest struct {
	Notifications []notificationPayload `json:"notifications"`
}

type notificationPayload struct {
	UserID      string         `json:"userId"`
	ActorUserID *string        `json:"actorUserId,omitempty"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	EntityType  *string        `json:"entityType,omitempty"`
	EntityID    *string        `json:"entityId,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func toPayload(in CreateInput) notificationPayload {
	return notificationPayload{
		UserID:      in.UserID.String(),
		ActorUserID: uuidPtrString(in.ActorUserID),
		Type:        in.Type,
		Title:       in.Title,
		Message:     in.Message,
		EntityType:  in.EntityType,
		EntityID:    uuidPtrString(in.EntityID),
		Metadata:    in.Metadata,
	}
}

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
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
