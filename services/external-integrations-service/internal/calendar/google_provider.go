package calendar

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

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

type GoogleCalendarProvider struct {
	client       *http.Client
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	authURL      string
	tokenURL     string
	userInfoURL  string
	calendarAPI  string
}

func NewGoogleCalendarProvider(cfg config.CalendarConfig) *GoogleCalendarProvider {
	return &GoogleCalendarProvider{
		client:       &http.Client{Timeout: 15 * time.Second},
		clientID:     cfg.GoogleClientID,
		clientSecret: cfg.GoogleClientSecret,
		redirectURL:  cfg.GoogleRedirectURL,
		scopes:       cfg.Scopes(),
		authURL:      cfg.GoogleAuthURL,
		tokenURL:     cfg.GoogleTokenURL,
		userInfoURL:  cfg.GoogleUserInfoURL,
		calendarAPI:  cfg.GoogleCalendarAPI,
	}
}

func (p *GoogleCalendarProvider) BuildAuthURL(_ context.Context, state string) (string, error) {
	u, err := url.Parse(p.authURL)
	if err != nil {
		return "", fmt.Errorf("parse auth url: %w", err)
	}
	q := u.Query()
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(p.scopes, " "))
	q.Set("state", state)
	q.Set("access_type", "offline")
	q.Set("include_granted_scopes", "true")
	q.Set("prompt", "consent")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p *GoogleCalendarProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokenResult, error) {
	values := url.Values{}
	values.Set("code", code)
	values.Set("client_id", p.clientID)
	values.Set("client_secret", p.clientSecret)
	values.Set("redirect_uri", p.redirectURL)
	values.Set("grant_type", "authorization_code")
	return p.tokenRequest(ctx, values)
}

func (p *GoogleCalendarProvider) RefreshToken(ctx context.Context, refreshToken string) (*OAuthTokenResult, error) {
	values := url.Values{}
	values.Set("refresh_token", refreshToken)
	values.Set("client_id", p.clientID)
	values.Set("client_secret", p.clientSecret)
	values.Set("grant_type", "refresh_token")
	return p.tokenRequest(ctx, values)
}

func (p *GoogleCalendarProvider) tokenRequest(ctx context.Context, values url.Values) (*OAuthTokenResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google token request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google token request failed with status %d", resp.StatusCode)
	}

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode google token response: %w", err)
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("google token response missing access token")
	}
	var expiresAt *time.Time
	if out.ExpiresIn > 0 {
		t := time.Now().UTC().Add(time.Duration(out.ExpiresIn) * time.Second)
		expiresAt = &t
	}
	return &OAuthTokenResult{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       out.Scope,
	}, nil
}

func (p *GoogleCalendarProvider) GetAccountInfo(ctx context.Context, accessToken string) (*CalendarAccountInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build account request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google account request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google account request failed with status %d", resp.StatusCode)
	}

	var out struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode google account response: %w", err)
	}
	return &CalendarAccountInfo{Email: out.Email}, nil
}

func (p *GoogleCalendarProvider) GetFreeBusy(ctx context.Context, accessToken string, input ProviderFreeBusyRequest) ([]FreeBusyBlock, error) {
	calendarIDs := input.CalendarIDs
	if len(calendarIDs) == 0 {
		calendarIDs = []string{"primary"}
	}
	items := make([]map[string]string, 0, len(calendarIDs))
	for _, id := range calendarIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			trimmed = "primary"
		}
		items = append(items, map[string]string{"id": trimmed})
	}
	body := map[string]any{
		"timeMin":  input.Start.Format(time.RFC3339),
		"timeMax":  input.End.Format(time.RFC3339),
		"timeZone": input.TimeZone,
		"items":    items,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode google freebusy request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.calendarAPI+"/freeBusy", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build freebusy request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google freebusy request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, ErrCalendarReauthRequired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("google freebusy request failed with status %d", resp.StatusCode)
	}

	var out struct {
		Calendars map[string]struct {
			Busy []struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"busy"`
			Errors []struct {
				Domain string `json:"domain"`
				Reason string `json:"reason"`
			} `json:"errors"`
		} `json:"calendars"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: decode google freebusy response: %v", ErrCalendarFreeBusyMalformedResponse, err)
	}
	blocks := make([]FreeBusyBlock, 0)
	for _, calendarResult := range out.Calendars {
		if len(calendarResult.Errors) > 0 {
			return nil, ErrCalendarFreeBusyUnavailable
		}
		for _, busy := range calendarResult.Busy {
			start, allDayStart, err := parseGoogleBusyTime(busy.Start, input.TimeZone)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid busy start", ErrCalendarFreeBusyMalformedResponse)
			}
			end, allDayEnd, err := parseGoogleBusyTime(busy.End, input.TimeZone)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid busy end", ErrCalendarFreeBusyMalformedResponse)
			}
			if !end.After(start) {
				continue
			}
			blocks = append(blocks, FreeBusyBlock{
				Start:  start,
				End:    end,
				AllDay: allDayStart && allDayEnd || looksAllDay(start, end),
				Source: "google_calendar",
			})
		}
	}
	return blocks, nil
}

func (p *GoogleCalendarProvider) CreateEvent(ctx context.Context, accessToken string, input CalendarEventInput) (*CalendarEventResult, error) {
	return p.writeEvent(ctx, http.MethodPost, accessToken, "primary", "", input)
}

func (p *GoogleCalendarProvider) UpdateEvent(ctx context.Context, accessToken string, calendarID string, eventID string, input CalendarEventInput) (*CalendarEventResult, error) {
	if strings.TrimSpace(calendarID) == "" {
		calendarID = "primary"
	}
	return p.writeEvent(ctx, http.MethodPut, accessToken, calendarID, eventID, input)
}

func (p *GoogleCalendarProvider) DeleteEvent(ctx context.Context, accessToken string, calendarID string, eventID string) error {
	if strings.TrimSpace(calendarID) == "" {
		calendarID = "primary"
	}
	endpoint := fmt.Sprintf("%s/calendars/%s/events/%s", p.calendarAPI, url.PathEscape(calendarID), url.PathEscape(eventID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build delete event request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("google delete event request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("google delete event failed with status %d", resp.StatusCode)
	}
	return nil
}

func (p *GoogleCalendarProvider) writeEvent(ctx context.Context, method, accessToken, calendarID, eventID string, input CalendarEventInput) (*CalendarEventResult, error) {
	endpoint := fmt.Sprintf("%s/calendars/%s/events", p.calendarAPI, url.PathEscape(calendarID))
	if eventID != "" {
		endpoint += "/" + url.PathEscape(eventID)
	}
	body := googleEventBody(input)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode google event: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build event request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google event request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("google event request failed with status %d", resp.StatusCode)
	}
	var out struct {
		ID       string `json:"id"`
		HtmlLink string `json:"htmlLink"`
		Updated  string `json:"updated"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode google event response: %w", err)
	}
	updatedAt := time.Now().UTC()
	if out.Updated != "" {
		if parsed, err := time.Parse(time.RFC3339, out.Updated); err == nil {
			updatedAt = parsed
		}
	}
	return &CalendarEventResult{
		CalendarID: calendarID,
		EventID:    out.ID,
		HtmlLink:   out.HtmlLink,
		UpdatedAt:  updatedAt,
	}, nil
}

func googleEventBody(input CalendarEventInput) map[string]any {
	timeZone := strings.TrimSpace(input.TimeZone)
	if timeZone == "" {
		timeZone = "UTC"
	}
	return map[string]any{
		"summary":     input.Title,
		"description": input.Description,
		"location":    input.Location,
		"start": map[string]string{
			"dateTime": input.Start.Format(time.RFC3339),
			"timeZone": timeZone,
		},
		"end": map[string]string{
			"dateTime": input.End.Format(time.RFC3339),
			"timeZone": timeZone,
		},
	}
}

func parseGoogleBusyTime(value, timezone string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, fmt.Errorf("empty time")
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, false, nil
	}
	loc, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil || loc == nil {
		loc = time.UTC
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, loc)
	if err != nil {
		return time.Time{}, false, err
	}
	return parsed, true, nil
}

func looksAllDay(start, end time.Time) bool {
	localStart := start
	localEnd := end.In(localStart.Location())
	if localStart.Hour() != 0 || localStart.Minute() != 0 || localStart.Second() != 0 || localStart.Nanosecond() != 0 {
		return false
	}
	if localEnd.Hour() != 0 || localEnd.Minute() != 0 || localEnd.Second() != 0 || localEnd.Nanosecond() != 0 {
		return false
	}
	return localEnd.Sub(localStart) >= 24*time.Hour
}
