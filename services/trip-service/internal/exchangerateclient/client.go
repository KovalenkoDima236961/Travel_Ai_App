package exchangerateclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

const internalServiceTokenHeader = "X-Internal-Service-Token"

type Config struct {
	BaseURL        string
	Token          string
	TimeoutSeconds int
}

type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("external integrations service url is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid external integrations service url")
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 8
	}
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(cfg.Token),
		client: observability.InstrumentHTTPClient(&http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		}),
	}, nil
}

func (c *Client) Latest(ctx context.Context, base string) (*ExchangeRateTable, error) {
	values := url.Values{}
	values.Set("base", strings.ToUpper(strings.TrimSpace(base)))
	var out ExchangeRateTable
	if err := c.get(ctx, "/exchange-rates/latest", values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Convert(ctx context.Context, amount float64, from string, to string) (*CurrencyConversionResult, error) {
	values := url.Values{}
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("from", strings.ToUpper(strings.TrimSpace(from)))
	values.Set("to", strings.ToUpper(strings.TrimSpace(to)))
	var out CurrencyConversionResult
	if err := c.get(ctx, "/exchange-rates/convert", values, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) get(ctx context.Context, path string, values url.Values, output any) error {
	endpoint := c.baseURL + path
	if len(values) > 0 {
		endpoint += "?" + values.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build exchange rate request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set(internalServiceTokenHeader, c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return &Error{Code: "conversion_unavailable", StatusCode: 0}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var body struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		code := normalizeErrorCode(body.Error, resp.StatusCode)
		return &Error{StatusCode: resp.StatusCode, Code: code}
	}

	if output != nil {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
			return &Error{StatusCode: resp.StatusCode, Code: "conversion_unavailable"}
		}
	}
	return nil
}

type Error struct {
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	return e.Code
}

func (e *Error) Reason() string {
	if e == nil || strings.TrimSpace(e.Code) == "" {
		return "conversion_unavailable"
	}
	return e.Code
}

func normalizeErrorCode(raw string, status int) string {
	switch strings.TrimSpace(raw) {
	case "unsupported_currency":
		return "unsupported_currency"
	case "exchange_rate_provider_unavailable":
		return "provider_unavailable"
	}
	if status == http.StatusBadRequest {
		return "conversion_unavailable"
	}
	return "provider_unavailable"
}
