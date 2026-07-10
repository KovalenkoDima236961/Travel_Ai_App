package calendarsync

import (
	"strings"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

func TestParseTimeRangeSupportedFormats(t *testing.T) {
	cases := []string{"09:00", "9:00", "09:00 AM", "9:00 AM", "2:30 PM", "14:30", "09:00-10:30", "09:00 – 10:30"}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, _, ok := parseTimeRange(tc); !ok {
				t.Fatalf("expected %q to parse", tc)
			}
		})
	}
}

func TestParseTimeRangeRejectsUnparseable(t *testing.T) {
	if _, _, ok := parseTimeRange("morning"); ok {
		t.Fatal("expected morning to be skipped")
	}
}

func TestDefaultDuration(t *testing.T) {
	if defaultDuration("food").Minutes() != 90 {
		t.Fatal("food should default to 90 minutes")
	}
	if defaultDuration("transport").Minutes() != 30 {
		t.Fatal("transport should default to 30 minutes")
	}
	if defaultDuration("transfer").Minutes() != 120 {
		t.Fatal("transfer should default to 120 minutes")
	}
	if defaultDuration("activity").Minutes() != 60 {
		t.Fatal("activity should default to 60 minutes")
	}
}

func TestTransferCalendarFields(t *testing.T) {
	duration := 150
	distance := 295.0
	item := aggregate.ItineraryItem{
		Time: "09:30",
		Type: "transfer",
		Name: "Train from Vienna to Salzburg",
		Transfer: &aggregate.TransferDetails{
			From:                     "Vienna",
			To:                       "Salzburg",
			Mode:                     "train",
			EstimatedDurationMinutes: &duration,
			EstimatedDistanceKm:      &distance,
			Warnings:                 []string{"This is an estimate, not a live schedule."},
		},
	}

	if got := itemTitle(item); got != "Transfer: Vienna -> Salzburg" {
		t.Fatalf("unexpected title: %q", got)
	}
	if got := itemLocation(item, "Austria"); got != "Vienna -> Salzburg" {
		t.Fatalf("unexpected location: %q", got)
	}
	description := buildDescription(item, "", "EUR")
	for _, want := range []string{
		"Transport mode: train",
		"Estimated duration: 150 minutes",
		"Estimated distance: 295.0 km",
		"Verify schedules before travel.",
	} {
		if !strings.Contains(description, want) {
			t.Fatalf("description missing %q: %s", want, description)
		}
	}
}
