package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

type Config struct {
	BaseURL        string
	TimeoutSeconds int
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("users client base URL is required")
	}
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 5
	}
	return &Client{
		baseURL: baseURL,
		httpClient: observability.InstrumentHTTPClient(&http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		}),
	}, nil
}

func (c *Client) LookupByEmail(ctx context.Context, email string) (*appdto.UserLookupResult, error) {
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

	var body struct {
		UserID      string `json:"userId"`
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode user lookup: %w", err)
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		return nil, fmt.Errorf("decode user lookup userId: %w", err)
	}
	return &appdto.UserLookupResult{
		UserID:      userID,
		Email:       body.Email,
		DisplayName: body.DisplayName,
	}, nil
}
