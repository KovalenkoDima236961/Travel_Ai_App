package users

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

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/observability"
)

const maxErrorBodyBytes = 2 * 1024

// internalServiceTokenHeader is the header Auth Service requires on internal
// endpoints. Kept local so this service has no source dependency on Auth
// Service's module.
const internalServiceTokenHeader = "X-Internal-Service-Token"

// Config configures the user-lookup client.
type Config struct {
	// BaseURL is the service that owns recipient email (Auth Service in v1).
	BaseURL string
	// Token is the shared internal service token presented on the lookup call.
	Token string
	// TimeoutSeconds bounds each lookup request.
	TimeoutSeconds int
}

// Client resolves recipient identities via the Auth Service internal batch
// endpoint. The internal service token is sent in a header and never logged.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New constructs a client with a validated base URL and internal service token.
func New(cfg Config) (*Client, error) {
	normalized, err := normalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("user lookup internal service token is required")
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 5
	}
	return &Client{
		baseURL: normalized,
		token:   strings.TrimSpace(cfg.Token),
		httpClient: observability.InstrumentHTTPClient(&http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		}),
	}, nil
}

// LookupByIDs resolves the given user ids to profiles, returning a map keyed by
// user id. Ids with no matching account are simply absent from the map; the
// caller decides how to handle a partial result. An empty input is a no-op
// (no HTTP call). A transport error or non-2xx response is returned to the
// caller.
func (c *Client) LookupByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]UserProfile, error) {
	out := make(map[uuid.UUID]UserProfile, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	payload := batchRequest{UserIDs: make([]string, 0, len(ids))}
	for _, id := range ids {
		payload.UserIDs = append(payload.UserIDs, id.String())
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal user lookup batch: %w", err)
	}

	endpoint, err := url.JoinPath(c.baseURL, "/internal/users/batch")
	if err != nil {
		return nil, fmt.Errorf("build user lookup endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create user lookup request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call user lookup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("user lookup returned status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var decoded batchResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode user lookup response: %w", err)
	}

	for _, item := range decoded.Items {
		id, err := uuid.Parse(strings.TrimSpace(item.UserID))
		if err != nil {
			// Skip a malformed id rather than failing the whole lookup.
			continue
		}
		out[id] = UserProfile{
			UserID:      id,
			Email:       strings.TrimSpace(item.Email),
			DisplayName: strings.TrimSpace(item.DisplayName),
		}
	}
	return out, nil
}

func readErrorBody(body io.Reader) string {
	limited, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
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
		return "", fmt.Errorf("user lookup base URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid user lookup base URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid user lookup base URL: scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid user lookup base URL: host is required")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}
