package copilot

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

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiprivacy"
)

type AIClient struct {
	baseURL string
	client  *http.Client
}

func NewAIClient(baseURL string, timeoutSeconds int) (*AIClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid AI planning service URL")
	}
	client := &http.Client{}
	if timeoutSeconds > 0 {
		client.Timeout = time.Duration(timeoutSeconds) * time.Second
	}
	return &AIClient{baseURL: baseURL, client: client}, nil
}

func (c *AIClient) Respond(
	ctx context.Context,
	message, language string,
	intent Intent,
	safeContext SafeContext,
	availableActions []Action,
	permissions PermissionSummary,
) (AIResponse, error) {
	if c == nil || c.client == nil {
		return AIResponse{}, fmt.Errorf("copilot AI client unavailable")
	}
	contextRaw, err := json.Marshal(safeContext)
	if err != nil {
		return AIResponse{}, fmt.Errorf("marshal safe context: %w", err)
	}
	contextRaw, _, err = aiprivacy.SanitizeJSON(contextRaw)
	if err != nil {
		return AIResponse{}, fmt.Errorf("sanitize safe context: %w", err)
	}
	var sanitizedContext any
	if err := json.Unmarshal(contextRaw, &sanitizedContext); err != nil {
		return AIResponse{}, fmt.Errorf("decode safe context: %w", err)
	}
	payload := map[string]any{
		"message":           message,
		"language":          language,
		"intent":            intent,
		"safeContext":       sanitizedContext,
		"availableActions":  availableActions,
		"permissionSummary": permissions,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AIResponse{}, fmt.Errorf("marshal copilot request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/copilot/respond", bytes.NewReader(body))
	if err != nil {
		return AIResponse{}, fmt.Errorf("create copilot request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	response, err := c.client.Do(req)
	if err != nil {
		return AIResponse{}, fmt.Errorf("call copilot AI: %w", err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 64*1024)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.ReadAll(limited)
		return AIResponse{}, fmt.Errorf("copilot AI returned status %d", response.StatusCode)
	}
	var result AIResponse
	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		return AIResponse{}, fmt.Errorf("decode copilot AI response: %w", err)
	}
	return result, nil
}
