package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestTravelDayForTripUsesRequestedDate(t *testing.T) {
	start := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	trip := &entity.Trip{
		ID:        uuid.New(),
		StartDate: &start,
		Days:      3,
		Itinerary: []byte(`{"days":[{"day":2,"date":"2026-07-16","title":"Day two","items":[]}]}`),
	}

	dayNumber, mode, day := travelDayForTrip(trip, time.Date(2026, time.July, 16, 0, 0, 0, 0, time.UTC))
	if dayNumber != 2 || mode != "active" || day == nil || day.Title != "Day two" {
		t.Fatalf("unexpected travel day result: number=%d mode=%q day=%+v", dayNumber, mode, day)
	}

	dayNumber, mode, day = travelDayForTrip(trip, time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC))
	if dayNumber != 1 || mode != "pre_trip" || day != nil {
		t.Fatalf("expected pre-trip mode, got number=%d mode=%q day=%+v", dayNumber, mode, day)
	}
}

func TestTravelDayNowNextSkipsDoneAndFindsCurrent(t *testing.T) {
	date := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.July, 18, 10, 15, 0, 0, time.UTC)
	timeline := []TravelDayTimelineItem{
		{ItemIndex: 0, StartTime: "09:00", EndTime: "10:00", Title: "Completed", TravelStatus: aggregate.TravelStatus{Status: travelStatusDone}},
		{ItemIndex: 1, StartTime: "10:00", EndTime: "11:00", Title: "Current", TravelStatus: aggregate.TravelStatus{Status: travelStatusPlanned}},
		{ItemIndex: 2, StartTime: "12:00", EndTime: "13:00", Title: "Next", TravelStatus: aggregate.TravelStatus{Status: travelStatusPlanned}},
	}

	result := travelDayNowNext(timeline, date, now)
	if result.CurrentItem == nil || result.CurrentItem.Title != "Current" {
		t.Fatalf("current item = %+v, want Current", result.CurrentItem)
	}
	if result.NextItem == nil || result.NextItem.Title != "Next" {
		t.Fatalf("next item = %+v, want Next", result.NextItem)
	}
}

func TestTravelStatusDefaultsAndValidation(t *testing.T) {
	if !validTravelStatus(travelStatusDone) || validTravelStatus("arrived") {
		t.Fatal("travel status validation mismatch")
	}
	if got := travelStatusForItem(aggregate.ItineraryItem{}); got.Status != travelStatusPlanned {
		t.Fatalf("default status = %q, want planned", got.Status)
	}
}
