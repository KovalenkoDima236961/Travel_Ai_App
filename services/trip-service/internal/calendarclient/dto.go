package calendarclient

import (
	"time"

	"github.com/google/uuid"
)

type ConnectionStatus struct {
	Connected            bool       `json:"connected"`
	Provider             string     `json:"provider"`
	ProviderAccountEmail *string    `json:"providerAccountEmail,omitempty"`
	ConnectedAt          *time.Time `json:"connectedAt,omitempty"`
	Scopes               *string    `json:"scopes,omitempty"`
}

type FreeBusyRequest struct {
	StartDate   string   `json:"startDate"`
	EndDate     string   `json:"endDate"`
	TimeZone    string   `json:"timezone"`
	CalendarIDs []string `json:"calendarIds,omitempty"`
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

type SyncRequest struct {
	UserID    uuid.UUID  `json:"userId"`
	TripID    uuid.UUID  `json:"tripId"`
	TripTitle string     `json:"tripTitle"`
	TripURL   string     `json:"tripUrl"`
	TimeZone  string     `json:"timeZone"`
	Items     []SyncItem `json:"items"`
}

type SyncItem struct {
	SyncKey            string    `json:"syncKey"`
	DayNumber          int       `json:"dayNumber"`
	ItemIndex          int       `json:"itemIndex"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Location           string    `json:"location"`
	MapURL             string    `json:"mapUrl"`
	Start              time.Time `json:"start"`
	End                time.Time `json:"end"`
	ExistingCalendarID string    `json:"existingCalendarId,omitempty"`
	ExistingEventID    string    `json:"existingEventId,omitempty"`
}

type SyncResult struct {
	Items []SyncItemResult `json:"items"`
}

type SyncItemResult struct {
	SyncKey    string `json:"syncKey"`
	DayNumber  int    `json:"dayNumber"`
	ItemIndex  int    `json:"itemIndex"`
	CalendarID string `json:"calendarId,omitempty"`
	EventID    string `json:"eventId,omitempty"`
	HtmlLink   string `json:"htmlLink,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type DeleteRequest struct {
	UserID uuid.UUID    `json:"userId"`
	Events []DeleteItem `json:"events"`
}

type DeleteItem struct {
	CalendarID string `json:"calendarId"`
	EventID    string `json:"eventId"`
}

type DeleteResult struct {
	Deleted int `json:"deleted"`
	Failed  int `json:"failed"`
}
