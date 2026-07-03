package authusers

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

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/observability"
)

const internalServiceTokenHeader = "X-Internal-Service-Token"

type Config struct {
	BaseURL        string
	Token          string
	TimeoutSeconds int
}

type User struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("AUTH_SERVICE_URL is required")
	}
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 5
	}
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(cfg.Token),
		httpClient: observability.InstrumentHTTPClient(&http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		}),
	}, nil
}

func (c *Client) LookupByEmail(ctx context.Context, email string) (*User, error) {
	reqURL := c.baseURL + "/internal/users/by-email?email=" + url.QueryEscape(email)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build user lookup request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call user lookup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domainerrs.ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("user lookup returned HTTP %d", resp.StatusCode)
	}

	var body userPayload
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode user lookup: %w", err)
	}
	return body.toUser()
}

func (c *Client) BatchByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]User, error) {
	result := make(map[uuid.UUID]User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	payload := struct {
		UserIDs []string `json:"userIds"`
	}{UserIDs: make([]string, 0, len(ids))}
	for _, id := range ids {
		if id != uuid.Nil {
			payload.UserIDs = append(payload.UserIDs, id.String())
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal user batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/users/batch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build user batch request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set(internalServiceTokenHeader, c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call user batch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("user batch returned HTTP %d", resp.StatusCode)
	}

	var decoded struct {
		Items []userPayload `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode user batch: %w", err)
	}
	for _, item := range decoded.Items {
		user, err := item.toUser()
		if err != nil {
			return nil, err
		}
		result[user.UserID] = *user
	}
	return result, nil
}

type userPayload struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

func (p userPayload) toUser() (*User, error) {
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		return nil, fmt.Errorf("decode userId: %w", err)
	}
	return &User{
		UserID:      userID,
		Email:       p.Email,
		DisplayName: p.DisplayName,
	}, nil
}
