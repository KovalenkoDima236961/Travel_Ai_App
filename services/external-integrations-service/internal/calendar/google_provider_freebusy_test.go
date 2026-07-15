package calendar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

func TestGoogleProviderGetFreeBusyUsesSanitizedFreeBusyAPI(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/freeBusy" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"calendars": map[string]any{
				"primary": map[string]any{
					"busy": []map[string]string{
						{
							"start": "2026-09-12T09:00:00+02:00",
							"end":   "2026-09-12T11:00:00+02:00",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	provider := NewGoogleCalendarProvider(config.CalendarConfig{
		GoogleCalendarAPI: server.URL,
	})
	blocks, err := provider.GetFreeBusy(context.Background(), "access-token", ProviderFreeBusyRequest{
		Start:       time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		TimeZone:    "Europe/Bratislava",
		CalendarIDs: []string{"primary"},
	})
	if err != nil {
		t.Fatalf("GetFreeBusy returned error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected one block, got %d", len(blocks))
	}
	if blocks[0].Source != "google_calendar" {
		t.Fatalf("unexpected source: %s", blocks[0].Source)
	}
	payload, _ := json.Marshal(requestBody)
	payloadText := string(payload)
	for _, forbidden := range []string{"events", "summary", "description", "attendees", "location"} {
		if strings.Contains(payloadText, forbidden) {
			t.Fatalf("freebusy request leaked event field %q in %s", forbidden, payloadText)
		}
	}
}

func TestGoogleProviderGetFreeBusyRejectsCalendarErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"calendars": map[string]any{
				"primary": map[string]any{
					"errors": []map[string]string{{"reason": "notFound"}},
				},
			},
		})
	}))
	defer server.Close()

	provider := NewGoogleCalendarProvider(config.CalendarConfig{GoogleCalendarAPI: server.URL})
	_, err := provider.GetFreeBusy(context.Background(), "access-token", ProviderFreeBusyRequest{
		Start:       time.Now(),
		End:         time.Now().Add(24 * time.Hour),
		TimeZone:    "UTC",
		CalendarIDs: []string{"primary"},
	})
	if err != ErrCalendarFreeBusyUnavailable {
		t.Fatalf("expected ErrCalendarFreeBusyUnavailable, got %v", err)
	}
}
