package places

import (
	"regexp"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

func TestNewUnsupportedProviderReturnsError(t *testing.T) {
	_, err := New(&config.Config{PlaceProvider: config.PlaceProviderConfig{Provider: "google"}}, zap.NewNop())
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if !strings.Contains(err.Error(), "unsupported PLACE_PROVIDER") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMockPlaceOpeningHoursUseValidConvention(t *testing.T) {
	timeFormat := regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`)

	for _, item := range mockPlaces() {
		if len(item.OpeningHours) == 0 {
			t.Fatalf("expected mock place %s to include opening hours", item.ProviderPlaceID)
		}
		for _, interval := range item.OpeningHours {
			if interval.DayOfWeek < 1 || interval.DayOfWeek > 7 {
				t.Fatalf("mock place %s has invalid dayOfWeek: %+v", item.ProviderPlaceID, interval)
			}
			if !timeFormat.MatchString(interval.Open) || !timeFormat.MatchString(interval.Close) {
				t.Fatalf("mock place %s has invalid HH:mm interval: %+v", item.ProviderPlaceID, interval)
			}
			if interval.Open >= interval.Close {
				t.Fatalf("mock place %s has non-ascending interval: %+v", item.ProviderPlaceID, interval)
			}
		}
	}
}
