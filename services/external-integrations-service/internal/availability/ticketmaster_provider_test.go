package availability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

func tmConfig(baseURL string) config.AvailabilityConfig {
	return config.AvailabilityConfig{
		Enabled:                    true,
		Provider:                   config.AvailabilityProviderTicketmaster,
		FallbackToMock:             true,
		TicketmasterAPIKey:         "test-key",
		TicketmasterBaseURL:        baseURL,
		TicketmasterTimeoutSeconds: 5,
		MinMatchConfidence:         0.55,
		LowConfidenceThreshold:     0.65,
		MaxOptions:                 10,
		DefaultCurrency:            "EUR",
	}
}

func newTMTestProvider(t *testing.T, baseURL string) AvailabilityProvider {
	t.Helper()
	provider, err := newTicketmasterProvider(tmConfig(baseURL), zap.NewNop())
	if err != nil {
		t.Fatalf("newTicketmasterProvider: %v", err)
	}
	return provider
}

// tmServer starts a test server that returns the given body/status and records
// whether it was hit, so tests can assert the network was (or was not) used.
func tmServer(t *testing.T, status int, body string) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Query().Get("apikey") == "" {
			t.Errorf("expected apikey query param, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func tmConcertRequest() AvailabilitySearchRequest {
	lat := 48.2072
	lng := 16.4205
	return AvailabilitySearchRequest{
		Destination: "Vienna",
		Date:        "2026-09-10",
		Currency:    "EUR",
		Item: AvailabilityItem{
			Name: "Coldplay Music of the Spheres",
			Type: "concert",
			Place: &AvailabilityPlace{
				Name:      "Ernst Happel Stadion",
				Latitude:  &lat,
				Longitude: &lng,
			},
		},
		Travelers: AvailabilityTravelers{Adults: 2},
	}
}

const tmMatchingEventJSON = `{
  "_embedded": {
    "events": [
      {
        "id": "vvG1zZ9Xabc",
        "name": "Coldplay: Music of the Spheres World Tour",
        "url": "https://www.ticketmaster.com/event/vvG1zZ9Xabc",
        "dates": {"start": {"localDate": "2026-09-10", "localTime": "20:00:00"}, "status": {"code": "onsale"}},
        "priceRanges": [{"type": "standard", "currency": "EUR", "min": 55.0, "max": 180.0}],
        "classifications": [{"segment": {"name": "Music"}, "genre": {"name": "Rock"}}],
        "_embedded": {"venues": [{"name": "Ernst Happel Stadion", "city": {"name": "Vienna"}, "address": {"line1": "Meiereistrasse 7"}, "location": {"latitude": "48.2070", "longitude": "16.4210"}}]}
      }
    ]
  },
  "page": {"totalElements": 1}
}`

func TestTicketmasterMapsMatchingEvent(t *testing.T) {
	srv, hits := tmServer(t, http.StatusOK, tmMatchingEventJSON)
	provider := newTMTestProvider(t, srv.URL)

	result, err := provider.SearchAvailability(context.Background(), tmConcertRequest())
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if *hits != 1 {
		t.Fatalf("expected one provider call, got %d", *hits)
	}
	if result.Status != StatusAvailable {
		t.Fatalf("expected available, got %q", result.Status)
	}
	if !result.Match.Matched || result.Match.Confidence < 0.80 {
		t.Fatalf("expected high-confidence match, got %+v", result.Match)
	}
	if result.Match.ProviderEntityID != "vvG1zZ9Xabc" {
		t.Fatalf("expected provider entity id, got %q", result.Match.ProviderEntityID)
	}
	if len(result.Options) != 1 {
		t.Fatalf("expected one option, got %d", len(result.Options))
	}
	option := result.Options[0]
	if option.Price == nil || option.Price.Amount != 55 || option.Price.Currency != "EUR" {
		t.Fatalf("expected 55 EUR price, got %+v", option.Price)
	}
	if option.Price.Qualifier != PriceQualifierFrom {
		t.Fatalf("expected 'from' qualifier, got %q", option.Price.Qualifier)
	}
	if len(option.StartTimes) != 1 || option.StartTimes[0] != "20:00" {
		t.Fatalf("expected 20:00 start time, got %v", option.StartTimes)
	}
	if option.Date != "2026-09-10" {
		t.Fatalf("expected option date, got %q", option.Date)
	}
	if option.Location == nil || option.Location.Name != "Ernst Happel Stadion" || option.Location.Latitude == nil {
		t.Fatalf("expected venue location, got %+v", option.Location)
	}
	if option.BookingURL != "https://www.ticketmaster.com/event/vvG1zZ9Xabc" {
		t.Fatalf("expected booking url, got %q", option.BookingURL)
	}
}

func TestTicketmasterUnsupportedItemSkipsNetwork(t *testing.T) {
	srv, hits := tmServer(t, http.StatusOK, tmMatchingEventJSON)
	provider := newTMTestProvider(t, srv.URL)

	req := tmConcertRequest()
	req.Item.Name = "Lunch break"
	req.Item.Type = "rest"

	result, err := provider.SearchAvailability(context.Background(), req)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if *hits != 0 {
		t.Fatalf("expected no provider call for unsupported type, got %d", *hits)
	}
	if result.Status != StatusUnknown || len(result.Options) != 0 {
		t.Fatalf("expected unknown/no options, got %+v", result)
	}
	if !containsWarning(result.Warnings, "does not support this item type") {
		t.Fatalf("expected unsupported warning, got %v", result.Warnings)
	}
}

func TestTicketmasterNoEventsReturnsUnknown(t *testing.T) {
	srv, _ := tmServer(t, http.StatusOK, `{"_embedded":{"events":[]},"page":{"totalElements":0}}`)
	provider := newTMTestProvider(t, srv.URL)

	result, err := provider.SearchAvailability(context.Background(), tmConcertRequest())
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Status != StatusUnknown || result.Result != ProviderResultNoMatch {
		t.Fatalf("expected unknown/no_match, got %+v", result)
	}
}

func TestTicketmasterCancelledEventIsUnavailable(t *testing.T) {
	body := strings.Replace(tmMatchingEventJSON, `"code": "onsale"`, `"code": "canceled"`, 1)
	srv, _ := tmServer(t, http.StatusOK, body)
	provider := newTMTestProvider(t, srv.URL)

	result, err := provider.SearchAvailability(context.Background(), tmConcertRequest())
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Status != StatusUnavailable {
		t.Fatalf("expected unavailable, got %q", result.Status)
	}
	if result.Options[0].Availability != StatusUnavailable {
		t.Fatalf("expected option unavailable, got %q", result.Options[0].Availability)
	}
}

func TestTicketmasterLowConfidenceIsUnknown(t *testing.T) {
	// Event that shares nothing with the request: different name/city/date.
	body := `{"_embedded":{"events":[{"id":"x1","name":"Local Poetry Reading","url":"https://www.ticketmaster.com/event/x1","dates":{"start":{"localDate":"2027-01-01"},"status":{"code":"onsale"}}}]},"page":{"totalElements":1}}`
	srv, _ := tmServer(t, http.StatusOK, body)
	provider := newTMTestProvider(t, srv.URL)

	result, err := provider.SearchAvailability(context.Background(), tmConcertRequest())
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Status != StatusUnknown {
		t.Fatalf("expected unknown for low confidence, got %q (confidence %v)", result.Status, result.Match.Confidence)
	}
	if result.Match.Matched {
		t.Fatalf("expected match=false for low confidence, got %+v", result.Match)
	}
	if !containsWarning(result.Warnings, "Possible match") {
		t.Fatalf("expected possible-match warning, got %v", result.Warnings)
	}
	if len(result.Options) == 0 {
		t.Fatalf("expected low-confidence candidate options to still be returned")
	}
}

func TestTicketmasterMissingPriceAndVenue(t *testing.T) {
	body := `{"_embedded":{"events":[{"id":"vvG1","name":"Coldplay Music of the Spheres","url":"https://www.ticketmaster.com/event/vvG1","dates":{"start":{"localDate":"2026-09-10","localTime":"20:00:00"},"status":{"code":"onsale"}},"classifications":[{"segment":{"name":"Music"}}]}]},"page":{"totalElements":1}}`
	srv, _ := tmServer(t, http.StatusOK, body)
	provider := newTMTestProvider(t, srv.URL)

	result, err := provider.SearchAvailability(context.Background(), tmConcertRequest())
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Options) != 1 {
		t.Fatalf("expected one option, got %d", len(result.Options))
	}
	if result.Options[0].Price != nil {
		t.Fatalf("expected nil price when provider omits priceRanges, got %+v", result.Options[0].Price)
	}
	if result.Options[0].Location != nil {
		t.Fatalf("expected nil location when provider omits venue, got %+v", result.Options[0].Location)
	}
}

func TestTicketmasterClientClassifiesHTTPErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		kind   string
	}{
		{"auth", http.StatusUnauthorized, providerErrorAuthConfig},
		{"forbidden", http.StatusForbidden, providerErrorAuthConfig},
		{"rate_limited", http.StatusTooManyRequests, providerErrorRateLimit},
		{"server_error", http.StatusBadGateway, providerErrorUnavailable},
		{"bad_request", http.StatusBadRequest, providerErrorBadResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := tmServer(t, tc.status, `{}`)
			client := newTicketmasterClient("k", srv.URL, 5*time.Second, zap.NewNop())
			_, err := client.searchEvents(context.Background(), url.Values{})
			assertProviderErrorKind(t, err, tc.kind)
		})
	}
}

func TestTicketmasterClientClassifiesMalformedResponse(t *testing.T) {
	srv, _ := tmServer(t, http.StatusOK, `{not valid json`)
	client := newTicketmasterClient("k", srv.URL, 5*time.Second, zap.NewNop())
	_, err := client.searchEvents(context.Background(), url.Values{})
	assertProviderErrorKind(t, err, providerErrorBadResponse)
}

func TestTicketmasterClientClassifiesTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	client := newTicketmasterClient("k", srv.URL, time.Millisecond, zap.NewNop())
	_, err := client.searchEvents(context.Background(), url.Values{})
	assertProviderErrorKind(t, err, providerErrorTimeout)
}

func TestTicketmasterClient404IsEmptyResult(t *testing.T) {
	srv, _ := tmServer(t, http.StatusNotFound, `{}`)
	client := newTicketmasterClient("k", srv.URL, 5*time.Second, zap.NewNop())
	payload, err := client.searchEvents(context.Background(), url.Values{})
	if err != nil {
		t.Fatalf("expected nil error for 404, got %v", err)
	}
	if len(payload.Embedded.Events) != 0 {
		t.Fatalf("expected empty events, got %d", len(payload.Embedded.Events))
	}
}

func TestScoreTicketmasterEventHighForExactMatch(t *testing.T) {
	req := tmConcertRequest()
	event := tmEvent{
		Name:            "Coldplay: Music of the Spheres World Tour",
		Dates:           tmDates{Start: tmDateStart{LocalDate: "2026-09-10"}, Status: tmDateStatus{Code: "onsale"}},
		Classifications: []tmClassification{{Segment: tmNamed{Name: "Music"}}},
		Embedded: tmEventEmbedded{Venues: []tmVenue{{
			Name:     "Ernst Happel Stadion",
			City:     tmNamed{Name: "Vienna"},
			Location: tmGeoLocation{Latitude: "48.2071", Longitude: "16.4208"},
		}}},
	}
	score := scoreTicketmasterEvent(req, event)
	if score.total < 0.80 {
		t.Fatalf("expected high confidence, got %v (%+v)", score.total, score)
	}
	if score.titleScore == 0 || score.venueScore == 0 || score.cityScore == 0 || score.dateScore == 0 {
		t.Fatalf("expected all core signals to contribute, got %+v", score)
	}
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}

func assertProviderErrorKind(t *testing.T, err error, kind string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error of kind %q, got nil", kind)
	}
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected *ProviderError, got %T: %v", err, err)
	}
	if providerErr.Kind != kind {
		t.Fatalf("expected kind %q, got %q", kind, providerErr.Kind)
	}
}
