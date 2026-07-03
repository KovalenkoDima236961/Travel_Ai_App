package workspaces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

const internalServiceTokenHeader = "X-Internal-Service-Token"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

type Access struct {
	HasAccess         bool
	Role              Role
	Status            string
	WorkspaceArchived bool
}

type UserWorkspace struct {
	ID   uuid.UUID
	Role Role
}

type Config struct {
	BaseURL        string
	Token          string
	TimeoutSeconds int
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("USER_SERVICE_URL is required")
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, fmt.Errorf("WORKSPACE_SERVICE_TOKEN is required")
	}
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 5
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: observability.InstrumentHTTPClient(&http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		}),
	}, nil
}

func (c *Client) AccessCheck(ctx context.Context, userID, workspaceID uuid.UUID) (*Access, error) {
	var resp accessResponse
	if err := c.postJSON(ctx, "/internal/workspaces/access-check", accessRequest{
		UserID:      userID.String(),
		WorkspaceID: workspaceID.String(),
	}, &resp); err != nil {
		return nil, err
	}
	return &Access{
		HasAccess:         resp.HasAccess,
		Role:              Role(resp.Role),
		Status:            resp.Status,
		WorkspaceArchived: resp.WorkspaceArchived,
	}, nil
}

func (c *Client) ListForUser(ctx context.Context, userID uuid.UUID) ([]UserWorkspace, error) {
	var resp listForUserResponse
	if err := c.postJSON(ctx, "/internal/workspaces/list-for-user", listForUserRequest{
		UserID: userID.String(),
	}, &resp); err != nil {
		return nil, err
	}
	out := make([]UserWorkspace, 0, len(resp.Workspaces))
	for _, item := range resp.Workspaces {
		id, err := uuid.Parse(item.ID)
		if err != nil {
			return nil, fmt.Errorf("decode workspace id: %w", err)
		}
		out = append(out, UserWorkspace{ID: id, Role: Role(item.Role)})
	}
	return out, nil
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, dst any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal workspace request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build workspace request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call workspace endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("workspace endpoint returned HTTP %d", resp.StatusCode)
	}
	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("decode workspace response: %w", err)
		}
	}
	return nil
}

type accessRequest struct {
	UserID      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

type accessResponse struct {
	HasAccess         bool   `json:"hasAccess"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	WorkspaceArchived bool   `json:"workspaceArchived"`
}

type listForUserRequest struct {
	UserID string `json:"userId"`
}

type listForUserResponse struct {
	Workspaces []struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	} `json:"workspaces"`
}
