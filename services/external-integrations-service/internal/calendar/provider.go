package calendar

import "context"

type CalendarProvider interface {
	BuildAuthURL(ctx context.Context, state string) (string, error)
	ExchangeCode(ctx context.Context, code string) (*OAuthTokenResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (*OAuthTokenResult, error)
	GetAccountInfo(ctx context.Context, accessToken string) (*CalendarAccountInfo, error)
	CreateEvent(ctx context.Context, accessToken string, input CalendarEventInput) (*CalendarEventResult, error)
	UpdateEvent(ctx context.Context, accessToken string, calendarID string, eventID string, input CalendarEventInput) (*CalendarEventResult, error)
	DeleteEvent(ctx context.Context, accessToken string, calendarID string, eventID string) error
}
