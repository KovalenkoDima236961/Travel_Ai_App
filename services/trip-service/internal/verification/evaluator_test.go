package verification

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestEvaluateClassifiesTransportFreshnessAndSourceQuality(t *testing.T) {
	now := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	minutes := 75

	tests := []struct {
		name       string
		option     *aggregate.SelectedTransportOption
		wantStatus Status
		wantSource Source
	}{
		{
			name: "recent provider result is verified",
			option: &aggregate.SelectedTransportOption{
				ID: "rail-1", Provider: "rail-provider", Status: "available", CheckedAt: now.Add(-time.Hour).Format(time.RFC3339),
			},
			wantStatus: StatusVerified,
			wantSource: SourceProvider,
		},
		{
			name: "old provider result is stale",
			option: &aggregate.SelectedTransportOption{
				ID: "rail-1", Provider: "rail-provider", Status: "available", CheckedAt: now.Add(-8 * 24 * time.Hour).Format(time.RFC3339),
			},
			wantStatus: StatusStale,
			wantSource: SourceProvider,
		},
		{
			name: "mock provider is estimated rather than verified",
			option: &aggregate.SelectedTransportOption{
				ID: "rail-1", Provider: "mock-rail", Status: "available", CheckedAt: now.Add(-time.Hour).Format(time.RFC3339),
			},
			wantStatus: StatusEstimated,
			wantSource: SourceMock,
		},
		{
			name:       "no selection is missing",
			option:     nil,
			wantStatus: StatusMissing,
			wantSource: SourceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := verificationTestTrip(tt.option, minutes, now)
			response := Evaluate(Input{Trip: trip, Now: now, Config: DefaultConfig()})
			detail := findDetail(t, response, ScopeTransport, "leg-1")
			if detail.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", detail.Status, tt.wantStatus)
			}
			if detail.Source != tt.wantSource {
				t.Fatalf("source = %q, want %q", detail.Source, tt.wantSource)
			}
		})
	}
}

func TestEvaluateUsesPersistedWeatherCheckAndDoesNotTreatFallbackAsVerified(t *testing.T) {
	now := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	trip := verificationTestTrip(nil, 30, now)
	trip.CreationMetadata = map[string]any{
		"verification": map[string]any{
			"weather": map[string]any{
				"provider": "weather-provider", "checkedAt": now.Add(-2 * time.Hour).Format(time.RFC3339), "fallbackUsed": true,
			},
		},
	}

	response := Evaluate(Input{Trip: trip, Now: now, Config: DefaultConfig()})
	weather := findDetail(t, response, ScopeWeather, trip.ID.String())
	if weather.Status != StatusEstimated {
		t.Fatalf("weather status = %q, want %q", weather.Status, StatusEstimated)
	}
	if weather.Source != SourceFallback {
		t.Fatalf("weather source = %q, want %q", weather.Source, SourceFallback)
	}
	for _, issue := range response.TopIssues {
		if issue.Status == StatusVerified {
			t.Fatal("verified details must not be presented as top issues")
		}
	}
}

func TestEvaluateSortsUnavailableAheadOfMissingTopIssues(t *testing.T) {
	now := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	option := &aggregate.SelectedTransportOption{
		ID: "rail-1", Provider: "rail-provider", Status: "unavailable", CheckedAt: now.Format(time.RFC3339),
	}
	response := Evaluate(Input{Trip: verificationTestTrip(option, 30, now), Now: now, Config: DefaultConfig()})

	if len(response.TopIssues) == 0 {
		t.Fatal("expected top issues")
	}
	if response.TopIssues[0].Status != StatusUnavailable {
		t.Fatalf("first top issue status = %q, want %q", response.TopIssues[0].Status, StatusUnavailable)
	}
}

func TestEvaluateCapsReturnedSectionDetails(t *testing.T) {
	response := Evaluate(Input{
		Now:    time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC),
		Config: Config{MaxDetails: 3},
	})

	count := 0
	for _, section := range response.Sections {
		count += len(section.Details)
	}
	if count > 3 {
		t.Fatalf("returned %d section details, want at most 3", count)
	}
}

func verificationTestTrip(option *aggregate.SelectedTransportOption, duration int, now time.Time) *entity.Trip {
	start := now.AddDate(0, 0, 3)
	return &entity.Trip{
		ID:          uuid.New(),
		Destination: "Bratislava",
		StartDate:   &start,
		Days:        3,
		Route: &aggregate.TripRoute{Legs: []aggregate.RouteLeg{{
			ID: "leg-1", FromName: "Vienna", ToName: "Bratislava", Mode: aggregate.TransportModeTrain,
			EstimatedDurationMinutes: &duration, SelectedTransportOption: option,
		}}},
	}
}

func findDetail(t *testing.T, response Response, scope Scope, entityID string) Detail {
	t.Helper()
	for _, section := range response.Sections {
		if section.Scope != scope {
			continue
		}
		for _, detail := range section.Details {
			if detail.EntityID == entityID {
				return detail
			}
		}
	}
	t.Fatalf("detail not found: scope=%s entity=%s", scope, entityID)
	return Detail{}
}
