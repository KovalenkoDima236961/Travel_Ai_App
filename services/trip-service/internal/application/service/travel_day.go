package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/verification"
)

const (
	travelStatusPlanned = "planned"
	travelStatusDone    = "done"
	travelStatusSkipped = "skipped"
	travelStatusDelayed = "delayed"
	maxTravelStatusNote = 280
)

// TravelDaySummary is intentionally compact and private. It is designed for
// the execution screen and avoids receipt OCR, calendar details, and raw
// provider responses that belong in their dedicated screens.
type TravelDaySummary struct {
	TripID            uuid.UUID                `json:"tripId"`
	Date              string                   `json:"date"`
	DayNumber         int                      `json:"dayNumber"`
	Mode              string                   `json:"mode"`
	Timezone          string                   `json:"timezone"`
	Trip              TravelDayTrip            `json:"trip"`
	Today             TravelDayToday           `json:"today"`
	NowNext           TravelDayNowNext         `json:"nowNext"`
	Timeline          []TravelDayTimelineItem  `json:"timeline"`
	Route             TravelDayRoute           `json:"route"`
	Weather           TravelDayWeather         `json:"weather"`
	Verification      TravelDayVerification    `json:"verification"`
	Checklist         TravelDayChecklist       `json:"checklist"`
	Reminders         TravelDayReminders       `json:"reminders"`
	Accommodation     *aggregate.Accommodation `json:"accommodation,omitempty"`
	Expenses          TravelDayExpenses        `json:"expenses"`
	Offline           TravelDayOffline         `json:"offline"`
	Permissions       TravelDayPermissions     `json:"permissions"`
	SectionErrors     []TravelDaySectionError  `json:"sectionErrors"`
	GeneratedAt       time.Time                `json:"generatedAt"`
	ItineraryRevision int                      `json:"itineraryRevision"`
}

type TravelDayTrip struct {
	Title       string `json:"title"`
	Destination string `json:"destination"`
	StartDate   string `json:"startDate,omitempty"`
	EndDate     string `json:"endDate,omitempty"`
	TripType    string `json:"tripType"`
}

type TravelDayToday struct {
	Title           string `json:"title"`
	PrimaryLocation string `json:"primaryLocation,omitempty"`
	Summary         string `json:"summary,omitempty"`
}

type TravelDayNowNext struct {
	CurrentItem    *TravelDayTimelineItem  `json:"currentItem,omitempty"`
	NextItem       *TravelDayTimelineItem  `json:"nextItem,omitempty"`
	AfterNextItems []TravelDayTimelineItem `json:"afterNextItems"`
}

type TravelDayTimelineItem struct {
	DayNumber         int                                `json:"dayNumber"`
	ItemIndex         int                                `json:"itemIndex"`
	ItemID            string                             `json:"itemId,omitempty"`
	StartTime         string                             `json:"startTime,omitempty"`
	EndTime           string                             `json:"endTime,omitempty"`
	Title             string                             `json:"title"`
	Type              string                             `json:"type"`
	Description       string                             `json:"description,omitempty"`
	LocationName      string                             `json:"locationName,omitempty"`
	Place             *aggregate.PlaceRef                `json:"place,omitempty"`
	SelectedTransport *aggregate.SelectedTransportOption `json:"selectedTransport,omitempty"`
	EstimatedCost     *aggregate.EstimatedCost           `json:"estimatedCost,omitempty"`
	TravelStatus      aggregate.TravelStatus             `json:"travelStatus"`
	Verification      []verification.Detail              `json:"verification,omitempty"`
	Actions           []TravelDayAction                  `json:"actions"`
}

type TravelDayAction struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Href  string `json:"href,omitempty"`
}

type TravelDayRoute struct {
	TodayLegs                []aggregate.RouteLeg                `json:"todayLegs"`
	SelectedTransportSummary []aggregate.SelectedTransportOption `json:"selectedTransportSummary"`
}

type TravelDayWeather struct {
	Summary  string                `json:"summary"`
	Warnings []verification.Detail `json:"warnings"`
}

type TravelDayVerification struct {
	Score       int                   `json:"score"`
	Level       verification.Level    `json:"level"`
	TopWarnings []verification.Detail `json:"topWarnings"`
	Unavailable bool                  `json:"unavailable"`
}

type TravelDayChecklist struct {
	DueToday []appdto.TripChecklistItemDTO `json:"dueToday"`
	Overdue  []appdto.TripChecklistItemDTO `json:"overdue"`
	Progress TravelDayProgress             `json:"progress"`
}

type TravelDayReminders struct {
	DueToday []appdto.TripReminderDTO `json:"dueToday"`
	Overdue  []appdto.TripReminderDTO `json:"overdue"`
}

type TravelDayProgress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

type TravelDayExpenses struct {
	TodayTotal       appdto.MoneyAmount `json:"todayTotal"`
	QuickAddDefaults struct {
		Currency string `json:"currency"`
	} `json:"quickAddDefaults"`
}

type TravelDayOffline struct {
	CacheRecommended bool `json:"cacheRecommended"`
}

type TravelDayPermissions struct {
	CanUpdateTravelStatus bool `json:"canUpdateTravelStatus"`
	CanAddExpense         bool `json:"canAddExpense"`
	CanUploadReceipt      bool `json:"canUploadReceipt"`
	CanEditTrip           bool `json:"canEditTrip"`
}

type TravelDaySectionError struct {
	Section string `json:"section"`
	Code    string `json:"code"`
}

// GetTravelDay returns an execution-focused summary for a single calendar
// date. The caller's explicit local date is authoritative; there is no GPS or
// background location use in this flow.
func (s *Service) GetTravelDay(ctx context.Context, tripID uuid.UUID, requestedDate string) (TravelDaySummary, error) {
	started := time.Now()
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return TravelDaySummary{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return TravelDaySummary{}, err
	}

	date, err := travelDayDate(requestedDate)
	if err != nil {
		return TravelDaySummary{}, err
	}
	cacheKey := summaryCacheKey("travel_day", trip, user.ID, access.Role(), date.Format("2006-01-02"))
	if cached, ok := s.summaryCache.get("travel_day", cacheKey); ok {
		if summary, valid := cached.(TravelDaySummary); valid {
			tripobs.RecordTravelDayRead("cache_hit", time.Since(started), access.Role())
			return summary, nil
		}
	}

	dayNumber, mode, day := travelDayForTrip(trip, date)
	summary := TravelDaySummary{
		TripID:            trip.ID,
		Date:              date.Format("2006-01-02"),
		DayNumber:         dayNumber,
		Mode:              mode,
		Timezone:          "UTC",
		Trip:              travelDayTrip(trip),
		Today:             travelDayToday(day),
		Timeline:          make([]TravelDayTimelineItem, 0),
		Route:             travelDayRoute(trip, date),
		Weather:           TravelDayWeather{Summary: "Weather has not been verified for this day.", Warnings: []verification.Detail{}},
		Verification:      TravelDayVerification{TopWarnings: []verification.Detail{}},
		Checklist:         TravelDayChecklist{DueToday: []appdto.TripChecklistItemDTO{}, Overdue: []appdto.TripChecklistItemDTO{}},
		Reminders:         TravelDayReminders{DueToday: []appdto.TripReminderDTO{}, Overdue: []appdto.TripReminderDTO{}},
		Expenses:          travelDayExpenses(trip.BudgetCurrency),
		Offline:           TravelDayOffline{CacheRecommended: true},
		Permissions:       travelDayPermissions(access),
		SectionErrors:     []TravelDaySectionError{},
		GeneratedAt:       time.Now().UTC(),
		ItineraryRevision: trip.ItineraryRevision,
	}

	readiness := verification.Response{}
	if s.verificationConfig.Enabled {
		readiness = s.verificationForTrip(ctx, trip)
		summary.Verification = TravelDayVerification{
			Score: readiness.Score, Level: readiness.Level, TopWarnings: travelDayWarnings(readiness.TopIssues),
		}
		summary.Weather = travelDayWeather(readiness)
	} else {
		summary.Verification.Unavailable = true
		summary.SectionErrors = append(summary.SectionErrors, TravelDaySectionError{Section: "verification", Code: "unavailable"})
	}
	if day != nil {
		summary.Timeline = travelDayTimeline(day, trip, readiness)
	}
	summary.NowNext = travelDayNowNext(summary.Timeline, date, time.Now().UTC())

	if checklist, loadErr := s.activeChecklistWithItems(ctx, tripID); loadErr == nil {
		summary.Checklist = travelDayChecklist(checklist, date)
	} else if !errors.Is(loadErr, domainerrs.ErrNotFound) {
		summary.SectionErrors = append(summary.SectionErrors, TravelDaySectionError{Section: "checklist", Code: "unavailable"})
	}

	if reminders, loadErr := s.repo.ListTripRemindersByTrip(ctx, tripID, entity.TripReminderFilters{}); loadErr == nil {
		summary.Reminders = travelDayReminders(reminders, date)
	} else {
		summary.SectionErrors = append(summary.SectionErrors, TravelDaySectionError{Section: "reminders", Code: "unavailable"})
	}

	if expenses, loadErr := s.repo.ListTripExpensesByTrip(ctx, tripID, appdto.ListExpensesInput{}); loadErr == nil {
		summary.Expenses.TodayTotal = travelDayExpenseTotal(expenses, date, trip.BudgetCurrency)
	} else {
		summary.SectionErrors = append(summary.SectionErrors, TravelDaySectionError{Section: "expenses", Code: "unavailable"})
	}

	s.summaryCache.set("travel_day", cacheKey, summary)
	tripobs.RecordTravelDayRead("computed", time.Since(started), access.Role())
	s.log.Info("travel day summary requested", zap.String("trip_id", tripID.String()), zap.String("role", access.Role()), zap.String("date", summary.Date), zap.Duration("duration", time.Since(started)))
	return summary, nil
}

// UpdateTravelItemStatus updates a single travel execution state. Unlike a
// planning edit, it deliberately does not reset approval or send notifications.
func (s *Service) UpdateTravelItemStatus(ctx context.Context, tripID uuid.UUID, dayNumber, itemIndex int, in appdto.UpdateTravelItemStatusInput) (*entity.Trip, aggregate.TravelStatus, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, aggregate.TravelStatus{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, aggregate.TravelStatus{}, err
	}
	if !access.CanEdit() {
		return nil, aggregate.TravelStatus{}, apperrs.ErrForbidden
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if !validTravelStatus(status) {
		return nil, aggregate.TravelStatus{}, apperrs.NewInvalidInput("status must be planned, done, skipped, or delayed")
	}
	note := strings.TrimSpace(in.Note)
	if len([]rune(note)) > maxTravelStatusNote {
		return nil, aggregate.TravelStatus{}, apperrs.NewInvalidInput("note must be at most %d characters", maxTravelStatusNote)
	}
	expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
	if err != nil {
		return nil, aggregate.TravelStatus{}, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, trip.ItineraryRevision); err != nil {
		return nil, aggregate.TravelStatus{}, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return nil, aggregate.TravelStatus{}, err
	}
	itinerary, dayIndex, err := currentItineraryAndDayIndex(trip, dayNumber)
	if err != nil {
		return nil, aggregate.TravelStatus{}, err
	}
	if itemIndex < 0 || itemIndex >= len(itinerary.Days[dayIndex].Items) {
		return nil, aggregate.TravelStatus{}, currentItineraryInvalidError()
	}
	travelStatus := aggregate.TravelStatus{Status: status, UpdatedAt: time.Now().UTC(), UpdatedByUserID: user.ID, Note: note}
	item := &itinerary.Days[dayIndex].Items[itemIndex]
	item.TravelStatus = &travelStatus
	raw, err := jsonMarshalTravelDayItinerary(itinerary)
	if err != nil {
		return nil, aggregate.TravelStatus{}, err
	}
	updated, _, err := s.repo.UpdateItineraryAndCreateVersion(ctx, tripID, ownerID, user.ID, raw, entity.StatusCompleted, expectedRevision, entity.ItineraryVersionSourceTravelStatusUpdated, map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex, "status": status})
	if err != nil {
		tripobs.RecordTravelStatusUpdateFailure(string(access.Level))
		return updated, aggregate.TravelStatus{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventItineraryItemStatusUpdated, EntityType: activityEntityType(activity.EntityItineraryItem), Metadata: map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex, "status": status, "itemTitle": item.Name}})
	tripobs.RecordTravelStatusUpdate(status, string(access.Level))
	s.log.Info("travel status updated", zap.String("trip_id", tripID.String()), zap.Int("day_number", dayNumber), zap.Int("item_index", itemIndex), zap.String("status", status))
	return updated, travelStatus, nil
}

func jsonMarshalTravelDayItinerary(itinerary aggregate.Itinerary) ([]byte, error) {
	return json.Marshal(itinerary)
}

func validTravelStatus(status string) bool {
	return status == travelStatusPlanned || status == travelStatusDone || status == travelStatusSkipped || status == travelStatusDelayed
}

func travelDayDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, apperrs.NewInvalidInput("date must be in YYYY-MM-DD format")
	}
	return date, nil
}

func travelDayForTrip(trip *entity.Trip, date time.Time) (int, string, *aggregate.ItineraryDay) {
	if trip == nil || trip.StartDate == nil {
		return 1, "pre_trip", nil
	}
	start := dateOnly(*trip.StartDate)
	if date.Before(start) {
		return 1, "pre_trip", nil
	}
	end := start.AddDate(0, 0, int(trip.Days)-1)
	if date.After(end) {
		return int(trip.Days), "post_trip", nil
	}
	dayNumber := int(date.Sub(start).Hours()/24) + 1
	itinerary := parseItineraryLenient(trip.Itinerary)
	for index := range itinerary.Days {
		candidate := &itinerary.Days[index]
		if candidate.Day == dayNumber || candidate.Date == date.Format("2006-01-02") {
			return dayNumber, "active", candidate
		}
	}
	return dayNumber, "active", nil
}

func travelDayTrip(trip *entity.Trip) TravelDayTrip {
	result := TravelDayTrip{Title: trip.Destination, Destination: trip.Destination, TripType: trip.TripType}
	if trip.StartDate != nil {
		result.StartDate = trip.StartDate.Format("2006-01-02")
		result.EndDate = dateOnly(*trip.StartDate).AddDate(0, 0, int(trip.Days)-1).Format("2006-01-02")
	}
	return result
}

func travelDayToday(day *aggregate.ItineraryDay) TravelDayToday {
	if day == nil {
		return TravelDayToday{Title: "No plan for this day"}
	}
	parts := make([]string, 0, min(len(day.Items), 4))
	for _, item := range day.Items {
		if title := strings.TrimSpace(item.Name); title != "" && len(parts) < 4 {
			parts = append(parts, title)
		}
	}
	return TravelDayToday{Title: day.Title, PrimaryLocation: day.LocationName, Summary: strings.Join(parts, ", ")}
}

func travelDayTimeline(day *aggregate.ItineraryDay, trip *entity.Trip, readiness verification.Response) []TravelDayTimelineItem {
	items := make([]TravelDayTimelineItem, 0, len(day.Items))
	for index := range day.Items {
		item := day.Items[index]
		status := travelStatusForItem(item)
		location := day.LocationName
		if item.Place != nil && item.Place.Name != "" {
			location = item.Place.Name
		}
		selectedTransport := selectedTransportForItem(trip, item)
		items = append(items, TravelDayTimelineItem{DayNumber: day.Day, ItemIndex: index, StartTime: item.Time, EndTime: item.EndTime, Title: item.Name, Type: item.Type, Description: item.Note, LocationName: location, Place: item.Place, SelectedTransport: selectedTransport, EstimatedCost: item.EstimatedCost, TravelStatus: status, Verification: verificationForTravelItem(readiness, day.Day, index), Actions: travelDayItemActions(item, status)})
	}
	return items
}

func travelStatusForItem(item aggregate.ItineraryItem) aggregate.TravelStatus {
	if item.TravelStatus == nil || !validTravelStatus(item.TravelStatus.Status) {
		return aggregate.TravelStatus{Status: travelStatusPlanned}
	}
	return *item.TravelStatus
}

func selectedTransportForItem(trip *entity.Trip, item aggregate.ItineraryItem) *aggregate.SelectedTransportOption {
	if trip == nil || trip.Route == nil {
		return nil
	}
	legID := ""
	if item.Transfer != nil {
		legID = item.Transfer.LegID
	}
	for _, leg := range trip.Route.Legs {
		if (legID != "" && leg.ID == legID) || (legID == "" && item.Type == "transport" && leg.SelectedTransportOption != nil) {
			return leg.SelectedTransportOption
		}
	}
	return nil
}

func verificationForTravelItem(readiness verification.Response, dayNumber, itemIndex int) []verification.Detail {
	matched := []verification.Detail{}
	needle := fmt.Sprintf("%d:%d", dayNumber, itemIndex)
	for _, issue := range readiness.TopIssues {
		if strings.Contains(issue.EntityID, needle) {
			matched = append(matched, issue)
		}
	}
	return matched
}

func travelDayItemActions(item aggregate.ItineraryItem, status aggregate.TravelStatus) []TravelDayAction {
	actions := []TravelDayAction{{Type: "open_map", Label: "Open map"}}
	if status.Status != travelStatusDone {
		actions = append(actions, TravelDayAction{Type: "mark_done", Label: "Mark done"})
	}
	if item.Transfer != nil || item.Type == "transport" {
		actions = append(actions, TravelDayAction{Type: "open_transport", Label: "Open transport"})
	}
	return actions
}

func travelDayNowNext(timeline []TravelDayTimelineItem, date, now time.Time) TravelDayNowNext {
	result := TravelDayNowNext{AfterNextItems: []TravelDayTimelineItem{}}
	if len(timeline) == 0 {
		return result
	}
	nowMinutes := now.Hour()*60 + now.Minute()
	isToday := date.Format("2006-01-02") == now.UTC().Format("2006-01-02")
	candidates := make([]TravelDayTimelineItem, 0, len(timeline))
	for _, item := range timeline {
		if item.TravelStatus.Status == travelStatusDone || item.TravelStatus.Status == travelStatusSkipped {
			continue
		}
		start, hasStart := parseHHMM(item.StartTime)
		end, hasEnd := parseHHMM(item.EndTime)
		if isToday && hasStart && hasEnd && nowMinutes >= start && nowMinutes <= end && result.CurrentItem == nil {
			current := item
			result.CurrentItem = &current
			continue
		}
		candidates = append(candidates, item)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, lok := parseHHMM(candidates[i].StartTime)
		right, rok := parseHHMM(candidates[j].StartTime)
		if lok && rok {
			return left < right
		}
		return lok && !rok
	})
	if len(candidates) > 0 {
		next := candidates[0]
		result.NextItem = &next
		if len(candidates) > 1 {
			limit := min(len(candidates), 4)
			result.AfterNextItems = candidates[1:limit]
		}
	}
	return result
}

func travelDayRoute(trip *entity.Trip, date time.Time) TravelDayRoute {
	result := TravelDayRoute{TodayLegs: []aggregate.RouteLeg{}, SelectedTransportSummary: []aggregate.SelectedTransportOption{}}
	if trip == nil || trip.Route == nil {
		return result
	}
	for _, leg := range trip.Route.Legs {
		if leg.DepartureDate == date.Format("2006-01-02") || (leg.SelectedTransportOption != nil && leg.SelectedTransportOption.DepartureDate == date.Format("2006-01-02")) {
			result.TodayLegs = append(result.TodayLegs, leg)
		}
		if leg.SelectedTransportOption != nil {
			result.SelectedTransportSummary = append(result.SelectedTransportSummary, *leg.SelectedTransportOption)
		}
	}
	return result
}

func travelDayWarnings(issues []verification.Detail) []verification.Detail {
	out := []verification.Detail{}
	for _, issue := range issues {
		if issue.Severity == verification.SeverityWarning || issue.Severity == verification.SeverityHigh || issue.Severity == verification.SeverityCritical {
			out = append(out, issue)
		}
	}
	return out
}

func travelDayWeather(readiness verification.Response) TravelDayWeather {
	warnings := []verification.Detail{}
	for _, issue := range readiness.TopIssues {
		if issue.Scope == verification.ScopeWeather {
			warnings = append(warnings, issue)
		}
	}
	if len(warnings) == 0 {
		return TravelDayWeather{Summary: "No high-signal weather warnings.", Warnings: warnings}
	}
	return TravelDayWeather{Summary: warnings[0].Message, Warnings: warnings}
}

func travelDayChecklist(checklist *entity.TripChecklist, date time.Time) TravelDayChecklist {
	result := TravelDayChecklist{DueToday: []appdto.TripChecklistItemDTO{}, Overdue: []appdto.TripChecklistItemDTO{}}
	if checklist == nil {
		return result
	}
	for index := range checklist.Items {
		item := checklist.Items[index]
		if item.Checked {
			result.Progress.Completed++
		}
		result.Progress.Total++
		if item.Checked || item.DueDate == nil {
			continue
		}
		dto := appdto.NewTripChecklistItemDTO(&item)
		due := dateOnly(*item.DueDate)
		if due.Equal(date) {
			result.DueToday = append(result.DueToday, dto)
		} else if due.Before(date) {
			result.Overdue = append(result.Overdue, dto)
		}
	}
	return result
}

func travelDayReminders(reminders []entity.TripReminder, date time.Time) TravelDayReminders {
	result := TravelDayReminders{DueToday: []appdto.TripReminderDTO{}, Overdue: []appdto.TripReminderDTO{}}
	for index := range reminders {
		reminder := reminders[index]
		if reminder.Status == entity.ReminderStatusCompleted || reminder.Status == entity.ReminderStatusDisabled || reminder.Status == entity.ReminderStatusCancelled {
			continue
		}
		due := dateOnly(reminder.TriggerDate)
		dto := appdto.NewTripReminderDTO(&reminder)
		if due.Equal(date) {
			result.DueToday = append(result.DueToday, dto)
		} else if due.Before(date) {
			result.Overdue = append(result.Overdue, dto)
		}
	}
	return result
}

func travelDayExpenses(currency string) TravelDayExpenses {
	result := TravelDayExpenses{TodayTotal: appdto.MoneyAmount{Currency: currency}}
	result.QuickAddDefaults.Currency = currency
	return result
}

func travelDayExpenseTotal(expenses []entity.TripExpense, date time.Time, currency string) appdto.MoneyAmount {
	total := appdto.MoneyAmount{Currency: currency}
	for _, expense := range expenses {
		if expense.Status == entity.ExpenseStatusActive && dateOnly(expense.ExpenseDate).Equal(date) && strings.EqualFold(expense.Currency, currency) {
			total.Amount += expense.Amount
		}
	}
	return total
}

func travelDayPermissions(access TripAccess) TravelDayPermissions {
	return TravelDayPermissions{CanUpdateTravelStatus: access.CanEdit(), CanAddExpense: access.CanView(), CanUploadReceipt: access.CanView(), CanEditTrip: access.CanEdit()}
}
