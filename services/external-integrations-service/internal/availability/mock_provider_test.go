package availability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
)

func TestMockProviderReturnsDeterministicAvailableOption(t *testing.T) {
	provider := NewMockAvailabilityProvider()
	input := availabilityInput("Rome", "Colosseum", "attraction")

	first, err := provider.SearchAvailability(context.Background(), input)
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	second, err := provider.SearchAvailability(context.Background(), input)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	if len(first.Options) == 0 {
		t.Fatalf("expected options, got %+v", first)
	}
	if first.Options[0].Price == nil || second.Options[0].Price == nil {
		t.Fatalf("expected prices, got %+v and %+v", first.Options[0], second.Options[0])
	}
	if first.Options[0].Price.Amount != second.Options[0].Price.Amount {
		t.Fatalf("expected deterministic amount, got %v and %v", first.Options[0].Price.Amount, second.Options[0].Price.Amount)
	}
	if !isSafeHTTPURL(first.Options[0].BookingURL) {
		t.Fatalf("expected safe booking URL, got %q", first.Options[0].BookingURL)
	}
}

func TestMockProviderReturnsUnavailableForSoldOutItem(t *testing.T) {
	result, err := NewMockAvailabilityProvider().SearchAvailability(
		context.Background(),
		availabilityInput("Rome", "Sold out museum", "museum"),
	)
	if err != nil {
		t.Fatalf("search unavailable: %v", err)
	}
	if result.Status != StatusUnavailable || result.Result != ProviderResultUnavailable {
		t.Fatalf("expected unavailable result, got %+v", result)
	}
}

func TestMockProviderReturnsUnknownForFreeParkWalk(t *testing.T) {
	result, err := NewMockAvailabilityProvider().SearchAvailability(
		context.Background(),
		availabilityInput("Paris", "Walk through Luxembourg Gardens", "walk"),
	)
	if err != nil {
		t.Fatalf("search park: %v", err)
	}
	if result.Status != StatusUnknown || len(result.Options) != 0 {
		t.Fatalf("expected unknown/no options, got %+v", result)
	}
}

func TestAvailabilityCacheReturnsCachedSecondResponse(t *testing.T) {
	counting := &countingAvailabilityProvider{result: &AvailabilitySearchResult{
		Status:              StatusAvailable,
		Result:              ProviderResultSuccess,
		Provider:            "fake",
		ProviderDisplayName: "Fake Tickets",
		Match:               AvailabilityMatch{Matched: true, Confidence: 0.8},
		Options: []AvailabilityOption{{
			ID:           "fake-1",
			Title:        "Fake entry",
			Availability: StatusAvailable,
			PriceType:    PriceTypePerPerson,
			ProviderName: "Fake Tickets",
			BookingURL:   "https://example.com/book/fake",
		}},
	}}
	provider := newCachingProvider("fake", counting, cache.New(10), time.Minute, zap.NewNop())
	input := availabilityInput("Rome", "Colosseum", "attraction")

	first, err := provider.SearchAvailability(context.Background(), input)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, err := provider.SearchAvailability(context.Background(), input)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if first.Cached {
		t.Fatal("first response should not be cached")
	}
	if !second.Cached {
		t.Fatal("second response should be cached")
	}
	if counting.calls != 1 {
		t.Fatalf("expected one provider call, got %d", counting.calls)
	}
	if second.CacheExpiresAt == nil {
		t.Fatal("expected cacheExpiresAt on cached response")
	}
}

func TestAvailabilityHandlerValidationAndResponseMetadata(t *testing.T) {
	resp := performAvailabilityRequest(newAvailabilityTestRouter(NewMockAvailabilityProvider()), `{
		"destination":"Rome",
		"date":"2026-08-10",
		"currency":"EUR",
		"item":{"name":"Colosseum","type":"attraction","place":{"name":"Colosseum","lat":41.8902,"lng":12.4922}},
		"travelers":{"adults":2}
	}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body AvailabilitySearchResult
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.CheckedAt.IsZero() || len(body.Options) == 0 {
		t.Fatalf("expected checkedAt and options, got %+v", body)
	}
	if !isSafeHTTPURL(body.Options[0].BookingURL) {
		t.Fatalf("unsafe booking URL returned: %q", body.Options[0].BookingURL)
	}
}

func TestAvailabilityHandlerRejectsInvalidDateCurrencyAndCoordinates(t *testing.T) {
	router := newAvailabilityTestRouter(NewMockAvailabilityProvider())
	for _, body := range []string{
		`{"destination":"Rome","date":"2026/08/10","currency":"EUR","item":{"name":"Colosseum"}}`,
		`{"destination":"Rome","date":"2026-08-10","currency":"EU","item":{"name":"Colosseum"}}`,
		`{"destination":"Rome","date":"2026-08-10","currency":"EUR","item":{"name":"Colosseum","place":{"lat":120}}}`,
	} {
		resp := performAvailabilityRequest(router, body)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d body=%s", body, resp.Code, resp.Body.String())
		}
	}
}

func TestAvailabilityHandlerRejectsUnsafeBookingURL(t *testing.T) {
	resp := performAvailabilityRequest(newAvailabilityTestRouter(unsafeURLProvider{}), `{
		"destination":"Rome",
		"date":"2026-08-10",
		"currency":"EUR",
		"item":{"name":"Colosseum","type":"attraction"}
	}`)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%s", resp.Code, resp.Body.String())
	}
}

type countingAvailabilityProvider struct {
	result *AvailabilitySearchResult
	calls  int
}

func (p *countingAvailabilityProvider) Name() string { return "fake" }

func (p *countingAvailabilityProvider) SearchAvailability(context.Context, AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	p.calls++
	out := copyResult(*p.result)
	return &out, nil
}

type unsafeURLProvider struct{}

func (unsafeURLProvider) Name() string { return "unsafe" }

func (unsafeURLProvider) SearchAvailability(context.Context, AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	return &AvailabilitySearchResult{
		Status:              StatusAvailable,
		Result:              ProviderResultSuccess,
		Provider:            "unsafe",
		ProviderDisplayName: "Unsafe",
		Match:               AvailabilityMatch{Matched: true, Confidence: 0.8},
		Options: []AvailabilityOption{{
			ID:           "unsafe-1",
			Title:        "Unsafe option",
			Availability: StatusAvailable,
			PriceType:    PriceTypePerPerson,
			ProviderName: "Unsafe",
			BookingURL:   "javascript:alert(1)",
		}},
	}, nil
}

func newAvailabilityTestRouter(provider AvailabilityProvider) http.Handler {
	r := chi.NewRouter()
	svc := NewService(provider, zap.NewNop(), true)
	NewHandler(svc, zap.NewNop(), "EUR").RegisterRoutes(r)
	return r
}

func performAvailabilityRequest(router http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/availability/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func availabilityInput(destination, name, itemType string) AvailabilitySearchRequest {
	return AvailabilitySearchRequest{
		Destination: destination,
		Date:        "2026-08-10",
		Currency:    "EUR",
		Item: AvailabilityItem{
			Name: name,
			Type: itemType,
			Place: &AvailabilityPlace{
				Name: name,
			},
		},
		Travelers: AvailabilityTravelers{Adults: 1},
	}
}
