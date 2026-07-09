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

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
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
	got, err := gen.Generate(context.Background(), application.GenerateItineraryInput{
		Trip: entity.Trip{
			ID:             tripID,
			Destination:    "Rome",
			StartDate:      &startDate,
			Days:           4,
			BudgetAmount:   &budget,
			BudgetCurrency: "EUR",
			Travelers:      2,
			Interests:      []string{"food", "history", "hidden_gems"},
			Pace:           "balanced",
			Accommodation:  testAccommodation(),
		},
		WeatherForecast: validWeatherForecast(),
		WorkspacePolicyConstraints: &workspacepolicies.AIConstraints{
			Summary: "Avoid activities after 22:00.",
			Rules:   json.RawMessage(`{"schemaVersion":1,"rules":{}}`),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil || len(got.Days) != 1 {
		t.Fatalf("expected one itinerary day, got %+v", got)
	}
	gotCost := got.Days[0].Items[0].EstimatedCost
	if gotCost == nil || gotCost.Amount == nil || *gotCost.Amount != 18 {
		t.Fatalf("expected legacy numeric estimated cost to decode, got %+v", gotCost)
	}
	if got.Destination != "Rome" || got.Source != "ai-planning-service-http" {
		t.Fatalf("expected enriched itinerary metadata, got %+v", got)
	}

	assertCapturedPayload(t, captured, tripID.String(), "2026-08-10", &budget)
	if captured.WeatherForecast == nil || captured.WeatherForecast.Days[0].Condition != "hot" {
		t.Fatalf("expected weatherForecast to be serialized, got %+v", captured.WeatherForecast)
	}
	if captured.Accommodation == nil || captured.Accommodation.Name != "Hotel Roma" {
		t.Fatalf("expected accommodation to be serialized, got %+v", captured.Accommodation)
	}
	if captured.WorkspacePolicyConstraints == nil ||
		captured.WorkspacePolicyConstraints.Summary != "Avoid activities after 22:00." {
		t.Fatalf("expected workspace policy constraints, got %+v", captured.WorkspacePolicyConstraints)
	}
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
	_, err := gen.Generate(context.Background(), application.GenerateItineraryInput{Trip: entity.Trip{
		ID:          tripID,
		Destination: "Paris",
		Days:        2,
		Travelers:   1,
	}})
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

	_, err := gen.Generate(context.Background(), application.GenerateItineraryInput{Trip: validTrip()})
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

	_, err := gen.Generate(context.Background(), application.GenerateItineraryInput{Trip: validTrip()})
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

	_, err := gen.Generate(context.Background(), application.GenerateItineraryInput{Trip: validTrip()})
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

	_, err := gen.Generate(context.Background(), application.GenerateItineraryInput{Trip: validTrip()})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "call ai planning service") {
		t.Fatalf("expected request failure error, got %v", err)
	}
}

func TestAIPlanningHTTPGeneratorGenerate_SerializesUserContext(t *testing.T) {
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

	displayName := "Dmytro"
	homeCity := "Bratislava"
	homeCountry := "Slovakia"
	walking := 8.0
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	_, err := gen.Generate(context.Background(), application.GenerateItineraryInput{
		Trip: validTrip(),
		UserProfile: &usercontext.UserProfile{
			UserID:            userID,
			DisplayName:       &displayName,
			HomeCity:          &homeCity,
			HomeCountry:       &homeCountry,
			PreferredCurrency: "EUR",
			PreferredLanguage: "en",
		},
		UserPreferences: &usercontext.UserPreferences{
			UserID:             userID,
			TravelStyles:       []string{"budget", "food", "hidden_gems"},
			Pace:               "balanced",
			MaxWalkingKmPerDay: &walking,
			FoodPreferences:    []string{"local", "cheap"},
			Avoid:              []string{"nightclubs"},
			PreferredTransport: []string{"walking", "public_transport"},
			AccommodationStyle: []string{"budget_hotel"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.UserProfile == nil || captured.UserProfile.DisplayName == nil || *captured.UserProfile.DisplayName != "Dmytro" {
		t.Fatalf("expected userProfile to be serialized, got %+v", captured.UserProfile)
	}
	if captured.UserPreferences == nil {
		t.Fatal("expected userPreferences to be serialized")
	}
	if strings.Join(captured.UserPreferences.TravelStyles, ",") != "budget,food,hidden_gems" {
		t.Fatalf("unexpected travelStyles: %#v", captured.UserPreferences.TravelStyles)
	}
	if captured.UserPreferences.MaxWalkingKmPerDay == nil || *captured.UserPreferences.MaxWalkingKmPerDay != 8 {
		t.Fatalf("expected maxWalkingKmPerDay 8, got %+v", captured.UserPreferences.MaxWalkingKmPerDay)
	}
}

func TestAIPlanningHTTPGeneratorRegenerateDay_SendsCorrectPayload(t *testing.T) {
	var captured aiPlanningRegenerateDayRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/regenerate-day" {
			t.Errorf("expected path /regenerate-day, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"day":{"day":2,"title":"Relaxed food day","items":[{"time":"10:00","type":"food","name":"Bakery","note":"Local start","estimatedCost":8}]}}`)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Second
	gen := newTestHTTPGenerator(t, server.URL, client)
	walking := 8.0
	input := application.RegenerateDayInput{
		Trip:             validTrip(),
		CurrentItinerary: validHTTPTestItinerary(),
		DayNumber:        2,
		Instruction:      "Make it more relaxed",
		UserPreferences:  &usercontext.UserPreferences{TravelStyles: []string{"food"}, MaxWalkingKmPerDay: &walking},
		WeatherForecast:  validWeatherForecast(),
	}

	got, err := gen.RegenerateDay(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Day != 2 || got.Title != "Relaxed food day" {
		t.Fatalf("unexpected replacement day: %+v", got)
	}
	if captured.Trip.ID != input.Trip.ID.String() || captured.Trip.Destination != "Rome" {
		t.Fatalf("unexpected trip payload: %+v", captured.Trip)
	}
	if len(captured.CurrentItinerary.Days) != 2 || captured.CurrentItinerary.Days[1].Day != 2 {
		t.Fatalf("expected current itinerary in payload, got %+v", captured.CurrentItinerary)
	}
	if captured.DayNumber != 2 || captured.Instruction != "Make it more relaxed" {
		t.Fatalf("unexpected partial fields: %+v", captured)
	}
	if captured.UserPreferences == nil || captured.UserPreferences.MaxWalkingKmPerDay == nil || *captured.UserPreferences.MaxWalkingKmPerDay != 8 {
		t.Fatalf("expected userPreferences to be serialized, got %+v", captured.UserPreferences)
	}
	if captured.WeatherForecast == nil || captured.WeatherForecast.Provider != "mock" {
		t.Fatalf("expected weatherForecast to be serialized, got %+v", captured.WeatherForecast)
	}
	if captured.Accommodation == nil || captured.Accommodation.Type != aggregate.AccommodationTypeHotel {
		t.Fatalf("expected accommodation to be serialized, got %+v", captured.Accommodation)
	}
}

func TestAIPlanningHTTPGeneratorRegenerateItem_SendsCorrectPayload(t *testing.T) {
	var captured aiPlanningRegenerateItemRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/regenerate-item" {
			t.Errorf("expected path /regenerate-item, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"item":{"time":"12:30","type":"food","name":"Local trattoria","note":"Cheaper option","estimatedCost":15}}`)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = time.Second
	gen := newTestHTTPGenerator(t, server.URL, client)
	input := application.RegenerateItemInput{
		Trip:             validTrip(),
		CurrentItinerary: validHTTPTestItinerary(),
		DayNumber:        2,
		ItemIndex:        1,
		Instruction:      "Replace with cheaper food",
		WeatherForecast:  validWeatherForecast(),
	}

	got, err := gen.RegenerateItem(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Name != "Local trattoria" || got.EstimatedCost == nil || got.EstimatedCost.Amount == nil || *got.EstimatedCost.Amount != 15 {
		t.Fatalf("unexpected replacement item: %+v", got)
	}
	if captured.Trip.ID != input.Trip.ID.String() || captured.DayNumber != 2 || captured.ItemIndex != 1 {
		t.Fatalf("unexpected item regeneration payload: %+v", captured)
	}
	if captured.Instruction != "Replace with cheaper food" {
		t.Fatalf("expected instruction to be serialized, got %q", captured.Instruction)
	}
	if len(captured.CurrentItinerary.Days) != 2 {
		t.Fatalf("expected current itinerary in payload, got %+v", captured.CurrentItinerary)
	}
	if captured.WeatherForecast == nil || captured.WeatherForecast.Days[0].TemperatureMaxC != 35 {
		t.Fatalf("expected weatherForecast to be serialized, got %+v", captured.WeatherForecast)
	}
	if captured.Accommodation == nil || captured.Accommodation.Address != "Via Roma 10" {
		t.Fatalf("expected accommodation to be serialized, got %+v", captured.Accommodation)
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
		Accommodation:  testAccommodation(),
	}
}

func testAccommodation() *aggregate.Accommodation {
	return &aggregate.Accommodation{
		Name:         "Hotel Roma",
		Type:         aggregate.AccommodationTypeHotel,
		Address:      "Via Roma 10",
		CheckInDate:  "2026-08-10",
		CheckOutDate: "2026-08-14",
	}
}

func validHTTPTestItinerary() aggregate.Itinerary {
	return aggregate.Itinerary{
		Destination: "Rome",
		Days: []aggregate.ItineraryDay{
			{
				Day:   1,
				Title: "Day 1",
				Items: []aggregate.ItineraryItem{
					{Time: "09:00", Type: "activity", Name: "Walk"},
				},
			},
			{
				Day:   2,
				Title: "Day 2",
				Items: []aggregate.ItineraryItem{
					{Time: "10:00", Type: "place", Name: "Museum"},
					{Time: "13:00", Type: "food", Name: "Lunch"},
				},
			},
		},
	}
}

func validWeatherForecast() *weathercontext.WeatherForecast {
	return &weathercontext.WeatherForecast{
		Destination: "Rome",
		Provider:    "mock",
		Days: []weathercontext.WeatherDay{
			{
				Date:                "2026-08-10",
				Condition:           "hot",
				TemperatureMinC:     24,
				TemperatureMaxC:     35,
				PrecipitationChance: 5,
				WindSpeedKph:        10,
				Summary:             "Hot and sunny",
				Warnings:            []string{"High heat: avoid long outdoor walks at midday"},
			},
		},
	}
}
