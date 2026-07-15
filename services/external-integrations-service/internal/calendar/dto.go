package calendar

import (
	"time"

	"github.com/google/uuid"
)

const ProviderGoogle = "google"

type CalendarConnection struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	Provider              string
	ProviderAccountEmail  *string
	AccessTokenEncrypted  string
	RefreshTokenEncrypted *string
	TokenExpiresAt        *time.Time
	Scopes                *string
	ConnectedAt           time.Time
	UpdatedAt             time.Time
	DisconnectedAt        *time.Time
	Status                string
}

type OAuthState struct {
	State     string
	UserID    uuid.UUID
	Provider  string
	ReturnURL *string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type OAuthTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	Scopes       string
}

type CalendarAccountInfo struct {
	Email string
}

type CalendarEventInput struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	TimeZone    string    `json:"timeZone"`
}

type CalendarEventResult struct {
	CalendarID string    `json:"calendarId"`
	EventID    string    `json:"eventId"`
	HtmlLink   string    `json:"htmlLink"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type FreeBusyRequest struct {
	StartDate   string   `json:"startDate"`
	EndDate     string   `json:"endDate"`
	TimeZone    string   `json:"timezone"`
	CalendarIDs []string `json:"calendarIds,omitempty"`
}

type ProviderFreeBusyRequest struct {
	Start       time.Time
	End         time.Time
	TimeZone    string
	CalendarIDs []string
}

type FreeBusyBlock struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	AllDay bool      `json:"allDay"`
	Source string    `json:"source"`
}

type FreeBusySummary struct {
	StartDate         string `json:"startDate"`
	EndDate           string `json:"endDate"`
	TimeZone          string `json:"timezone"`
	BusyBlockCount    int    `json:"busyBlockCount"`
	BusyDays          int    `json:"busyDays"`
	FullyBusyDays     int    `json:"fullyBusyDays"`
	PartiallyBusyDays int    `json:"partiallyBusyDays"`
	CalendarCount     int    `json:"calendarCount"`
}

type FreeBusyResponse struct {
	BusyBlocks []FreeBusyBlock `json:"busyBlocks"`
	Summary    FreeBusySummary `json:"summary"`
	Warnings   []string        `json:"warnings"`
}

type ConnectionStatus struct {
	Connected            bool       `json:"connected"`
	Provider             string     `json:"provider"`
	ProviderAccountEmail *string    `json:"providerAccountEmail,omitempty"`
	ConnectedAt          *time.Time `json:"connectedAt,omitempty"`
	Scopes               *string    `json:"scopes,omitempty"`
}

type SyncEventsRequest struct {
	UserID    uuid.UUID       `json:"userId"`
	TripID    uuid.UUID       `json:"tripId"`
	TripTitle string          `json:"tripTitle"`
	TripURL   string          `json:"tripUrl"`
	TimeZone  string          `json:"timeZone"`
	Items     []SyncEventItem `json:"items"`
}

type SyncEventItem struct {
	SyncKey            string    `json:"syncKey"`
	DayNumber          int       `json:"dayNumber"`
	ItemIndex          int       `json:"itemIndex"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Location           string    `json:"location"`
	MapURL             string    `json:"mapUrl"`
	Start              time.Time `json:"start"`
	End                time.Time `json:"end"`
	ExistingCalendarID string    `json:"existingCalendarId"`
	ExistingEventID    string    `json:"existingEventId"`
}

type SyncEventsResponse struct {
	Items []SyncEventItemResult `json:"items"`
}

type SyncEventItemResult struct {
	SyncKey    string `json:"syncKey"`
	DayNumber  int    `json:"dayNumber"`
	ItemIndex  int    `json:"itemIndex"`
	CalendarID string `json:"calendarId,omitempty"`
	EventID    string `json:"eventId,omitempty"`
	HtmlLink   string `json:"htmlLink,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type DeleteEventsRequest struct {
	UserID uuid.UUID         `json:"userId"`
	Events []DeleteEventItem `json:"events"`
}

type DeleteEventItem struct {
	CalendarID string `json:"calendarId"`
	EventID    string `json:"eventId"`
}

type DeleteEventsResponse struct {
	Deleted int `json:"deleted"`
	Failed  int `json:"failed"`
}
