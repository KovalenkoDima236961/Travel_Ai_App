package tripdiscovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

const maxErrorBody = 4 * 1024

type AIClient interface {
	SuggestDestinations(context.Context, AIRequest) (*SuggestionResponse, error)
}

type HTTPAIClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPAIClient(baseURL string, timeout time.Duration) (*HTTPAIClient, error) {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return nil, fmt.Errorf("AI_PLANNING_SERVICE_URL is required")
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &HTTPAIClient{
		baseURL: normalized,
		client:  observability.InstrumentHTTPClient(&http.Client{Timeout: timeout}),
	}, nil
}

func (c *HTTPAIClient) SuggestDestinations(
	ctx context.Context,
	input AIRequest,
) (*SuggestionResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal destination suggestion request: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/suggest-destinations",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build destination suggestion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call destination suggestion endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return nil, fmt.Errorf(
			"destination suggestion endpoint returned HTTP %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(raw)),
		)
	}
	var result SuggestionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode destination suggestion response: %w", err)
	}
	if len(result.Suggestions) == 0 {
		return nil, fmt.Errorf("destination suggestion endpoint returned no suggestions")
	}
	return &result, nil
}
