package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateAndNormalizeItineraryAllowsExplicitUnscheduledItem(t *testing.T) {
	raw := json.RawMessage(`{
		"destination":"Vienna",
		"currency":"EUR",
		"days":[{
			"day":1,
			"title":"Arrival",
			"items":[{"time":"","type":"activity","name":"Museum option","schedulingStatus":"Unscheduled"}]
		}]
	}`)

	normalized, err := validateAndNormalizeItinerary(raw)
	if err != nil {
		t.Fatalf("validateAndNormalizeItinerary returned error: %v", err)
	}
	if !strings.Contains(string(normalized), `"schedulingStatus":"Unscheduled"`) {
		t.Fatalf("expected unscheduled status in normalized itinerary: %s", normalized)
	}
}

func TestValidateAndNormalizeItineraryNormalizesStartTimeAlias(t *testing.T) {
	raw := json.RawMessage(`{
		"destination":"Vienna",
		"currency":"EUR",
		"days":[{
			"day":1,
			"title":"Arrival",
			"items":[{"startTime":"09:30","type":"activity","name":"Museum","durationMinutes":90}]
		}]
	}`)

	normalized, err := validateAndNormalizeItinerary(raw)
	if err != nil {
		t.Fatalf("validateAndNormalizeItinerary returned error: %v", err)
	}
	payload := string(normalized)
	if !strings.Contains(payload, `"time":"09:30"`) || !strings.Contains(payload, `"startTime":"09:30"`) {
		t.Fatalf("expected time/startTime alias in normalized itinerary: %s", normalized)
	}
}

func TestValidateAndNormalizeItineraryRejectsOverlappingSchedule(t *testing.T) {
	raw := json.RawMessage(`{
		"destination":"Vienna",
		"currency":"EUR",
		"days":[{
			"day":1,
			"title":"Arrival",
			"items":[
				{"time":"09:00","type":"activity","name":"Museum","durationMinutes":90},
				{"time":"10:00","type":"activity","name":"Market","durationMinutes":60}
			]
		}]
	}`)

	_, err := validateAndNormalizeItinerary(raw)
	if err == nil {
		t.Fatal("expected overlap validation error")
	}
	if !strings.Contains(err.Error(), "overlaps") {
		t.Fatalf("expected overlap error, got %v", err)
	}
}
