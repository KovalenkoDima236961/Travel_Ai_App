package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/calendarclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

const (
	calendarImportSourceGoogle     = "google_calendar"
	calendarImportModeMerge        = "merge"
	calendarImportModeOverwriteAll = "overwrite_all_my_availability"
)

type normalizedCalendarImportInput struct {
	startDate   string
	endDate     string
	timezone    string
	calendarIDs []string
	conversion  appdto.CalendarImportConversionSettings
}

type calendarDayAccumulator struct {
	date           string
	busyHours      float64
	busyBlockCount int
	allDay         bool
	status         string
}

func (s *Service) PreviewCalendarAvailabilityImport(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.CalendarImportPreviewInput,
) (appdto.CalendarImportPreviewResult, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.CalendarImportPreviewResult{}, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return appdto.CalendarImportPreviewResult{}, err
	}
	normalized, freeBusy, err := s.fetchCalendarFreeBusy(ctx, in.CalendarImportBaseInput)
	if err != nil {
		if preview, ok := s.calendarImportFailOpenPreview(ctx, in.CalendarImportBaseInput, err); ok {
			return appdto.CalendarImportPreviewResult{Preview: preview}, nil
		}
		return appdto.CalendarImportPreviewResult{}, err
	}
	preview := buildCalendarImportPreview(normalized, freeBusy.BusyBlocks, freeBusy.Warnings)
	return appdto.CalendarImportPreviewResult{Preview: preview}, nil
}

func (s *Service) ApplyCalendarAvailabilityImport(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.CalendarImportApplyInput,
) (appdto.CalendarImportApplyResult, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.CalendarImportApplyResult{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.CalendarImportApplyResult{}, err
	}
	normalized, freeBusy, err := s.fetchCalendarFreeBusy(ctx, in.CalendarImportBaseInput)
	if err != nil {
		return appdto.CalendarImportApplyResult{}, err
	}
	preview := buildCalendarImportPreview(normalized, freeBusy.BusyBlocks, freeBusy.Warnings)
	mode := strings.TrimSpace(in.Mode)
	if mode == "" {
		mode = calendarImportModeMerge
	}
	if mode != calendarImportModeMerge && mode != calendarImportModeOverwriteAll && mode != "overwrite_calendar_imported" {
		return appdto.CalendarImportApplyResult{}, apperrs.NewInvalidInput("mode must be merge or overwrite_all_my_availability")
	}

	existing, err := s.repo.GetTripAvailabilityResponseByTripAndUser(ctx, tripID, user.ID)
	if err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
		return appdto.CalendarImportApplyResult{}, err
	}
	settings, err := normalizeAvailabilityInput(in.AvailabilitySettings)
	if err != nil {
		return appdto.CalendarImportApplyResult{}, err
	}
	importedUnavailable := rangesWithoutReasons(preview.SuggestedUnavailableRanges)
	importedPreferred := rangesWithoutReasons(preview.SuggestedPreferredRanges)

	availableRanges := cloneAvailabilityRanges(settings.AvailableRanges)
	unavailableRanges := importedUnavailable
	preferredRanges := importedPreferred
	if mode == calendarImportModeMerge || mode == "overwrite_calendar_imported" {
		if existing != nil {
			availableRanges = mergeAvailabilityRanges(existing.AvailableRanges, availableRanges)
			unavailableRanges = mergeAvailabilityRanges(existing.UnavailableRanges, importedUnavailable)
			preferredRanges = mergeAvailabilityRanges(existing.PreferredRanges, importedPreferred)
		}
	}
	unavailableRanges = mergeAvailabilityRanges(unavailableRanges)
	preferredRanges = subtractOverriddenRanges(mergeAvailabilityRanges(preferredRanges), unavailableRanges)
	availableRanges = subtractOverriddenRanges(mergeAvailabilityRanges(availableRanges), unavailableRanges)

	notes := strings.TrimSpace(settings.Notes)
	if notes == "" && existing != nil {
		notes = existing.Notes
	}
	if notes == "" {
		notes = "Some unavailable dates were imported from Google Calendar."
	}
	minTripDays := settings.MinTripDays
	maxTripDays := settings.MaxTripDays
	if mode == calendarImportModeMerge && existing != nil {
		if minTripDays == nil {
			minTripDays = cloneIntPtr(existing.MinTripDays)
		}
		if maxTripDays == nil {
			maxTripDays = cloneIntPtr(existing.MaxTripDays)
		}
	}
	timezone := strings.TrimSpace(settings.Timezone)
	if timezone == "" {
		timezone = normalized.timezone
	}
	saved, err := s.repo.UpsertTripAvailabilityResponse(ctx, &entity.TripAvailabilityResponse{
		ID:                uuid.New(),
		TripID:            tripID,
		UserID:            user.ID,
		AvailableRanges:   availableRanges,
		UnavailableRanges: unavailableRanges,
		PreferredRanges:   preferredRanges,
		MinTripDays:       minTripDays,
		MaxTripDays:       maxTripDays,
		Timezone:          timezone,
		Notes:             notes,
	})
	if err != nil {
		return appdto.CalendarImportApplyResult{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventAvailabilityImportedFromCalendar,
		EntityType:  activityEntityType(activity.EntityAvailability),
		EntityID:    activityEntityID(saved.ID),
		Metadata: map[string]any{
			"provider":          "google",
			"rangeStart":        normalized.startDate,
			"rangeEnd":          normalized.endDate,
			"fullyBusyDays":     preview.BusyBlocksSummary.FullyBusyDays,
			"partiallyBusyDays": preview.BusyBlocksSummary.PartiallyBusyDays,
		},
	})
	responses, err := s.repo.ListTripAvailabilityResponsesByTrip(ctx, tripID)
	if err != nil {
		return appdto.CalendarImportApplyResult{}, err
	}
	participants, err := s.availabilityParticipants(ctx, trip, responses, &user)
	if err != nil {
		return appdto.CalendarImportApplyResult{}, err
	}
	list := buildAvailabilityList(tripID, participants, responses)
	dateOptions := calculateDateOptions(trip, participants, responses, appdto.DateOptionsInput{
		MinDays:         minTripDays,
		MaxDays:         maxTripDays,
		SearchStartDate: normalized.startDate,
		SearchEndDate:   normalized.endDate,
		Limit:           defaultDateOptionLimit,
	})
	dateOptions.Summary.ResponseCount = list.Summary.SubmittedCount
	dateOptions.Summary.TotalCollaborators = list.Summary.TotalCollaborators
	dateOptions.Summary.MissingResponseCount = list.Summary.MissingCount
	if len(dateOptions.Options) > 0 {
		dateOptions.Summary.RecommendedOptionID = dateOptions.Options[0].ID
	}
	displayName := displayNameForUser(user.ID, &user, trip, nil)
	return appdto.CalendarImportApplyResult{
		Availability: availabilityResponseInfo(displayName, *saved, true),
		DateOptions:  dateOptions,
	}, nil
}

func (s *Service) fetchCalendarFreeBusy(
	ctx context.Context,
	in appdto.CalendarImportBaseInput,
) (normalizedCalendarImportInput, *calendarclient.FreeBusyResponse, error) {
	if !s.calendarAvailabilityImportEnabled || s.calendarAvailabilityProvider == nil {
		return normalizedCalendarImportInput{}, nil, apperrs.NewDependencyError("calendar free/busy import is not configured")
	}
	normalized, err := normalizeCalendarImportInput(in, s.calendarSyncDefaultTimeZone)
	if err != nil {
		return normalizedCalendarImportInput{}, nil, err
	}
	accessToken, ok := auth.AccessTokenFromContext(ctx)
	if !ok {
		return normalizedCalendarImportInput{}, nil, apperrs.NewInvalidInput("calendar_auth_required")
	}
	freeBusy, err := s.calendarAvailabilityProvider.GetGoogleFreeBusy(ctx, accessToken, calendarclient.FreeBusyRequest{
		StartDate:   normalized.startDate,
		EndDate:     normalized.endDate,
		TimeZone:    normalized.timezone,
		CalendarIDs: normalized.calendarIDs,
	})
	if err != nil {
		return normalizedCalendarImportInput{}, nil, calendarClientError(err)
	}
	if freeBusy == nil {
		return normalizedCalendarImportInput{}, nil, apperrs.NewDependencyError("calendar_free_busy_unavailable")
	}
	return normalized, freeBusy, nil
}

func (s *Service) calendarImportFailOpenPreview(
	ctx context.Context,
	in appdto.CalendarImportBaseInput,
	err error,
) (appdto.CalendarImportPreview, bool) {
	if !s.calendarAvailabilityImportFailOpen || !s.calendarAvailabilityImportEnabled || s.calendarAvailabilityProvider == nil {
		return appdto.CalendarImportPreview{}, false
	}
	if !isCalendarImportDependencyError(err) {
		return appdto.CalendarImportPreview{}, false
	}
	normalized, normalizeErr := normalizeCalendarImportInput(in, s.calendarSyncDefaultTimeZone)
	if normalizeErr != nil {
		return appdto.CalendarImportPreview{}, false
	}
	if s.log != nil {
		userID := ""
		if user, userErr := auth.MustUserFromContext(ctx); userErr == nil {
			userID = user.ID.String()
		}
		s.log.Warn("calendar free busy import preview failed; returning empty fail-open preview",
			zap.String("userId", userID),
			zap.String("startDate", normalized.startDate),
			zap.String("endDate", normalized.endDate),
			zap.Error(err),
		)
	}
	return buildCalendarImportPreview(normalized, nil, []string{
		"Calendar free/busy import is temporarily unavailable. No calendar busy dates were imported.",
	}), true
}

func isCalendarImportDependencyError(err error) bool {
	var dependencyErr *apperrs.DependencyError
	return errors.As(err, &dependencyErr)
}

func normalizeCalendarImportInput(in appdto.CalendarImportBaseInput, defaultTimezone string) (normalizedCalendarImportInput, error) {
	provider := strings.ToLower(strings.TrimSpace(in.CalendarProvider))
	if provider == "" {
		provider = "google"
	}
	if provider != "google" {
		return normalizedCalendarImportInput{}, apperrs.NewInvalidInput("calendarProvider must be google")
	}
	start, err := parseAvailabilityDate(in.StartDate)
	if err != nil {
		return normalizedCalendarImportInput{}, apperrs.NewInvalidInput("startDate must be in YYYY-MM-DD format")
	}
	end, err := parseAvailabilityDate(in.EndDate)
	if err != nil {
		return normalizedCalendarImportInput{}, apperrs.NewInvalidInput("endDate must be in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return normalizedCalendarImportInput{}, apperrs.NewInvalidInput("endDate must be on or after startDate")
	}
	if int(end.Sub(start).Hours()/24)+1 > 180 {
		return normalizedCalendarImportInput{}, apperrs.NewInvalidInput("date range must be at most 180 days")
	}
	timezone := strings.TrimSpace(in.Timezone)
	if timezone == "" {
		timezone = strings.TrimSpace(defaultTimezone)
	}
	if timezone == "" {
		timezone = "UTC"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return normalizedCalendarImportInput{}, apperrs.NewInvalidInput("timezone is invalid")
	}
	calendarIDs := normalizeImportCalendarIDs(in.CalendarIDs)
	conversion := normalizeCalendarImportConversion(in.Conversion)
	return normalizedCalendarImportInput{
		startDate:   start.Format("2006-01-02"),
		endDate:     end.Format("2006-01-02"),
		timezone:    timezone,
		calendarIDs: calendarIDs,
		conversion:  conversion,
	}, nil
}

func normalizeCalendarImportConversion(in appdto.CalendarImportConversionSettings) appdto.CalendarImportConversionSettings {
	if in.FullyBusyThresholdHours <= 0 {
		in.FullyBusyThresholdHours = 6
	}
	if in.FullyBusyThresholdHours > 24 {
		in.FullyBusyThresholdHours = 24
	}
	if !in.MarkFullyBusyDaysUnavailable && !in.MarkPartiallyBusyDaysUnavailable && !in.IncludeWeekendsAsPreferredIfFree {
		in.MarkFullyBusyDaysUnavailable = true
	}
	return in
}

func normalizeImportCalendarIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{"primary"}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			trimmed = "primary"
		}
		out = append(out, trimmed)
	}
	return out
}

func buildCalendarImportPreview(
	input normalizedCalendarImportInput,
	blocks []calendarclient.FreeBusyBlock,
	providerWarnings []string,
) appdto.CalendarImportPreview {
	days := calendarBusyDayAccumulators(input, blocks)
	daySummaries := make([]appdto.CalendarBusyDaySummary, 0, len(days))
	fullyBusyDays := 0
	partiallyBusyDays := 0
	busyDays := 0
	unavailableDates := []string{}
	freeWeekendDates := []string{}
	start, _ := parseAvailabilityDate(input.startDate)
	end, _ := parseAvailabilityDate(input.endDate)
	dayByDate := map[string]calendarDayAccumulator{}
	for _, day := range days {
		dayByDate[day.date] = day
	}
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		key := date.Format("2006-01-02")
		day, ok := dayByDate[key]
		if ok && day.busyBlockCount > 0 {
			busyDays++
			if day.status == "fully_busy" {
				fullyBusyDays++
			} else {
				partiallyBusyDays++
			}
			daySummaries = append(daySummaries, appdto.CalendarBusyDaySummary{
				Date:           key,
				Status:         day.status,
				BusyHours:      roundBusyHours(day.busyHours),
				BusyBlockCount: day.busyBlockCount,
			})
		}
		if shouldMarkUnavailable(day, input.conversion) {
			unavailableDates = append(unavailableDates, key)
			continue
		}
		if input.conversion.IncludeWeekendsAsPreferredIfFree && !ok && (date.Weekday() == time.Saturday || date.Weekday() == time.Sunday) {
			freeWeekendDates = append(freeWeekendDates, key)
		}
	}
	warnings := append([]string{}, providerWarnings...)
	warnings = append(warnings, "Only busy/free information was imported. Event details are not stored.")
	return appdto.CalendarImportPreview{
		Source: calendarImportSourceGoogle,
		Range: appdto.CalendarImportRangeInfo{
			StartDate: input.startDate,
			EndDate:   input.endDate,
			Timezone:  input.timezone,
		},
		BusyBlocksSummary: appdto.CalendarBusyBlocksSummary{
			BusyBlockCount:    len(blocks),
			BusyDays:          busyDays,
			FullyBusyDays:     fullyBusyDays,
			PartiallyBusyDays: partiallyBusyDays,
		},
		SuggestedUnavailableRanges: rangesFromDateStrings(unavailableDates, "calendar_fully_busy"),
		SuggestedPreferredRanges:   rangesFromDateStrings(freeWeekendDates, "calendar_free_window"),
		DaySummaries:               daySummaries,
		Warnings:                   dedupeStrings(warnings),
	}
}

func calendarBusyDayAccumulators(input normalizedCalendarImportInput, blocks []calendarclient.FreeBusyBlock) []calendarDayAccumulator {
	loc, err := time.LoadLocation(input.timezone)
	if err != nil {
		loc = time.UTC
	}
	byDate := map[string]*calendarDayAccumulator{}
	for _, block := range blocks {
		if !block.End.After(block.Start) {
			continue
		}
		startDay := localCalendarDayStart(block.Start.In(loc))
		endDay := localCalendarDayStart(block.End.Add(-time.Nanosecond).In(loc))
		for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
			next := day.AddDate(0, 0, 1)
			overlapStart := maxTime(block.Start.In(loc), day)
			overlapEnd := minTime(block.End.In(loc), next)
			if !overlapEnd.After(overlapStart) {
				continue
			}
			key := day.Format("2006-01-02")
			acc := byDate[key]
			if acc == nil {
				acc = &calendarDayAccumulator{date: key}
				byDate[key] = acc
			}
			acc.busyHours += overlapEnd.Sub(overlapStart).Hours()
			acc.busyBlockCount++
			if block.AllDay {
				acc.allDay = true
			}
		}
	}
	out := make([]calendarDayAccumulator, 0, len(byDate))
	for _, acc := range byDate {
		if acc.busyHours > 24 {
			acc.busyHours = 24
		}
		if acc.allDay || acc.busyHours >= input.conversion.FullyBusyThresholdHours {
			acc.status = "fully_busy"
		} else {
			acc.status = "partially_busy"
		}
		out = append(out, *acc)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].date < out[j].date })
	return out
}

func shouldMarkUnavailable(day calendarDayAccumulator, conversion appdto.CalendarImportConversionSettings) bool {
	switch day.status {
	case "fully_busy":
		return conversion.MarkFullyBusyDaysUnavailable
	case "partially_busy":
		return conversion.MarkPartiallyBusyDaysUnavailable
	default:
		return false
	}
}

func rangesFromDateStrings(dates []string, reason string) []appdto.CalendarImportRange {
	if len(dates) == 0 {
		return []appdto.CalendarImportRange{}
	}
	sort.Strings(dates)
	ranges := []appdto.CalendarImportRange{}
	start := dates[0]
	prev := dates[0]
	for _, current := range dates[1:] {
		prevDate, _ := parseAvailabilityDate(prev)
		currentDate, _ := parseAvailabilityDate(current)
		if currentDate.Equal(prevDate.AddDate(0, 0, 1)) {
			prev = current
			continue
		}
		ranges = append(ranges, appdto.CalendarImportRange{StartDate: start, EndDate: prev, Reason: reason})
		start = current
		prev = current
	}
	ranges = append(ranges, appdto.CalendarImportRange{StartDate: start, EndDate: prev, Reason: reason})
	return ranges
}

func rangesWithoutReasons(ranges []appdto.CalendarImportRange) []entity.AvailabilityDateRange {
	out := make([]entity.AvailabilityDateRange, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, entity.AvailabilityDateRange{StartDate: r.StartDate, EndDate: r.EndDate})
	}
	return out
}

func mergeAvailabilityRanges(groups ...[]entity.AvailabilityDateRange) []entity.AvailabilityDateRange {
	parsed := []availabilityRange{}
	for _, ranges := range groups {
		parsed = append(parsed, parseRangesBestEffort(ranges)...)
	}
	if len(parsed) == 0 {
		return []entity.AvailabilityDateRange{}
	}
	sort.SliceStable(parsed, func(i, j int) bool {
		if parsed[i].start.Equal(parsed[j].start) {
			return parsed[i].end.Before(parsed[j].end)
		}
		return parsed[i].start.Before(parsed[j].start)
	})
	merged := []availabilityRange{parsed[0]}
	for _, r := range parsed[1:] {
		last := &merged[len(merged)-1]
		if !r.start.After(last.end.AddDate(0, 0, 1)) {
			if r.end.After(last.end) {
				last.end = r.end
				last.raw.EndDate = r.end.Format("2006-01-02")
			}
			continue
		}
		merged = append(merged, r)
	}
	out := make([]entity.AvailabilityDateRange, 0, len(merged))
	for _, r := range merged {
		out = append(out, entity.AvailabilityDateRange{
			StartDate: r.start.Format("2006-01-02"),
			EndDate:   r.end.Format("2006-01-02"),
		})
	}
	return out
}

func subtractOverriddenRanges(ranges []entity.AvailabilityDateRange, overrides []entity.AvailabilityDateRange) []entity.AvailabilityDateRange {
	if len(ranges) == 0 || len(overrides) == 0 {
		return ranges
	}
	out := []entity.AvailabilityDateRange{}
	for _, r := range parseRangesBestEffort(ranges) {
		segments := []availabilityRange{r}
		for _, override := range parseRangesBestEffort(overrides) {
			next := []availabilityRange{}
			for _, segment := range segments {
				if !rangesOverlap(segment.start, segment.end, override.start, override.end) {
					next = append(next, segment)
					continue
				}
				if override.start.After(segment.start) {
					next = append(next, availabilityRange{start: segment.start, end: override.start.AddDate(0, 0, -1)})
				}
				if override.end.Before(segment.end) {
					next = append(next, availabilityRange{start: override.end.AddDate(0, 0, 1), end: segment.end})
				}
			}
			segments = next
		}
		for _, segment := range segments {
			if !segment.end.Before(segment.start) {
				out = append(out, entity.AvailabilityDateRange{
					StartDate: segment.start.Format("2006-01-02"),
					EndDate:   segment.end.Format("2006-01-02"),
				})
			}
		}
	}
	return mergeAvailabilityRanges(out)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func localCalendarDayStart(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func roundBusyHours(value float64) float64 {
	return math.Round(value*10) / 10
}

func dedupeStrings(values []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
