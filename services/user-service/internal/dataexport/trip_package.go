package dataexport

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
)

const internalServiceTokenHeader = "X-Internal-Service-Token"

// TripPackageClient fetches a one-time in-memory package over the private
// service network. It sends only the account owner ID and user selections; no
// browser bearer token is forwarded or persisted.
type TripPackageClient struct {
	baseURL  string
	token    string
	maxBytes int64
	client   *http.Client
}

func NewTripPackageClient(baseURL, token string, timeout time.Duration, maxBytes int64) (*TripPackageClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if _, err := url.ParseRequestURI(baseURL); err != nil || baseURL == "" {
		return nil, fmt.Errorf("invalid trip export service URL")
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("trip export service token is required")
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if maxBytes <= 0 {
		maxBytes = 250 * 1024 * 1024
	}
	return &TripPackageClient{baseURL: baseURL, token: token, maxBytes: maxBytes, client: &http.Client{Timeout: timeout}}, nil
}

func (c *TripPackageClient) BuildAccountTripPackage(ctx context.Context, userID uuid.UUID, includeWorkspaceData, includeReceiptFiles bool) ([]byte, error) {
	body, err := json.Marshal(map[string]any{"userId": userID.String(), "includeWorkspaceData": includeWorkspaceData, "includeReceiptFiles": includeReceiptFiles})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/data-exports/account-package", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create trip package request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalServiceTokenHeader, c.token)
	req.Header.Set("X-Internal-Service-Name", "user-service")
	response, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request trip package: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("trip package request failed with status %d", response.StatusCode)
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, c.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read trip package: %w", err)
	}
	if int64(len(contents)) > c.maxBytes {
		return nil, fmt.Errorf("trip package exceeds configured account export limit")
	}
	return contents, nil
}
