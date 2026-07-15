package service

import (
	"context"
	"testing"
	"time"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/calendarclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"go.uber.org/zap"
)

func TestBuildCalendarImportPreviewConvertsBusyBlocksToDateRanges(t *testing.T) {
	loc := mustLoadLocation(t, "Europe/Bratislava")
	preview := buildCalendarImportPreview(normalizedCalendarImportInput{
		startDate:   "2026-09-10",
		endDate:     "2026-09-15",
		timezone:    "Europe/Bratislava",
		calendarIDs: []string{"primary"},
		conversion: appdto.CalendarImportConversionSettings{
			FullyBusyThresholdHours:          6,
			MarkFullyBusyDaysUnavailable:     true,
			MarkPartiallyBusyDaysUnavailable: false,
		},
	}, []calendarclient.FreeBusyBlock{
		{
			Start:  time.Date(2026, 9, 11, 9, 0, 0, 0, loc),
			End:    time.Date(2026, 9, 11, 11, 0, 0, 0, loc),
			Source: "google_calendar",
		},
		{
			Start:  time.Date(2026, 9, 12, 0, 0, 0, 0, loc),
			End:    time.Date(2026, 9, 14, 0, 0, 0, 0, loc),
			AllDay: true,
			Source: "google_calendar",
		},
	}, nil)

	if preview.BusyBlocksSummary.BusyBlockCount != 2 {
		t.Fatalf("unexpected block count: %d", preview.BusyBlocksSummary.BusyBlockCount)
	}
	if preview.BusyBlocksSummary.FullyBusyDays != 2 {
		t.Fatalf("expected two fully busy days, got %d", preview.BusyBlocksSummary.FullyBusyDays)
	}
	if preview.BusyBlocksSummary.PartiallyBusyDays != 1 {
		t.Fatalf("expected one partially busy day, got %d", preview.BusyBlocksSummary.PartiallyBusyDays)
	}
	if len(preview.SuggestedUnavailableRanges) != 1 {
		t.Fatalf("expected one unavailable range, got %d", len(preview.SuggestedUnavailableRanges))
	}
	got := preview.SuggestedUnavailableRanges[0]
	if got.StartDate != "2026-09-12" || got.EndDate != "2026-09-13" {
		t.Fatalf("unexpected unavailable range: %+v", got)
	}
}

func TestBuildCalendarImportPreviewCanIncludePartialBusyAndFreeWeekends(t *testing.T) {
	loc := mustLoadLocation(t, "Europe/Bratislava")
	preview := buildCalendarImportPreview(normalizedCalendarImportInput{
		startDate:   "2026-09-18",
		endDate:     "2026-09-21",
		timezone:    "Europe/Bratislava",
		calendarIDs: []string{"primary"},
		conversion: appdto.CalendarImportConversionSettings{
			FullyBusyThresholdHours:          6,
			MarkFullyBusyDaysUnavailable:     true,
			MarkPartiallyBusyDaysUnavailable: true,
			IncludeWeekendsAsPreferredIfFree: true,
		},
	}, []calendarclient.FreeBusyBlock{
		{
			Start:  time.Date(2026, 9, 18, 10, 0, 0, 0, loc),
			End:    time.Date(2026, 9, 18, 12, 0, 0, 0, loc),
			Source: "google_calendar",
		},
	}, nil)

	if len(preview.SuggestedUnavailableRanges) != 1 || preview.SuggestedUnavailableRanges[0].StartDate != "2026-09-18" {
		t.Fatalf("partial busy day was not suggested unavailable: %+v", preview.SuggestedUnavailableRanges)
	}
	if len(preview.SuggestedPreferredRanges) != 1 {
		t.Fatalf("expected one preferred weekend range, got %d", len(preview.SuggestedPreferredRanges))
	}
	if preview.SuggestedPreferredRanges[0].StartDate != "2026-09-19" || preview.SuggestedPreferredRanges[0].EndDate != "2026-09-20" {
		t.Fatalf("unexpected preferred range: %+v", preview.SuggestedPreferredRanges[0])
	}
}

func TestMergeAvailabilityRangesDeduplicatesAndUnavailableOverrides(t *testing.T) {
	merged := mergeAvailabilityRanges(
		[]entity.AvailabilityDateRange{{StartDate: "2026-09-10", EndDate: "2026-09-11"}},
		[]entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-12"}},
	)
	if len(merged) != 1 || merged[0].StartDate != "2026-09-10" || merged[0].EndDate != "2026-09-12" {
		t.Fatalf("unexpected merged ranges: %+v", merged)
	}
	preferred := subtractOverriddenRanges(
		[]entity.AvailabilityDateRange{{StartDate: "2026-09-10", EndDate: "2026-09-15"}},
		[]entity.AvailabilityDateRange{{StartDate: "2026-09-12", EndDate: "2026-09-13"}},
	)
	if len(preferred) != 2 {
		t.Fatalf("expected split preferred ranges, got %+v", preferred)
	}
}

func TestCalendarImportFailOpenPreviewReturnsEmptyWarningOnDependencyFailure(t *testing.T) {
	svc := &Service{
		log:                                zap.NewNop(),
		calendarAvailabilityProvider:       stubCalendarAvailabilityProvider{},
		calendarAvailabilityImportEnabled:  true,
		calendarAvailabilityImportFailOpen: true,
		calendarSyncDefaultTimeZone:        "Europe/Bratislava",
	}

	preview, ok := svc.calendarImportFailOpenPreview(context.Background(), appdto.CalendarImportBaseInput{
		StartDate:        "2026-09-01",
		EndDate:          "2026-09-03",
		CalendarProvider: "google",
	}, apperrs.NewDependencyError("calendar_free_busy_unavailable"))
	if !ok {
		t.Fatal("expected fail-open preview")
	}
	if preview.BusyBlocksSummary.BusyBlockCount != 0 || len(preview.SuggestedUnavailableRanges) != 0 {
		t.Fatalf("expected empty calendar suggestions, got %+v", preview)
	}
	if preview.Range.Timezone != "Europe/Bratislava" {
		t.Fatalf("expected default timezone, got %q", preview.Range.Timezone)
	}
	if len(preview.Warnings) == 0 {
		t.Fatal("expected warning")
	}

	_, ok = svc.calendarImportFailOpenPreview(context.Background(), appdto.CalendarImportBaseInput{
		StartDate:        "2026-09-01",
		EndDate:          "2026-09-03",
		CalendarProvider: "google",
	}, apperrs.NewInvalidInput("calendar_not_connected"))
	if ok {
		t.Fatal("invalid input errors must not fail open")
	}
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return loc
}

type stubCalendarAvailabilityProvider struct{}

func (stubCalendarAvailabilityProvider) GetGoogleCalendarStatus(context.Context, string) (*calendarclient.ConnectionStatus, error) {
	return nil, nil
}

func (stubCalendarAvailabilityProvider) GetGoogleFreeBusy(context.Context, string, calendarclient.FreeBusyRequest) (*calendarclient.FreeBusyResponse, error) {
	return nil, nil
}

func (stubCalendarAvailabilityProvider) SyncGoogleCalendarEvents(context.Context, calendarclient.SyncRequest) (*calendarclient.SyncResult, error) {
	return nil, nil
}

func (stubCalendarAvailabilityProvider) DeleteGoogleCalendarEvents(context.Context, calendarclient.DeleteRequest) (*calendarclient.DeleteResult, error) {
	return nil, nil
}
