package usercontext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

const maxUserContextErrorBodyBytes = 2 * 1024

// ErrorType classifies User Service failures for logging and fail-open logic.
type ErrorType string

const (
	ErrorTypeAuth        ErrorType = "auth"
	ErrorTypeMissing     ErrorType = "missing"
	ErrorTypeService     ErrorType = "service"
	ErrorTypeInvalidJSON ErrorType = "invalid_json"
)

// Error is returned for non-success User Service responses and malformed
// payloads. It never includes the forwarded access token.
type Error struct {
	Type       ErrorType
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("user context %s error: status %d: %s", e.Type, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("user context %s error: %s", e.Type, e.Message)
}

// Client calls User Service profile and preferences endpoints.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient constructs a client with a validated base URL and caller-provided
// HTTP client.
func NewClient(baseURL string, client *http.Client) (*Client, error) {
	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("user context http client is required")
	}
	return &Client{baseURL: normalized, client: observability.InstrumentHTTPClient(client)}, nil
}

// GetMyProfile fetches the authenticated user's profile from User Service.
func (c *Client) GetMyProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	var profile UserProfile
	found, err := c.get(ctx, "/users/me/profile", accessToken, &profile)
	if err != nil || !found {
		return nil, err
	}
	return &profile, nil
}

// GetMyPreferences fetches the authenticated user's preferences from User
// Service.
func (c *Client) GetMyPreferences(ctx context.Context, accessToken string) (*UserPreferences, error) {
	var preferences UserPreferences
	found, err := c.get(ctx, "/users/me/preferences", accessToken, &preferences)
	if err != nil || !found {
		return nil, err
	}
	return &preferences, nil
}

// GetUserContext fetches profile and preferences. Missing profile/preferences
// are tolerated as partial context; auth/service/JSON errors are returned.
func (c *Client) GetUserContext(ctx context.Context, accessToken string) (*UserContext, error) {
	profile, err := c.GetMyProfile(ctx, accessToken)
	if err != nil && !isMissing(err) {
		return nil, err
	}

	preferences, err := c.GetMyPreferences(ctx, accessToken)
	if err != nil && !isMissing(err) {
		return nil, err
	}

	return &UserContext{
		Profile:     profile,
		Preferences: preferences,
	}, nil
}

func (c *Client) get(ctx context.Context, path, accessToken string, out any) (bool, error) {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return false, &Error{Type: ErrorTypeAuth, Message: "missing access token"}
	}

	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return false, fmt.Errorf("build user service endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("create user service request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, &Error{Type: ErrorTypeService, Message: err.Error()}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return false, &Error{Type: ErrorTypeInvalidJSON, StatusCode: resp.StatusCode, Message: err.Error()}
		}
		return true, nil
	case resp.StatusCode == http.StatusNotFound:
		return false, &Error{Type: ErrorTypeMissing, StatusCode: resp.StatusCode, Message: "context not found"}
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return false, &Error{Type: ErrorTypeAuth, StatusCode: resp.StatusCode, Message: readErrorBody(resp.Body)}
	case resp.StatusCode >= http.StatusInternalServerError:
		return false, &Error{Type: ErrorTypeService, StatusCode: resp.StatusCode, Message: readErrorBody(resp.Body)}
	default:
		return false, &Error{Type: ErrorTypeService, StatusCode: resp.StatusCode, Message: readErrorBody(resp.Body)}
	}
}

func isMissing(err error) bool {
	var userContextErr *Error
	return errors.As(err, &userContextErr) && userContextErr.Type == ErrorTypeMissing
}

func readErrorBody(body io.Reader) string {
	limited, err := io.ReadAll(io.LimitReader(body, maxUserContextErrorBodyBytes))
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
		return "", fmt.Errorf("USER_SERVICE_URL is required when user context is enabled")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid USER_SERVICE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid USER_SERVICE_URL: scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid USER_SERVICE_URL: host is required")
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}
