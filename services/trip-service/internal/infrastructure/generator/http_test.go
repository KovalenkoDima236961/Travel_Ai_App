package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestAIPlanningHTTPGeneratorGenerate_Success(t *testing.T) {
	var captured aiPlanningGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate-itinerary" {
			t.Errorf("expected path /generate-itinerary, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"days": [
				{
					"day": 1,
					"title": "Historic center and local food",
					"items": [
						{
							"time": "09:00",
							"type": "place",
							"name": "Colosseum",
							"note": "Start early.",
							"estimatedCost": 18
						}
					]
				}
			]
		}`)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Second
	gen := newTestHTTPGenerator(t, server.URL, client)

	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	budget := 600.0
	tripID := uuid.New()
	got, err := gen.Generate(context.Background(), entity.Trip{
		ID:             tripID,
		Destination:    "Rome",
		StartDate:      &startDate,
		Days:           4,
		BudgetAmount:   &budget,
		BudgetCurrency: "EUR",
		Travelers:      2,
		Interests:      []string{"food", "history", "hidden_gems"},
		Pace:           "balanced",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil || len(got.Days) != 1 {
		t.Fatalf("expected one itinerary day, got %+v", got)
	}
	if got.Days[0].Items[0].EstimatedCost == nil || *got.Days[0].Items[0].EstimatedCost != 18 {
		t.Fatalf("expected estimated cost to decode, got %+v", got.Days[0].Items[0].EstimatedCost)
	}
	if got.Destination != "Rome" || got.Source != "ai-planning-service-http" {
		t.Fatalf("expected enriched itinerary metadata, got %+v", got)
	}

	assertCapturedPayload(t, captured, tripID.String(), "2026-08-10", &budget)
}

func TestAIPlanningHTTPGeneratorGenerate_DefaultsRequestPayload(t *testing.T) {
	var captured aiPlanningGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
		}
		fmt.Fprint(w, `{"days":[{"day":1,"title":"Day","items":[]} ]}`)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Second
	gen := newTestHTTPGenerator(t, server.URL, client)

	tripID := uuid.New()
	_, err := gen.Generate(context.Background(), entity.Trip{
		ID:          tripID,
		Destination: "Paris",
		Days:        2,
		Travelers:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.TripID != tripID.String() {
		t.Errorf("expected tripId %s, got %s", tripID, captured.TripID)
	}
	if captured.StartDate != nil {
		t.Errorf("expected startDate to be omitted, got %q", *captured.StartDate)
	}
	if captured.BudgetAmount != nil {
		t.Errorf("expected budgetAmount to be omitted, got %v", *captured.BudgetAmount)
	}
	if captured.BudgetCurrency != "EUR" {
		t.Errorf("expected default budgetCurrency EUR, got %q", captured.BudgetCurrency)
	}
	if captured.Pace != "balanced" {
		t.Errorf("expected default pace balanced, got %q", captured.Pace)
	}
	if captured.Interests == nil || len(captured.Interests) != 0 {
		t.Errorf("expected interests to be an empty array, got %#v", captured.Interests)
	}
}

func TestAIPlanningHTTPGeneratorGenerate_Non2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid trip", http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Second
	gen := newTestHTTPGenerator(t, server.URL, client)

	_, err := gen.Generate(context.Background(), validTrip())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 422") || !strings.Contains(err.Error(), "invalid trip") {
		t.Fatalf("expected status and response body in error, got %v", err)
	}
}

func TestAIPlanningHTTPGeneratorGenerate_InvalidJSONReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"days":`)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Second
	gen := newTestHTTPGenerator(t, server.URL, client)

	_, err := gen.Generate(context.Background(), validTrip())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "decode ai planning response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestAIPlanningHTTPGeneratorGenerate_EmptyDaysReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"days":[]}`)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Second
	gen := newTestHTTPGenerator(t, server.URL, client)

	_, err := gen.Generate(context.Background(), validTrip())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "empty itinerary days") {
		t.Fatalf("expected empty-days error, got %v", err)
	}
}

func TestAIPlanningHTTPGeneratorGenerate_RequestFailureReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Millisecond
	gen := newTestHTTPGenerator(t, server.URL, client)

	_, err := gen.Generate(context.Background(), validTrip())
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "call ai planning service") {
		t.Fatalf("expected request failure error, got %v", err)
	}
}

func assertCapturedPayload(t *testing.T, got aiPlanningGenerateRequest, tripID, startDate string, budget *float64) {
	t.Helper()

	if got.TripID != tripID {
		t.Errorf("expected tripId %s, got %s", tripID, got.TripID)
	}
	if got.Destination != "Rome" {
		t.Errorf("expected destination Rome, got %q", got.Destination)
	}
	if got.StartDate == nil || *got.StartDate != startDate {
		t.Fatalf("expected startDate %s, got %v", startDate, got.StartDate)
	}
	if got.Days != 4 {
		t.Errorf("expected days 4, got %d", got.Days)
	}
	if got.BudgetAmount == nil || *got.BudgetAmount != *budget {
		t.Fatalf("expected budgetAmount %v, got %v", *budget, got.BudgetAmount)
	}
	if got.BudgetCurrency != "EUR" {
		t.Errorf("expected budgetCurrency EUR, got %q", got.BudgetCurrency)
	}
	if got.Travelers != 2 {
		t.Errorf("expected travelers 2, got %d", got.Travelers)
	}
	if strings.Join(got.Interests, ",") != "food,history,hidden_gems" {
		t.Errorf("unexpected interests: %#v", got.Interests)
	}
	if got.Pace != "balanced" {
		t.Errorf("expected pace balanced, got %q", got.Pace)
	}
}

func newTestHTTPGenerator(t *testing.T, baseURL string, client *http.Client) *AIPlanningHTTPGenerator {
	t.Helper()

	gen, err := NewAIPlanningHTTPGenerator(baseURL, client, zap.NewNop())
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	return gen
}

func validTrip() entity.Trip {
	return entity.Trip{
		ID:             uuid.New(),
		Destination:    "Rome",
		Days:           4,
		BudgetCurrency: "EUR",
		Travelers:      2,
		Pace:           "balanced",
	}
}
