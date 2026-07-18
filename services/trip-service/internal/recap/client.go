package recap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration) (*HTTPClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("AI planning service URL is required")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("trip recap AI timeout must be greater than zero")
	}
	return &HTTPClient{baseURL: baseURL, client: &http.Client{Timeout: timeout}}, nil
}

func (c *HTTPClient) Generate(ctx context.Context, input GenerateRequest) (GenerateResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("marshal trip recap request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/generate-trip-recap", bytes.NewReader(body))
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("create trip recap request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(req)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("call AI planning service for trip recap: %w", err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 1<<20)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("read trip recap response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return GenerateResponse{}, fmt.Errorf("AI planning trip recap returned status %d", response.StatusCode)
	}
	var out GenerateResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return GenerateResponse{}, fmt.Errorf("decode AI planning trip recap response: %w", err)
	}
	return out, nil
}
