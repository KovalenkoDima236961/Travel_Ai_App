package calendar

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

type MockCalendarProvider struct {
	accountEmail string
	linkBase     string
	mu           sync.Mutex
	events       map[string]CalendarEventInput
}

func NewMockCalendarProvider(cfg config.CalendarConfig) *MockCalendarProvider {
	return &MockCalendarProvider{
		accountEmail: cfg.MockAccountEmail,
		linkBase:     cfg.MockEventLinkBase,
		events:       make(map[string]CalendarEventInput),
	}
}

func (p *MockCalendarProvider) BuildAuthURL(_ context.Context, state string) (string, error) {
	u, _ := url.Parse("http://localhost:8084/calendar/google/callback")
	q := u.Query()
	q.Set("code", "mock-code")
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p *MockCalendarProvider) ExchangeCode(_ context.Context, _ string) (*OAuthTokenResult, error) {
	expires := time.Now().UTC().Add(time.Hour)
	return &OAuthTokenResult{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		ExpiresAt:    &expires,
		Scopes:       "https://www.googleapis.com/auth/calendar.events",
	}, nil
}

func (p *MockCalendarProvider) RefreshToken(_ context.Context, _ string) (*OAuthTokenResult, error) {
	expires := time.Now().UTC().Add(time.Hour)
	return &OAuthTokenResult{
		AccessToken: "mock-access-token-refreshed",
		ExpiresAt:   &expires,
		Scopes:      "https://www.googleapis.com/auth/calendar.events",
	}, nil
}

func (p *MockCalendarProvider) GetAccountInfo(_ context.Context, _ string) (*CalendarAccountInfo, error) {
	return &CalendarAccountInfo{Email: p.accountEmail}, nil
}

func (p *MockCalendarProvider) CreateEvent(_ context.Context, _ string, input CalendarEventInput) (*CalendarEventResult, error) {
	id := uuid.NewString()
	p.mu.Lock()
	p.events[id] = input
	p.mu.Unlock()
	return p.result(id), nil
}

func (p *MockCalendarProvider) UpdateEvent(_ context.Context, _ string, calendarID string, eventID string, input CalendarEventInput) (*CalendarEventResult, error) {
	p.mu.Lock()
	p.events[eventID] = input
	p.mu.Unlock()
	result := p.result(eventID)
	if strings.TrimSpace(calendarID) != "" {
		result.CalendarID = calendarID
	}
	return result, nil
}

func (p *MockCalendarProvider) DeleteEvent(_ context.Context, _ string, _ string, eventID string) error {
	p.mu.Lock()
	delete(p.events, eventID)
	p.mu.Unlock()
	return nil
}

func (p *MockCalendarProvider) result(eventID string) *CalendarEventResult {
	link := fmt.Sprintf("%s/%s", strings.TrimRight(p.linkBase, "/"), eventID)
	return &CalendarEventResult{
		CalendarID: "primary",
		EventID:    eventID,
		HtmlLink:   link,
		UpdatedAt:  time.Now().UTC(),
	}
}
