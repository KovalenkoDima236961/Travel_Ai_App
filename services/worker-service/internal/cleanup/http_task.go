package cleanup

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
)

const internalTokenHeader = "X-Internal-Service-Token"

type HTTPTask struct {
	descriptor Descriptor
	baseURL    string
	token      string
	client     *http.Client
}

func NewHTTPTask(descriptor Descriptor, baseURL, token string, timeout time.Duration) *HTTPTask {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &HTTPTask{descriptor: descriptor, baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), token: token, client: &http.Client{Timeout: timeout}}
}

func (t *HTTPTask) Name() string           { return t.descriptor.Name }
func (t *HTTPTask) Descriptor() Descriptor { return t.descriptor }

func (t *HTTPTask) Run(ctx context.Context, params Params) (Result, error) {
	result := Result{TaskName: t.Name(), DryRun: params.DryRun}
	if t.baseURL == "" || strings.TrimSpace(t.token) == "" {
		return result, fmt.Errorf("cleanup task is not configured")
	}
	body, err := json.Marshal(params)
	if err != nil {
		return result, fmt.Errorf("encode cleanup request: %w", err)
	}
	endpoint := t.baseURL + "/internal/cleanup/" + url.PathEscape(t.Name())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return result, fmt.Errorf("build cleanup request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, t.token)
	req.Header.Set("X-Internal-Service-Name", "worker-service")
	if params.RequestID != "" {
		req.Header.Set("X-Request-ID", params.RequestID)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("call cleanup task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return result, fmt.Errorf("cleanup task returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return Result{TaskName: t.Name(), DryRun: params.DryRun}, fmt.Errorf("decode cleanup result: %w", err)
	}
	result.TaskName = t.Name()
	result.DryRun = params.DryRun
	return result, nil
}
