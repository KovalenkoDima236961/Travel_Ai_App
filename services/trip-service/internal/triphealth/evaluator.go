package triphealth

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/verification"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

const (
	soonWindowDays         = 14
	activityDenseThreshold = 8
)

var transportOptionRequiredModes = map[string]struct{}{
	aggregate.TransportModeTrain:           {},
	aggregate.TransportModeBus:             {},
	aggregate.TransportModeFlight:          {},
	aggregate.TransportModeFerry:           {},
	aggregate.TransportModeBoat:            {},
	aggregate.TransportModeRentalCar:       {},
	aggregate.TransportModePublicTransport: {},
}

type issueBuilder struct {
	tripID uuid.UUID
	issues []Issue
	seen   map[string]struct{}
}

func Evaluate(snapshot Snapshot, options Options) Response {
	now := snapshot.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cfg := snapshot.Config
	if cfg == (Config{}) {
		cfg = DefaultConfig()
	}
	if cfg.LargeExpenseReceiptThreshold <= 0 {
		cfg.LargeExpenseReceiptThreshold = DefaultConfig().LargeExpenseReceiptThreshold
	}
	if cfg.DefaultMaxWalkingKmPerDay <= 0 {
		cfg.DefaultMaxWalkingKmPerDay = DefaultConfig().DefaultMaxWalkingKmPerDay
	}
	if cfg.DefaultMaxTransferMinutesPerDay <= 0 {
		cfg.DefaultMaxTransferMinutesPerDay = DefaultConfig().DefaultMaxTransferMinutesPerDay
	}

	tripID := uuid.Nil
	if snapshot.Trip != nil {
		tripID = snapshot.Trip.ID
	}
	builder := issueBuilder{tripID: tripID, seen: map[string]struct{}{}}
	builder.addSubsystemFailures(snapshot)
	builder.evaluateItinerary(snapshot, cfg, now)
	builder.evaluateRoute(snapshot, cfg, now)
	builder.evaluateTransport(snapshot, cfg, now)
	builder.evaluateBudget(snapshot, now)
	builder.evaluateAvailability(snapshot, now)
	builder.evaluateCollaboration(snapshot, now)
	builder.evaluateChecklist(snapshot, now)
	builder.evaluateReminders(snapshot, now)
	builder.evaluateAccommodation(snapshot, now)
	builder.evaluateExpenses(snapshot, cfg, now)
	builder.evaluatePolicy(snapshot)
	builder.evaluateApproval(snapshot, now)
	builder.evaluateVerification(snapshot)
	builder.evaluateDataQuality(snapshot)

	issues := builder.issues
	sortIssues(issues)
	if !options.IncludeResolved {
		issues = filterOpenIssues(issues)
	}
	score, categories := ScoreIssues(issues)
	level := ReadinessLevel(score, issues)

	resp := Response{
		TripID:       tripID,
		Score:        score,
		Level:        level,
		Summary:      Summary(level, issues),
		GeneratedAt:  now,
		Categories:   categories,
		Issues:       issues,
		TopFixes:     TopFixes(issues, 7),
		ComputedFrom: computedFrom(snapshot),
	}
	if options.IncludeDebug || cfg.IncludeDebug {
		resp.Debug = map[string]any{
			"issueCount":        len(issues),
			"subsystemFailures": append([]string(nil), snapshot.SubsystemFailures...),
		}
	}
	return resp
}

// evaluateVerification brings only the most important real-world gaps into
// Trip Health. Full details stay on the dedicated verification response, so
// these signals complement rather than dominate existing health checks.
func (b *issueBuilder) evaluateVerification(snapshot Snapshot) {
	if snapshot.Verification == nil {
		return
	}
	added := 0
	for _, detail := range snapshot.Verification.TopIssues {
		if added == 3 || detail.Status == verification.StatusVerified || detail.Status == verification.StatusNotApplicable {
			continue
		}
		var actionValue *Action
		if detail.Action != nil {
			actionValue = &Action{Type: detail.Action.Type, Label: detail.Action.Label, Href: detail.Action.Href}
		}
		b.issue(
			"verification_"+string(detail.Scope)+"_"+string(detail.Status)+":"+normalizeToken(detail.EntityID),
			verificationCategory(detail.Scope),
			verificationSeverity(detail),
			verificationIssueTitle(detail),
			detail.Message,
			"Real-world trip data should be reviewed before relying on it.",
			"Review the verification details and refresh provider data when available.",
			actionValue,
			map[string]any{"scope": detail.Scope, "status": detail.Status, "source": detail.Source},
		)
		added++
	}
}

func verificationCategory(scope verification.Scope) Category {
	switch scope {
	case verification.ScopeTransport:
		return CategoryTransport
	case verification.ScopeRouteEstimate:
		return CategoryRoute
	case verification.ScopePrice:
		return CategoryBudget
	case verification.ScopeAvailability:
		return CategoryAvailability
	case verification.ScopeAccommodation:
		return CategoryAccommodation
	default:
		return CategoryDataQuality
	}
}

func verificationSeverity(detail verification.Detail) Severity {
	if detail.Status == verification.StatusUnavailable {
		if detail.Scope == verification.ScopeTransport {
			return SeverityCritical
		}
		return SeverityHigh
	}
	if detail.Status == verification.StatusMissing && detail.Scope == verification.ScopeTransport {
		return SeverityHigh
	}
	return SeverityWarning
}

func verificationIssueTitle(detail verification.Detail) string {
	prefix := "Verification needs review"
	switch detail.Status {
	case verification.StatusStale:
		prefix = "Verification is stale"
	case verification.StatusMissing:
		prefix = "Verification data is missing"
	case verification.StatusUnavailable:
		prefix = "Provider reports unavailable"
	case verification.StatusEstimated:
		prefix = "Data is estimated"
	case verification.StatusFailed:
		prefix = "Verification failed"
	}
	if detail.Title == "" {
		return prefix
	}
	return prefix + ": " + detail.Title
}

func (b *issueBuilder) add(issue Issue) {
	if issue.ID == "" {
		return
	}
	if issue.Status == "" {
		issue.Status = StatusOpen
	}
	if issue.Metadata == nil {
		issue.Metadata = map[string]any{}
	}
	if _, exists := b.seen[issue.ID]; exists {
		return
	}
	b.seen[issue.ID] = struct{}{}
	b.issues = append(b.issues, issue)
}

func (b *issueBuilder) issue(
	id string,
	category Category,
	severity Severity,
	title string,
	description string,
	impact string,
	recommendation string,
	action *Action,
	metadata map[string]any,
) {
	b.add(Issue{
		ID:             id,
		Category:       category,
		Severity:       severity,
		Status:         StatusOpen,
		Title:          title,
		Description:    description,
		Impact:         impact,
		Recommendation: recommendation,
		Action:         action,
		Metadata:       metadata,
	})
}

func (b *issueBuilder) addSubsystemFailures(snapshot Snapshot) {
	for _, name := range snapshot.SubsystemFailures {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := normalizeToken(name)
		b.issue(
			"health_subsystem_unavailable:"+key,
			CategoryDataQuality,
			SeverityWarning,
			"Could not evaluate "+name+" health",
			"The health engine could not load this subsystem, so the score may be incomplete.",
			"Some readiness checks may be missing from this response.",
			"Retry after the subsystem is available.",
			nil,
			map[string]any{"subsystem": name},
		)
	}
}

func (b *issueBuilder) evaluateItinerary(snapshot Snapshot, cfg Config, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	itinerary := snapshot.Itinerary
	if len(itinerary.Days) == 0 {
		severity := SeverityHigh
		if isApprovalActive(snapshot.Approval) {
			severity = SeverityCritical
		}
		b.issue(
			"itinerary_missing",
			CategoryItinerary,
			severity,
			"Missing itinerary",
			"This trip does not have a saved itinerary.",
			"The trip has no day-by-day plan to validate.",
			"Generate or add an itinerary before departure.",
			action(b.tripID, "open_itinerary", "Open itinerary", "itinerary"),
			nil,
		)
		return
	}
	if trip.Days > 0 && len(itinerary.Days) != int(trip.Days) {
		b.issue(
			"itinerary_day_count_mismatch",
			CategoryItinerary,
			SeverityWarning,
			"Itinerary day count does not match trip duration",
			fmt.Sprintf("The trip is set to %d day(s), but the itinerary has %d day(s).", trip.Days, len(itinerary.Days)),
			"The trip timeline, budget, and reminders may not line up.",
			"Adjust the trip duration or regenerate the itinerary.",
			action(b.tripID, "open_itinerary", "Review itinerary", "itinerary"),
			map[string]any{"tripDays": int(trip.Days), "itineraryDays": len(itinerary.Days)},
		)
	}

	routeStops := routeStopIDs(trip.Route)
	legIntervals := selectedTransportIntervals(trip.Route)
	for dayIndex, day := range itinerary.Days {
		dayNumber := day.Day
		if dayNumber <= 0 {
			dayNumber = dayIndex + 1
			b.issue(
				fmt.Sprintf("itinerary_invalid_day_number:%d", dayIndex),
				CategoryItinerary,
				SeverityWarning,
				"Invalid itinerary day number",
				fmt.Sprintf("Itinerary day at position %d has an invalid day number.", dayIndex+1),
				"Sorting and day-specific actions may point to the wrong day.",
				"Save the itinerary with sequential day numbers.",
				action(b.tripID, "open_itinerary", "Review day", "itinerary"),
				map[string]any{"dayIndex": dayIndex},
			)
		}
		if len(day.Items) == 0 {
			b.issue(
				fmt.Sprintf("itinerary_empty_day:%d", dayNumber),
				CategoryItinerary,
				SeverityInfo,
				fmt.Sprintf("Day %d has no planned items", dayNumber),
				"This itinerary day is empty.",
				"The plan may omit travel or activity details for this day.",
				"Add planned items or confirm this is an intentional free day.",
				action(b.tripID, "open_itinerary", "Open day", "itinerary"),
				map[string]any{"dayNumber": dayNumber},
			)
		}
		if len(day.Items) > activityDenseThreshold {
			severity := SeverityWarning
			if len(day.Items) >= 11 {
				severity = SeverityHigh
			}
			b.issue(
				fmt.Sprintf("day_too_packed:%d", dayNumber),
				CategoryItinerary,
				severity,
				fmt.Sprintf("Day %d may be too packed", dayNumber),
				fmt.Sprintf("Day %d has %d planned items.", dayNumber, len(day.Items)),
				"A dense plan can make transfers, meals, and delays difficult.",
				"Remove lower-priority activities or use AI repair to rebalance the day.",
				action(b.tripID, "regenerate_day", "Rebalance day", "itinerary"),
				map[string]any{"dayNumber": dayNumber, "itemCount": len(day.Items)},
			)
		}
		if routeStops != nil {
			if day.PrimaryStopID == "" && day.LocationName == "" {
				b.issue(
					fmt.Sprintf("route_day_without_stop:%d", dayNumber),
					CategoryItinerary,
					SeverityWarning,
					fmt.Sprintf("Day %d is not tied to a route stop", dayNumber),
					"The day has no primary route stop or location name.",
					"Route and itinerary consistency checks may be less accurate.",
					"Assign the day to a route stop or add a location name.",
					action(b.tripID, "open_route", "Review route", "route"),
					map[string]any{"dayNumber": dayNumber},
				)
			} else if day.PrimaryStopID != "" {
				if _, ok := routeStops[day.PrimaryStopID]; !ok {
					b.issue(
						fmt.Sprintf("itinerary_route_stop_mismatch:%d", dayNumber),
						CategoryItinerary,
						SeverityHigh,
						fmt.Sprintf("Day %d references an unknown route stop", dayNumber),
						fmt.Sprintf("The day references route stop %q, which is not in the current route.", day.PrimaryStopID),
						"Activities may be scheduled in the wrong city or stop.",
						"Update the route stop assignment or regenerate the affected day.",
						action(b.tripID, "open_route", "Review route", "route"),
						map[string]any{"dayNumber": dayNumber, "primaryStopId": day.PrimaryStopID},
					)
				}
			}
		}

		b.evaluateItineraryDayTimes(dayNumber, day.Items)
		walkingKm := dayWalkingKm(day)
		if walkingKm > cfg.DefaultMaxWalkingKmPerDay {
			severity := SeverityWarning
			if walkingKm > cfg.DefaultMaxWalkingKmPerDay*1.5 {
				severity = SeverityHigh
			}
			b.issue(
				fmt.Sprintf("walking_distance_high:%d", dayNumber),
				CategoryItinerary,
				severity,
				fmt.Sprintf("Day %d has high walking distance", dayNumber),
				fmt.Sprintf("Planned walking is %.1f km, above the %.1f km default threshold.", walkingKm, cfg.DefaultMaxWalkingKmPerDay),
				"The day may be tiring or unrealistic for some travelers.",
				"Reduce walking distance or add transport between activities.",
				action(b.tripID, "regenerate_day", "Reduce walking", "itinerary"),
				map[string]any{"dayNumber": dayNumber, "walkingDistanceKm": round1(walkingKm)},
			)
		}
		for itemIndex, item := range day.Items {
			if item.Place != nil && (item.Place.Latitude == nil || item.Place.Longitude == nil) {
				b.issue(
					fmt.Sprintf("missing_place_coordinates:%d:%d", dayNumber, itemIndex),
					CategoryDataQuality,
					SeverityInfo,
					"Place is missing coordinates",
					fmt.Sprintf("%q does not have map coordinates.", item.Name),
					"Map, route, and walking estimates may be incomplete.",
					"Review the place match for this itinerary item.",
					action(b.tripID, "open_itinerary", "Review item", "itinerary"),
					map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex},
				)
			}
			if item.PlaceEnrichment != nil && item.PlaceEnrichment.Confidence > 0 && item.PlaceEnrichment.Confidence < 0.55 {
				b.issue(
					fmt.Sprintf("low_confidence_place_match:%d:%d", dayNumber, itemIndex),
					CategoryDataQuality,
					SeverityWarning,
					"Low-confidence place match",
					fmt.Sprintf("%q has a low-confidence place match.", item.Name),
					"Availability, map, and opening-hours checks may use the wrong place.",
					"Review or replace the place match.",
					action(b.tripID, "open_itinerary", "Review place", "itinerary"),
					map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex, "confidence": item.PlaceEnrichment.Confidence},
				)
			}
			if item.PriceEnrichment != nil && normalizeToken(item.PriceEnrichment.Status) == "failed" {
				b.issue(
					fmt.Sprintf("provider_data_unavailable:%d:%d", dayNumber, itemIndex),
					CategoryDataQuality,
					SeverityInfo,
					"Provider price data unavailable",
					fmt.Sprintf("Provider pricing could not be loaded for %q.", item.Name),
					"Budget confidence may be lower for this item.",
					"Review or add a manual cost estimate.",
					action(b.tripID, "open_budget", "Review budget", "budget"),
					map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex},
				)
			}
		}
		b.evaluateTransportConflicts(dayNumber, day.Items, legIntervals)
	}
	_ = now
}

func (b *issueBuilder) evaluateItineraryDayTimes(dayNumber int, items []aggregate.ItineraryItem) {
	type interval struct {
		start int
		end   int
		index int
		name  string
	}
	intervals := make([]interval, 0, len(items))
	for itemIndex, item := range items {
		start, ok := parseClockMinutes(item.Time)
		if !ok {
			continue
		}
		end := start
		if item.EndTime != "" {
			parsedEnd, ok := parseClockMinutes(item.EndTime)
			if !ok {
				continue
			}
			end = parsedEnd
		} else if item.DurationMinutes != nil && *item.DurationMinutes > 0 {
			end = start + *item.DurationMinutes
		} else {
			end = start + 60
		}
		if end < start {
			b.issue(
				fmt.Sprintf("itinerary_invalid_time_order:%d:%d", dayNumber, itemIndex),
				CategoryItinerary,
				SeverityHigh,
				"Activity ends before it starts",
				fmt.Sprintf("%q on Day %d has an end time before its start time.", item.Name, dayNumber),
				"This makes the schedule impossible.",
				"Fix the activity time or regenerate the day.",
				action(b.tripID, "open_itinerary", "Fix time", "itinerary"),
				map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex},
			)
			continue
		}
		if start >= 23*60 && !isTransportLike(item) {
			b.issue(
				fmt.Sprintf("late_activity_policy_warning:%d:%d", dayNumber, itemIndex),
				CategoryItinerary,
				SeverityInfo,
				"Very late activity",
				fmt.Sprintf("%q starts at %s.", item.Name, item.Time),
				"Late activities can conflict with rest time or workspace policy.",
				"Confirm this timing is intentional.",
				action(b.tripID, "open_itinerary", "Review timing", "itinerary"),
				map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex},
			)
		}
		intervals = append(intervals, interval{start: start, end: end, index: itemIndex, name: item.Name})
	}
	sort.SliceStable(intervals, func(i, j int) bool {
		if intervals[i].start == intervals[j].start {
			return intervals[i].index < intervals[j].index
		}
		return intervals[i].start < intervals[j].start
	})
	for i := 1; i < len(intervals); i++ {
		prev := intervals[i-1]
		current := intervals[i]
		if current.start < prev.end {
			b.issue(
				fmt.Sprintf("itinerary_overlapping_items:%d:%d:%d", dayNumber, prev.index, current.index),
				CategoryItinerary,
				SeverityHigh,
				"Overlapping itinerary items",
				fmt.Sprintf("Day %d has overlapping items: %q and %q.", dayNumber, prev.name, current.name),
				"The schedule cannot be followed as written.",
				"Move one item or regenerate the day.",
				action(b.tripID, "open_itinerary", "Fix overlap", "itinerary"),
				map[string]any{"dayNumber": dayNumber, "itemIndex": current.index},
			)
		}
	}
}

func (b *issueBuilder) evaluateRoute(snapshot Snapshot, cfg Config, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	route := trip.Route
	if trip.TripType == entity.TripTypeMultiDestination && route == nil {
		b.issue(
			"route_missing",
			CategoryRoute,
			SeverityHigh,
			"Missing multi-destination route",
			"This multi-destination trip has no route.",
			"Transfers, timing, and transport costs cannot be validated.",
			"Add route stops and legs.",
			action(b.tripID, "open_route", "Open route", "route"),
			nil,
		)
		return
	}
	if route == nil {
		return
	}
	if trip.TripType == entity.TripTypeMultiDestination && len(route.Stops) < 2 {
		b.issue(
			"route_incomplete:stops",
			CategoryRoute,
			SeverityHigh,
			"Route has too few stops",
			"A multi-destination trip needs at least two route stops.",
			"The route cannot describe inter-city movement.",
			"Add another route stop or change the trip type.",
			action(b.tripID, "open_route", "Review route", "route"),
			map[string]any{"stopCount": len(route.Stops)},
		)
	}
	stopIDs := routeStopIDs(route)
	for index, leg := range route.Legs {
		if strings.TrimSpace(leg.FromStopID) == "" || strings.TrimSpace(leg.ToStopID) == "" || strings.TrimSpace(leg.Mode) == "" {
			b.issue(
				fmt.Sprintf("route_incomplete:leg_%d", index+1),
				CategoryRoute,
				SeverityHigh,
				"Incomplete route leg",
				"A route leg is missing an origin, destination, or mode.",
				"Transport search and route timing may fail for this leg.",
				"Complete the route leg details.",
				action(b.tripID, "open_route", "Fix route leg", "route"),
				map[string]any{"routeLegId": legIDOrIndex(leg, index)},
			)
		}
		if stopIDs != nil {
			if _, ok := stopIDs[leg.ToStopID]; leg.ToStopID != "" && !ok {
				b.issue(
					fmt.Sprintf("route_incomplete:%s:to", legIDOrIndex(leg, index)),
					CategoryRoute,
					SeverityHigh,
					"Route leg references an unknown stop",
					"The route leg destination does not match a route stop.",
					"The route may send travelers to a removed destination.",
					"Update or remove the stale route leg.",
					action(b.tripID, "open_route", "Fix route leg", "route"),
					map[string]any{"routeLegId": legIDOrIndex(leg, index), "toStopId": leg.ToStopID},
				)
			}
		}
	}
	if trip.TripType == entity.TripTypeMultiDestination && len(route.Legs) < maxInt(len(route.Stops)-1, 0) {
		b.issue(
			"route_incomplete:missing_legs",
			CategoryRoute,
			SeverityHigh,
			"Route is missing transfer legs",
			"The route has fewer legs than expected for its stops.",
			"Some destination transfers may be missing from the plan.",
			"Add transfer legs between route stops.",
			action(b.tripID, "open_route", "Add route legs", "route"),
			map[string]any{"stopCount": len(route.Stops), "legCount": len(route.Legs)},
		)
	}
	for index, stop := range route.Stops {
		if stop.ArrivalDate != "" && !validDate(stop.ArrivalDate) {
			b.issue(
				fmt.Sprintf("route_stop_dates_invalid:%s:arrival", stopIDOrIndex(stop, index)),
				CategoryRoute,
				SeverityHigh,
				"Route stop has an invalid arrival date",
				fmt.Sprintf("%q has an invalid arrival date.", routeStopName(stop)),
				"Route timing cannot be validated reliably.",
				"Fix the route stop dates.",
				action(b.tripID, "open_route", "Fix dates", "route"),
				map[string]any{"stopId": stopIDOrIndex(stop, index)},
			)
		}
		if stop.DepartureDate != "" && !validDate(stop.DepartureDate) {
			b.issue(
				fmt.Sprintf("route_stop_dates_invalid:%s:departure", stopIDOrIndex(stop, index)),
				CategoryRoute,
				SeverityHigh,
				"Route stop has an invalid departure date",
				fmt.Sprintf("%q has an invalid departure date.", routeStopName(stop)),
				"Route timing cannot be validated reliably.",
				"Fix the route stop dates.",
				action(b.tripID, "open_route", "Fix dates", "route"),
				map[string]any{"stopId": stopIDOrIndex(stop, index)},
			)
		}
	}
	if trip.Days > 0 && len(route.Stops) > int(trip.Days)+1 {
		b.issue(
			"too_many_stops_for_duration",
			CategoryRoute,
			SeverityWarning,
			"Many route stops for the trip duration",
			fmt.Sprintf("The route has %d stops for a %d-day trip.", len(route.Stops), trip.Days),
			"The trip may spend too much time transferring.",
			"Remove stops or extend the trip duration.",
			action(b.tripID, "open_route", "Review stops", "route"),
			map[string]any{"stopCount": len(route.Stops), "tripDays": int(trip.Days)},
		)
	}
	if trip.StartDate != nil && trip.Days > 0 {
		tripEnd := dateOnly(*trip.StartDate).AddDate(0, 0, int(trip.Days)-1)
		for index, stop := range route.Stops {
			for _, raw := range []struct {
				label string
				date  string
			}{
				{"arrival", stop.ArrivalDate},
				{"departure", stop.DepartureDate},
			} {
				if raw.date == "" {
					continue
				}
				parsed, ok := parseDate(raw.date)
				if !ok {
					continue
				}
				if parsed.Before(dateOnly(*trip.StartDate)) || parsed.After(tripEnd) {
					b.issue(
						fmt.Sprintf("route_duration_mismatch:%s:%s", stopIDOrIndex(stop, index), raw.label),
						CategoryRoute,
						SeverityWarning,
						"Route stop date is outside trip dates",
						fmt.Sprintf("%q has a %s date outside the trip date range.", routeStopName(stop), raw.label),
						"Route and itinerary dates may not align.",
						"Update the route stop dates or trip dates.",
						action(b.tripID, "open_route", "Review route dates", "route"),
						map[string]any{"stopId": stopIDOrIndex(stop, index), "date": raw.date},
					)
				}
			}
		}
	}
	_ = cfg
	_ = now
}

func (b *issueBuilder) evaluateTransport(snapshot Snapshot, cfg Config, now time.Time) {
	trip := snapshot.Trip
	if trip == nil || trip.Route == nil {
		return
	}
	soon := tripStartsWithin(trip, now, soonWindowDays)
	budgetAmount := 0.0
	if trip.BudgetAmount != nil {
		budgetAmount = *trip.BudgetAmount
	}
	for index, leg := range trip.Route.Legs {
		mode := aggregate.NormalizeRouteToken(leg.Mode)
		legID := legIDOrIndex(leg, index)
		if _, required := transportOptionRequiredModes[mode]; required && leg.SelectedTransportOption == nil {
			severity := SeverityWarning
			if soon {
				severity = SeverityHigh
			}
			b.issue(
				"transport_missing_option:"+legID,
				CategoryTransport,
				severity,
				"Missing selected transport option",
				fmt.Sprintf("The %s route leg has no selected transport option.", routeLegName(leg)),
				"The itinerary may underestimate transfer time and budget.",
				"Search and attach a transport option for this route leg.",
				action(b.tripID, "open_route_leg_transport_search", "Find transport", "route"),
				map[string]any{"routeLegId": legID},
			)
		}
		option := leg.SelectedTransportOption
		if option == nil {
			continue
		}
		if normalizeToken(option.Confidence) == budget.ConfidenceLow {
			severity := SeverityWarning
			if option.DurationMinutes >= 180 {
				severity = SeverityHigh
			}
			b.issue(
				"transport_low_confidence:"+legID,
				CategoryTransport,
				severity,
				"Low-confidence transport option",
				fmt.Sprintf("The selected transport option for %s has low confidence.", routeLegName(leg)),
				"Actual departure time, duration, or cost may differ.",
				"Review provider details or search for another option.",
				action(b.tripID, "open_transport", "Review transport", "route"),
				map[string]any{"routeLegId": legID, "confidence": option.Confidence},
			)
		}
		if normalizeToken(option.Provider) == "mock" {
			b.issue(
				"transport_mock_option:"+legID,
				CategoryTransport,
				SeverityWarning,
				"Mock transport option selected",
				fmt.Sprintf("The selected transport option for %s came from the mock provider.", routeLegName(leg)),
				"Mock transport may not reflect real schedules or pricing.",
				"Replace it with a real provider option before booking.",
				action(b.tripID, "open_route_leg_transport_search", "Find real transport", "route"),
				map[string]any{"routeLegId": legID, "provider": option.Provider},
			)
		}
		if normalizeToken(option.Status) == "unavailable" {
			b.issue(
				"transport_unavailable:"+legID,
				CategoryTransport,
				SeverityHigh,
				"Selected transport is unavailable",
				fmt.Sprintf("The selected transport option for %s is marked unavailable.", routeLegName(leg)),
				"The transfer may not be bookable.",
				"Choose another transport option.",
				action(b.tripID, "open_route_leg_transport_search", "Find replacement", "route"),
				map[string]any{"routeLegId": legID},
			)
		}
		duration := option.DurationMinutes
		if duration == 0 && leg.EstimatedDurationMinutes != nil {
			duration = *leg.EstimatedDurationMinutes
		}
		if duration > cfg.DefaultMaxTransferMinutesPerDay {
			severity := SeverityWarning
			if duration > cfg.DefaultMaxTransferMinutesPerDay+4*60 {
				severity = SeverityHigh
			}
			b.issue(
				"transport_duration_high:"+legID,
				CategoryTransport,
				severity,
				"Long transfer duration",
				fmt.Sprintf("The %s transfer is estimated at %s.", routeLegName(leg), formatDuration(duration)),
				"Long transfers can crowd out activities and rest time.",
				"Review route alternatives or split this transfer across days.",
				action(b.tripID, "open_route", "Review route", "route"),
				map[string]any{"routeLegId": legID, "durationMinutes": duration},
			)
		}
		if budgetAmount > 0 && option.EstimatedPrice != nil && option.EstimatedPrice.Amount > budgetAmount*0.3 {
			severity := SeverityWarning
			if option.EstimatedPrice.Amount > budgetAmount*0.5 {
				severity = SeverityHigh
			}
			b.issue(
				"transport_cost_high:"+legID,
				CategoryTransport,
				severity,
				"Transport leg is expensive",
				fmt.Sprintf("The selected option for %s is %.2f %s against a %.2f %s trip budget.", routeLegName(leg), option.EstimatedPrice.Amount, option.EstimatedPrice.Currency, budgetAmount, trip.BudgetCurrency),
				"This transfer could consume a large share of the trip budget.",
				"Review cheaper route alternatives or update the budget.",
				action(b.tripID, "open_budget", "Review budget", "budget"),
				map[string]any{"routeLegId": legID, "amount": option.EstimatedPrice.Amount, "currency": option.EstimatedPrice.Currency},
			)
		}
	}
}

func (b *issueBuilder) evaluateTransportConflicts(dayNumber int, items []aggregate.ItineraryItem, intervals map[string]transportInterval) {
	if len(intervals) == 0 {
		return
	}
	for itemIndex, item := range items {
		if item.Transfer != nil && item.Transfer.LegID != "" {
			continue
		}
		if isTransportLike(item) {
			continue
		}
		start, ok := parseClockMinutes(item.Time)
		if !ok {
			continue
		}
		end := start + 60
		if item.EndTime != "" {
			if parsed, ok := parseClockMinutes(item.EndTime); ok {
				end = parsed
			}
		} else if item.DurationMinutes != nil && *item.DurationMinutes > 0 {
			end = start + *item.DurationMinutes
		}
		date := itemDateFromDayNumber(dayNumber)
		for legID, interval := range intervals {
			if interval.dayNumber != 0 {
				if interval.dayNumber != dayNumber {
					continue
				}
			} else if interval.date != "" && date != "" && interval.date != date {
				continue
			}
			if rangesOverlap(start, end, interval.start, interval.end) {
				b.issue(
					"transport_itinerary_time_conflict:"+legID,
					CategoryTransport,
					SeverityHigh,
					"Transport overlaps an itinerary activity",
					fmt.Sprintf("Selected transport for leg %s overlaps %q on Day %d.", legID, item.Name, dayNumber),
					"Travelers cannot be in transit and at the activity at the same time.",
					"Move the activity or select a different transport option.",
					action(b.tripID, "open_itinerary", "Fix schedule", "itinerary"),
					map[string]any{"routeLegId": legID, "dayNumber": dayNumber, "itemIndex": itemIndex},
				)
			}
		}
	}
}

func (b *issueBuilder) evaluateBudget(snapshot Snapshot, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	if snapshot.BudgetLoadFailed {
		b.issue(
			"health_subsystem_unavailable:budget",
			CategoryDataQuality,
			SeverityWarning,
			"Could not evaluate budget health",
			"Budget summary could not be computed.",
			"Budget readiness may be incomplete.",
			"Retry after budget conversion or summary services are available.",
			nil,
			map[string]any{"subsystem": "budget"},
		)
		return
	}
	if trip.BudgetAmount == nil {
		b.issue(
			"budget_missing",
			CategoryBudget,
			SeverityWarning,
			"Trip budget is missing",
			"This trip does not have a budget amount.",
			"Budget warnings and affordability checks are less useful.",
			"Set a trip budget.",
			action(b.tripID, "open_budget", "Set budget", "budget"),
			nil,
		)
	}
	summary := snapshot.Budget
	if summary == nil {
		return
	}
	if summary.OverBudgetBy != nil && *summary.OverBudgetBy > 0 {
		severity := SeverityWarning
		if trip.BudgetAmount != nil && *trip.BudgetAmount > 0 && (*summary.OverBudgetBy / *trip.BudgetAmount) >= 0.1 {
			severity = SeverityHigh
		}
		b.issue(
			"estimated_budget_exceeded",
			CategoryBudget,
			severity,
			"Estimated budget is exceeded",
			fmt.Sprintf("Estimated trip cost is %.2f %s, %.2f %s over budget.", summary.EstimatedTotal, summary.Currency, *summary.OverBudgetBy, summary.Currency),
			"The plan may not fit the budget.",
			"Reduce costs or update the trip budget.",
			action(b.tripID, "open_budget", "Review budget", "budget"),
			map[string]any{"estimatedTotal": summary.EstimatedTotal, "overBudgetBy": *summary.OverBudgetBy, "currency": summary.Currency},
		)
	}
	totalCount := summary.EstimatedItemCount + summary.MissingEstimateCount
	if summary.MissingEstimateCount > 0 {
		severity := SeverityWarning
		if totalCount > 0 && float64(summary.MissingEstimateCount)/float64(totalCount) >= 0.35 {
			severity = SeverityHigh
		}
		b.issue(
			"missing_cost_estimates",
			CategoryBudget,
			severity,
			"Some costs are missing estimates",
			fmt.Sprintf("%d cost-bearing item(s) are missing estimates.", summary.MissingEstimateCount),
			"The budget total may be too low.",
			"Add missing cost estimates or run budget optimization.",
			action(b.tripID, "open_budget", "Add cost estimates", "budget"),
			map[string]any{"missingEstimateCount": summary.MissingEstimateCount},
		)
	}
	if totalCount > 0 && summary.EstimatedItemCount > 0 {
		confidence := float64(summary.EstimatedItemCount) / float64(totalCount)
		if confidence < 0.7 {
			b.issue(
				"budget_low_confidence",
				CategoryBudget,
				SeverityWarning,
				"Budget confidence is low",
				fmt.Sprintf("Only %.0f%% of cost-bearing items have estimates.", confidence*100),
				"The trip may cost more than shown.",
				"Fill in missing accommodation, transport, and activity costs.",
				action(b.tripID, "open_budget", "Improve budget", "budget"),
				map[string]any{"confidence": round2(confidence)},
			)
		}
	}
	if summary.UnconvertedItemCount > 0 || len(summary.ConversionWarnings) > 0 {
		b.issue(
			"conversion_unavailable",
			CategoryBudget,
			SeverityWarning,
			"Some currencies could not be converted",
			"Budget summary has conversion warnings.",
			"Totals may exclude some foreign-currency costs.",
			"Review the budget summary and exchange-rate configuration.",
			action(b.tripID, "open_budget", "Review conversions", "budget"),
			map[string]any{"unconvertedItemCount": summary.UnconvertedItemCount},
		)
	}
	actual := activeExpenseTotal(snapshot.Expenses, trip.BudgetCurrency)
	if trip.BudgetAmount != nil && actual > *trip.BudgetAmount {
		b.issue(
			"actual_budget_exceeded",
			CategoryBudget,
			SeverityHigh,
			"Actual spending exceeds budget",
			fmt.Sprintf("Recorded expenses total %.2f %s against a %.2f %s budget.", actual, trip.BudgetCurrency, *trip.BudgetAmount, trip.BudgetCurrency),
			"Actual spending has already passed the planned budget.",
			"Review expenses and update the budget or spending plan.",
			action(b.tripID, "open_expenses", "Review expenses", "expenses"),
			map[string]any{"actualTotal": actual, "budget": *trip.BudgetAmount, "currency": trip.BudgetCurrency},
		)
	}
	if summary.EstimatedTotal > 0 && actual > summary.EstimatedTotal*1.25 {
		b.issue(
			"planned_actual_spend_gap",
			CategoryBudget,
			SeverityWarning,
			"Actual spending is far above estimates",
			fmt.Sprintf("Actual expenses are %.0f%% of estimated planned costs.", (actual/summary.EstimatedTotal)*100),
			"The original plan may understate real costs.",
			"Update estimates or review spending categories.",
			action(b.tripID, "open_expenses", "Review spending", "expenses"),
			map[string]any{"actualTotal": actual, "estimatedTotal": summary.EstimatedTotal},
		)
	}
	b.evaluateBudgetConfidence(snapshot)
	_ = now
}

func (b *issueBuilder) evaluateBudgetConfidence(snapshot Snapshot) {
	confidence := snapshot.BudgetConfidence
	if confidence == nil {
		return
	}
	if confidence.Level == budgetconfidence.LevelVeryLow || confidence.Level == budgetconfidence.LevelLow {
		severity := SeverityWarning
		if confidence.Level == budgetconfidence.LevelVeryLow {
			severity = SeverityHigh
		}
		b.issue(
			"budget_confidence_low",
			CategoryBudget,
			severity,
			"Budget confidence is low",
			confidence.Summary,
			"The budget may be missing major costs or relying on low-quality estimates.",
			"Open Budget Confidence and confirm the largest uncertain costs.",
			action(b.tripID, "open_budget_confidence", "Open Budget Confidence", "budget"),
			map[string]any{
				"score":     confidence.Score,
				"level":     confidence.Level,
				"riskLevel": confidence.RiskLevel,
			},
		)
	}
	for _, issue := range confidence.Issues {
		if issue.Severity != budgetconfidence.SeverityCritical && issue.Severity != budgetconfidence.SeverityHigh {
			continue
		}
		if budgetConfidenceIssueAlreadyCovered(issue.ID) {
			continue
		}
		severity := SeverityHigh
		if issue.Severity == budgetconfidence.SeverityCritical {
			severity = SeverityCritical
		}
		category := CategoryBudget
		if issue.Category == budgetconfidence.IssueCategoryTransport {
			category = CategoryTransport
		} else if issue.Category == budgetconfidence.IssueCategoryAccommodation {
			category = CategoryAccommodation
		} else if issue.Category == budgetconfidence.IssueCategoryActualSpend {
			category = CategoryExpenses
		}
		b.issue(
			"budget_confidence:"+issue.ID,
			category,
			severity,
			issue.Title,
			issue.Description,
			"Budget reliability is affected by this issue.",
			issue.Recommendation,
			action(b.tripID, "open_budget_confidence", "Open Budget Confidence", "budget"),
			map[string]any{"budgetConfidenceIssueId": issue.ID},
		)
	}
}

func budgetConfidenceIssueAlreadyCovered(id string) bool {
	if strings.HasPrefix(id, "planned_actual_gap:") {
		return true
	}
	switch id {
	case "budget_exceeded_estimated",
		"budget_exceeded_actual",
		"missing_transport_prices",
		"missing_activity_prices",
		"missing_food_budget",
		"currency_conversion_unavailable":
		return true
	default:
		return false
	}
}

func (b *issueBuilder) evaluateAvailability(snapshot Snapshot, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	accepted := acceptedCollaborators(snapshot.Collaborators)
	if len(accepted) == 0 {
		return
	}
	responses := map[uuid.UUID]struct{}{}
	for _, response := range snapshot.AvailabilityResponses {
		responses[response.UserID] = struct{}{}
	}
	missing := 0
	for _, collaborator := range accepted {
		if _, ok := responses[collaborator.UserID]; !ok {
			missing++
		}
	}
	if missing > 0 && trip.StartDate == nil {
		severity := SeverityWarning
		if missing == len(accepted) {
			severity = SeverityHigh
		}
		b.issue(
			"collaborator_availability_missing",
			CategoryAvailability,
			severity,
			"Collaborator availability is missing",
			fmt.Sprintf("%d accepted collaborator(s) have not submitted availability.", missing),
			"Date coordination may not reflect the whole group.",
			"Request availability from missing collaborators.",
			action(b.tripID, "request_availability", "Request availability", "dates"),
			map[string]any{"missingCount": missing},
		)
	}
	if trip.StartDate != nil {
		for _, response := range snapshot.AvailabilityResponses {
			if selectedDatesConflict(trip, response) {
				b.issue(
					"selected_dates_have_conflicts:"+response.UserID.String(),
					CategoryAvailability,
					SeverityWarning,
					"Selected dates conflict with collaborator availability",
					"At least one collaborator marked part of the selected trip dates as unavailable.",
					"The group may not be able to travel on the selected dates.",
					"Review the date coordination panel.",
					action(b.tripID, "open_availability", "Review dates", "dates"),
					map[string]any{"userId": response.UserID.String()},
				)
			}
		}
	}
	_ = now
}

func (b *issueBuilder) evaluateCollaboration(snapshot Snapshot, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	pendingInvites := 0
	for _, collaborator := range snapshot.Collaborators {
		if collaborator.Status == entity.CollaboratorStatusPending {
			pendingInvites++
		}
	}
	if pendingInvites > 0 {
		b.issue(
			"collaborator_not_ready:pending_invites",
			CategoryCollaboration,
			SeverityInfo,
			"Some collaborators have not accepted",
			fmt.Sprintf("%d collaborator invitation(s) are still pending.", pendingInvites),
			"The group plan may not include every intended traveler yet.",
			"Review sharing and collaborator access.",
			action(b.tripID, "open_comments", "Review collaborators", "sharing"),
			map[string]any{"pendingInviteCount": pendingInvites},
		)
	}
	openPolls := 0
	for _, poll := range snapshot.Polls {
		if poll.Status == entity.PollStatusOpen {
			openPolls++
		}
	}
	if openPolls > 0 && tripStartsWithin(trip, now, soonWindowDays) {
		b.issue(
			"pending_group_decisions",
			CategoryCollaboration,
			SeverityWarning,
			"Group decisions are still open",
			fmt.Sprintf("%d poll(s) are still open near departure.", openPolls),
			"Route, date, or activity decisions may remain unresolved.",
			"Close or resolve pending group polls.",
			action(b.tripID, "open_comments", "Review decisions", "decisions"),
			map[string]any{"openPollCount": openPolls},
		)
	}
	assignedChecklist := 0
	if snapshot.Checklist != nil {
		for _, item := range snapshot.Checklist.Items {
			if item.AssignedToUserID != nil && !item.Checked && item.DeletedAt == nil {
				assignedChecklist++
			}
		}
	}
	assignedReminders := 0
	for _, reminder := range snapshot.Reminders {
		if reminder.AssignedToUserID != nil && reminder.Status == entity.ReminderStatusPending && reminder.DeletedAt == nil {
			assignedReminders++
		}
	}
	if assignedChecklist+assignedReminders > 0 && tripStartsWithin(trip, now, soonWindowDays) {
		b.issue(
			"assigned_tasks_incomplete",
			CategoryCollaboration,
			SeverityWarning,
			"Assigned group tasks are incomplete",
			fmt.Sprintf("%d assigned checklist/reminder task(s) remain incomplete.", assignedChecklist+assignedReminders),
			"Preparation may depend on another collaborator finishing a task.",
			"Review assigned checklist items and reminders.",
			action(b.tripID, "open_checklist", "Review tasks", "checklist"),
			map[string]any{"assignedChecklistCount": assignedChecklist, "assignedReminderCount": assignedReminders},
		)
	}
}

func (b *issueBuilder) evaluateChecklist(snapshot Snapshot, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	if snapshot.Checklist == nil {
		if tripStartsWithin(trip, now, soonWindowDays) {
			b.issue(
				"checklist_missing",
				CategoryChecklist,
				SeverityWarning,
				"Checklist is missing",
				"This trip starts soon and has no active checklist.",
				"Packing, document, and booking tasks may be missed.",
				"Generate a trip checklist.",
				action(b.tripID, "generate_checklist", "Generate checklist", "checklist"),
				nil,
			)
		}
		return
	}
	if snapshot.Checklist.GeneratedFromItineraryRevision != nil &&
		*snapshot.Checklist.GeneratedFromItineraryRevision < trip.ItineraryRevision {
		b.issue(
			"checklist_stale",
			CategoryChecklist,
			SeverityWarning,
			"Checklist may be stale",
			"The active checklist was generated before the latest itinerary revision.",
			"New activities, routes, or bookings may not be reflected.",
			"Regenerate or review the checklist.",
			action(b.tripID, "generate_checklist", "Refresh checklist", "checklist"),
			map[string]any{"generatedFromItineraryRevision": *snapshot.Checklist.GeneratedFromItineraryRevision, "itineraryRevision": trip.ItineraryRevision},
		)
	}
	for _, item := range snapshot.Checklist.Items {
		if item.DeletedAt != nil || item.Checked {
			continue
		}
		if item.Priority == entity.ChecklistPriorityHigh || item.Priority == entity.ChecklistPriorityCritical {
			severity := SeverityWarning
			if item.Priority == entity.ChecklistPriorityCritical || tripStartsWithin(trip, now, 3) {
				severity = SeverityHigh
			}
			b.issue(
				"high_priority_checklist_incomplete:"+item.ID.String(),
				CategoryChecklist,
				severity,
				"High-priority checklist item is incomplete",
				fmt.Sprintf("%q is not checked.", item.Title),
				"Important preparation work may be unfinished.",
				"Complete or update the checklist item.",
				action(b.tripID, "open_checklist", "Open checklist", "checklist"),
				map[string]any{"checklistItemId": item.ID.String(), "priority": string(item.Priority)},
			)
		}
		if item.DueDate != nil && dateOnly(*item.DueDate).Before(dateOnly(now)) {
			severity := SeverityWarning
			if item.Priority == entity.ChecklistPriorityHigh || item.Priority == entity.ChecklistPriorityCritical {
				severity = SeverityHigh
			}
			b.issue(
				"checklist_item_overdue:"+item.ID.String(),
				CategoryChecklist,
				severity,
				"Checklist item is overdue",
				fmt.Sprintf("%q was due before today.", item.Title),
				"Overdue preparation can delay bookings or departure tasks.",
				"Complete or reschedule the checklist item.",
				action(b.tripID, "open_checklist", "Review overdue item", "checklist"),
				map[string]any{"checklistItemId": item.ID.String()},
			)
		}
	}
}

func (b *issueBuilder) evaluateReminders(snapshot Snapshot, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	activeCount := 0
	for _, reminder := range snapshot.Reminders {
		if reminder.DeletedAt == nil && reminder.Status != entity.ReminderStatusCancelled {
			activeCount++
		}
	}
	if activeCount == 0 && tripStartsWithin(trip, now, soonWindowDays) {
		b.issue(
			"reminders_missing",
			CategoryReminders,
			SeverityInfo,
			"Reminders are missing",
			"This trip starts soon and has no active reminders.",
			"Important preparation tasks may not notify anyone.",
			"Generate reminders from the trip checklist.",
			action(b.tripID, "generate_reminders", "Generate reminders", "reminders"),
			nil,
		)
	}
	for _, reminder := range snapshot.Reminders {
		if reminder.DeletedAt != nil || reminder.Status != entity.ReminderStatusPending {
			continue
		}
		if dateOnly(reminder.TriggerDate).Before(dateOnly(now)) {
			severity := SeverityWarning
			if reminder.Priority == entity.ReminderPriorityHigh || reminder.Priority == entity.ReminderPriorityCritical {
				severity = SeverityHigh
			}
			b.issue(
				"reminders_overdue:"+reminder.ID.String(),
				CategoryReminders,
				severity,
				"Reminder is overdue",
				fmt.Sprintf("%q was due before today.", reminder.Title),
				"The reminder did not get completed on time.",
				"Complete, reopen, or reschedule the reminder.",
				action(b.tripID, "open_reminders", "Open reminders", "reminders"),
				map[string]any{"reminderId": reminder.ID.String(), "priority": string(reminder.Priority)},
			)
		}
	}
	if latestReminderUpdate(snapshot.Reminders).Before(trip.UpdatedAt.Add(-time.Minute)) && activeCount > 0 {
		b.issue(
			"reminders_stale",
			CategoryReminders,
			SeverityInfo,
			"Reminders may be stale",
			"The trip changed after reminders were last updated.",
			"Reminder timing may not reflect the latest dates, route, or checklist.",
			"Review or regenerate reminders.",
			action(b.tripID, "open_reminders", "Review reminders", "reminders"),
			nil,
		)
	}
}

func (b *issueBuilder) evaluateAccommodation(snapshot Snapshot, now time.Time) {
	trip := snapshot.Trip
	if trip == nil || trip.Days <= 1 {
		return
	}
	if trip.Accommodation == nil {
		severity := SeverityWarning
		if tripStartsWithin(trip, now, soonWindowDays) {
			severity = SeverityHigh
		}
		b.issue(
			"accommodation_missing",
			CategoryAccommodation,
			severity,
			"Accommodation is missing",
			"This overnight trip has no accommodation saved.",
			"Budget and route checks may miss lodging costs and location.",
			"Add accommodation details.",
			action(b.tripID, "open_accommodation", "Add stay", "accommodation"),
			nil,
		)
		return
	}
	accommodation := trip.Accommodation
	if strings.TrimSpace(accommodation.Address) == "" &&
		(accommodation.Place == nil || accommodation.Place.Latitude == nil || accommodation.Place.Longitude == nil) {
		b.issue(
			"accommodation_location_missing",
			CategoryAccommodation,
			SeverityInfo,
			"Accommodation location is incomplete",
			"The accommodation has no address or map coordinates.",
			"Map and walking-distance checks may be less accurate.",
			"Add an address or place match.",
			action(b.tripID, "open_accommodation", "Review stay", "accommodation"),
			nil,
		)
	}
	if trip.StartDate != nil {
		tripStart := dateOnly(*trip.StartDate)
		tripEndExclusive := tripStart.AddDate(0, 0, int(trip.Days))
		checkIn, hasCheckIn := parseDate(accommodation.CheckInDate)
		checkOut, hasCheckOut := parseDate(accommodation.CheckOutDate)
		if !hasCheckIn || !hasCheckOut || checkIn.After(tripStart) || checkOut.Before(tripEndExclusive) {
			b.issue(
				"accommodation_dates_mismatch",
				CategoryAccommodation,
				SeverityHigh,
				"Accommodation dates do not cover the trip",
				"The accommodation check-in/check-out dates do not cover the full trip.",
				"Travelers may have one or more nights without lodging.",
				"Update accommodation dates or add the missing stay.",
				action(b.tripID, "open_accommodation", "Fix stay dates", "accommodation"),
				map[string]any{"checkInDate": accommodation.CheckInDate, "checkOutDate": accommodation.CheckOutDate},
			)
		}
	}
}

func (b *issueBuilder) evaluateExpenses(snapshot Snapshot, cfg Config, now time.Time) {
	trip := snapshot.Trip
	if trip == nil {
		return
	}
	receiptByExpense := map[uuid.UUID]int{}
	for _, signal := range snapshot.ExpenseReceiptSignals {
		receiptByExpense[signal.ExpenseID] = signal.ReceiptCount
	}
	for _, expense := range snapshot.Expenses {
		if expense.Status != entity.ExpenseStatusActive {
			continue
		}
		if expense.Amount >= cfg.LargeExpenseReceiptThreshold && receiptByExpense[expense.ID] == 0 {
			b.issue(
				"missing_receipts_for_large_expenses:"+expense.ID.String(),
				CategoryExpenses,
				SeverityInfo,
				"Large expense has no receipt",
				fmt.Sprintf("%q is %.2f %s and has no receipt attached.", expense.Title, expense.Amount, expense.Currency),
				"Shared expense review may be harder later.",
				"Attach a receipt if one is available.",
				action(b.tripID, "open_expenses", "Open expenses", "expenses"),
				map[string]any{"expenseId": expense.ID.String(), "amount": expense.Amount, "currency": expense.Currency},
			)
		}
	}
	afterTrip := tripEnded(trip, now)
	pendingSettlements := 0
	for _, settlement := range snapshot.Settlements {
		if settlement.Status == entity.SettlementStatusPending {
			pendingSettlements++
		}
	}
	if pendingSettlements > 0 {
		severity := SeverityInfo
		issueID := "unsettled_expenses"
		title := "Unsettled expenses"
		if afterTrip {
			severity = SeverityWarning
			issueID = "settlements_pending"
			title = "Settlements are pending after the trip"
		}
		b.issue(
			issueID,
			CategoryExpenses,
			severity,
			title,
			fmt.Sprintf("%d settlement(s) are still pending.", pendingSettlements),
			"Travelers may still owe each other money.",
			"Review and mark settlements paid when complete.",
			action(b.tripID, "open_settlements", "Open settlements", "expenses"),
			map[string]any{"pendingSettlementCount": pendingSettlements},
		)
	}
	for _, signal := range snapshot.ReceiptOCRSignals {
		if signal.Confidence == entity.ReceiptOCRConfidenceLow || len(signal.Warnings) > 0 {
			b.issue(
				"receipt_ocr_low_confidence:"+signal.ReceiptID.String(),
				CategoryExpenses,
				SeverityInfo,
				"Receipt OCR needs review",
				"A receipt extraction has low confidence or warnings.",
				"Expense amounts or merchant details may need correction.",
				"Review the receipt before creating or trusting the expense.",
				action(b.tripID, "open_expenses", "Review receipt", "expenses"),
				map[string]any{"receiptId": signal.ReceiptID.String()},
			)
		}
	}
}

func (b *issueBuilder) evaluatePolicy(snapshot Snapshot) {
	if snapshot.PolicyLoadFailed {
		b.issue(
			"health_subsystem_unavailable:policy",
			CategoryDataQuality,
			SeverityWarning,
			"Could not evaluate workspace policy health",
			"Workspace policy evaluation could not be loaded.",
			"Approval and compliance readiness may be incomplete.",
			"Retry after workspace policy evaluation is available.",
			nil,
			map[string]any{"subsystem": "policy"},
		)
		return
	}
	if snapshot.PolicyEvaluation == nil || snapshot.PolicyEvaluation.Status == workspacepolicies.EvaluationNotApplicable {
		return
	}
	blockingCount := 0
	warningCount := 0
	for _, result := range snapshot.PolicyEvaluation.Results {
		if result.Status == workspacepolicies.ResultPassed {
			continue
		}
		switch result.Severity {
		case workspacepolicies.SeverityBlocking:
			blockingCount++
		case workspacepolicies.SeverityWarning:
			warningCount++
		}
	}
	if blockingCount > 0 {
		b.issue(
			"policy_blocking_violation",
			CategoryPolicy,
			SeverityCritical,
			"Blocking workspace policy violation",
			fmt.Sprintf("%d blocking workspace policy rule(s) are not satisfied.", blockingCount),
			"This trip may not be approvable until policy blockers are fixed.",
			"Open policy evaluation and resolve blocking rules.",
			action(b.tripID, "open_policy", "Open policy", "policy"),
			map[string]any{"blockingPolicyViolation": true, "blockingCount": blockingCount},
		)
	}
	if warningCount > 0 {
		b.issue(
			"policy_warning_violation",
			CategoryPolicy,
			SeverityHigh,
			"Workspace policy warnings",
			fmt.Sprintf("%d workspace policy warning(s) should be reviewed.", warningCount),
			"Policy warnings can increase approval risk.",
			"Review policy evaluation and repair the plan if needed.",
			action(b.tripID, "open_policy", "Review policy", "policy"),
			map[string]any{"warningCount": warningCount},
		)
	}
}

func (b *issueBuilder) evaluateApproval(snapshot Snapshot, now time.Time) {
	trip := snapshot.Trip
	if trip == nil || trip.WorkspaceID == nil {
		return
	}
	if snapshot.ApprovalRiskLoadFailed {
		b.issue(
			"health_subsystem_unavailable:approval_risk",
			CategoryDataQuality,
			SeverityWarning,
			"Could not evaluate approval risk",
			"Approval risk could not be loaded.",
			"Approval readiness may be incomplete.",
			"Retry after approval risk is available.",
			nil,
			map[string]any{"subsystem": "approval_risk"},
		)
	}
	if snapshot.ApprovalLoadFailed {
		b.issue(
			"health_subsystem_unavailable:approval",
			CategoryDataQuality,
			SeverityWarning,
			"Could not evaluate approval status",
			"Approval status could not be loaded.",
			"Approval readiness may be incomplete.",
			"Retry after approval status is available.",
			nil,
			map[string]any{"subsystem": "approval"},
		)
	}
	if snapshot.ApprovalRisk != nil {
		switch snapshot.ApprovalRisk.Status {
		case approvalrisk.RiskLevelCritical:
			b.issue(
				"approval_high_risk",
				CategoryApproval,
				SeverityHigh,
				"Approval risk is critical",
				"Approval risk scoring found critical risk factors.",
				"The trip may need review or repair before approval.",
				"Open approval risk and address the top reasons.",
				action(b.tripID, "open_approval", "Open approval", "approval"),
				map[string]any{"riskStatus": string(snapshot.ApprovalRisk.Status), "riskScore": snapshot.ApprovalRisk.Score},
			)
		case approvalrisk.RiskLevelHigh:
			b.issue(
				"approval_high_risk",
				CategoryApproval,
				SeverityWarning,
				"Approval risk is high",
				"Approval risk scoring found high-risk factors.",
				"The trip may need review before approval.",
				"Open approval risk and address the top reasons.",
				action(b.tripID, "open_approval", "Open approval", "approval"),
				map[string]any{"riskStatus": string(snapshot.ApprovalRisk.Status), "riskScore": snapshot.ApprovalRisk.Score},
			)
		}
	}
	if snapshot.Approval == nil {
		return
	}
	status := approvals.Status(snapshot.Approval.Status)
	if status == approvals.StatusPendingApproval {
		b.issue(
			"approval_pending",
			CategoryApproval,
			SeverityInfo,
			"Approval is pending",
			"This workspace trip is waiting for approval.",
			"Departure readiness depends on the approval decision.",
			"Open approval workflow to review status.",
			action(b.tripID, "open_approval", "Open approval", "approval"),
			map[string]any{"approvalStatus": snapshot.Approval.Status},
		)
	}
	if status == approvals.StatusChangesRequested {
		b.issue(
			"approval_changes_requested",
			CategoryApproval,
			SeverityHigh,
			"Approval changes were requested",
			"This workspace trip needs changes before approval.",
			"Requested changes can block trip readiness.",
			"Open approval workflow and address the requested changes.",
			action(b.tripID, "open_approval", "Review changes", "approval"),
			map[string]any{"approvalStatus": snapshot.Approval.Status},
		)
	}
	if status != approvals.StatusApproved && tripStartsWithin(trip, now, soonWindowDays) {
		severity := SeverityHigh
		if tripStartsWithin(trip, now, 3) {
			severity = SeverityCritical
		}
		b.issue(
			"approval_not_ready",
			CategoryApproval,
			severity,
			"Trip is not approved near departure",
			"This workspace trip is not approved and departure is close.",
			"Workspace travel may be blocked by approval state.",
			"Submit or resolve approval before departure.",
			action(b.tripID, "open_approval", "Resolve approval", "approval"),
			map[string]any{"approvalStatus": snapshot.Approval.Status},
		)
	}
}

func (b *issueBuilder) evaluateDataQuality(snapshot Snapshot) {
	trip := snapshot.Trip
	if trip == nil || trip.Route == nil {
		return
	}
	for index, leg := range trip.Route.Legs {
		if len(leg.Warnings) > 0 || len(leg.ProviderMetadata) > 0 && metadataBool(leg.ProviderMetadata, "stale") {
			b.issue(
				"stale_provider_data:"+legIDOrIndex(leg, index),
				CategoryDataQuality,
				SeverityInfo,
				"Route provider data may be stale",
				fmt.Sprintf("Route leg %s has provider warnings or stale metadata.", routeLegName(leg)),
				"Route estimates may need refresh before booking.",
				"Refresh route or transport provider data.",
				action(b.tripID, "open_route", "Review route", "route"),
				map[string]any{"routeLegId": legIDOrIndex(leg, index)},
			)
		}
	}
}

func computedFrom(snapshot Snapshot) ComputedFrom {
	out := ComputedFrom{}
	if snapshot.Trip != nil {
		out.ItineraryRevision = snapshot.Trip.ItineraryRevision
		if snapshot.Trip.Route != nil {
			updated := snapshot.Trip.UpdatedAt
			out.RouteUpdatedAt = &updated
		}
		if snapshot.Trip.BudgetAmount != nil {
			updated := snapshot.Trip.UpdatedAt
			out.BudgetUpdatedAt = &updated
		}
	}
	if snapshot.Checklist != nil {
		updated := snapshot.Checklist.UpdatedAt
		out.ChecklistUpdatedAt = &updated
	}
	if latest := latestReminderUpdate(snapshot.Reminders); !latest.IsZero() {
		out.RemindersUpdatedAt = &latest
	}
	return out
}

func sortIssues(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		left, right := issues[i], issues[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) > severityRank(right.Severity)
		}
		if categorySortRank(left.Category) != categorySortRank(right.Category) {
			return categorySortRank(left.Category) < categorySortRank(right.Category)
		}
		return left.ID < right.ID
	})
}

func filterOpenIssues(issues []Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.Status == "" || issue.Status == StatusOpen {
			out = append(out, issue)
		}
	}
	return out
}

func action(tripID uuid.UUID, actionType, label, section string) *Action {
	return &Action{
		Type:  actionType,
		Label: label,
		Href:  tripHref(tripID, section),
	}
}

func tripHref(tripID uuid.UUID, section string) string {
	if section == "" {
		section = "health"
	}
	return fmt.Sprintf("/trips/%s#%s", tripID.String(), section)
}

func (b *issueBuilder) tripHref(section string) string {
	return tripHref(b.tripID, section)
}

func routeStopIDs(route *aggregate.TripRoute) map[string]struct{} {
	if route == nil {
		return nil
	}
	out := make(map[string]struct{}, len(route.Stops)+1)
	out["origin"] = struct{}{}
	for _, stop := range route.Stops {
		if stop.ID != "" {
			out[stop.ID] = struct{}{}
		}
	}
	return out
}

func legIDOrIndex(leg aggregate.RouteLeg, index int) string {
	if strings.TrimSpace(leg.ID) != "" {
		return strings.TrimSpace(leg.ID)
	}
	return fmt.Sprintf("leg_%d", index+1)
}

func stopIDOrIndex(stop aggregate.RouteStop, index int) string {
	if strings.TrimSpace(stop.ID) != "" {
		return strings.TrimSpace(stop.ID)
	}
	return fmt.Sprintf("stop_%d", index+1)
}

func routeStopName(stop aggregate.RouteStop) string {
	for _, value := range []string{stop.Destination, stop.City, stop.Country, stop.ID} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Route stop"
}

func routeLegName(leg aggregate.RouteLeg) string {
	from := strings.TrimSpace(leg.FromName)
	to := strings.TrimSpace(leg.ToName)
	if from == "" {
		from = strings.TrimSpace(leg.FromStopID)
	}
	if to == "" {
		to = strings.TrimSpace(leg.ToStopID)
	}
	if from != "" && to != "" {
		return from + " to " + to
	}
	if leg.ID != "" {
		return leg.ID
	}
	return "route leg"
}

type transportInterval struct {
	start     int
	end       int
	date      string
	dayNumber int
}

func selectedTransportIntervals(route *aggregate.TripRoute) map[string]transportInterval {
	if route == nil {
		return nil
	}
	out := map[string]transportInterval{}
	for index, leg := range route.Legs {
		option := leg.SelectedTransportOption
		if option == nil || option.DepartureTime == "" {
			continue
		}
		start, ok := parseClockMinutes(option.DepartureTime)
		if !ok {
			continue
		}
		end := start + option.DurationMinutes
		if option.ArrivalTime != "" {
			if parsed, ok := parseClockMinutes(option.ArrivalTime); ok {
				end = parsed
			}
		}
		if end < start {
			end += 24 * 60
		}
		out[legIDOrIndex(leg, index)] = transportInterval{
			start: start,
			end:   end,
			date:  firstNonEmpty(option.DepartureDate, leg.DepartureDate),
		}
	}
	return out
}

func itemDateFromDayNumber(dayNumber int) string {
	_ = dayNumber
	return ""
}

func parseClockMinutes(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func parseDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func validDate(value string) bool {
	_, ok := parseDate(value)
	return ok
}

func dateOnly(value time.Time) time.Time {
	y, m, d := value.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func tripStartsWithin(trip *entity.Trip, now time.Time, days int) bool {
	if trip == nil || trip.StartDate == nil {
		return false
	}
	start := dateOnly(*trip.StartDate)
	today := dateOnly(now)
	return !start.Before(today) && !start.After(today.AddDate(0, 0, days))
}

func tripEnded(trip *entity.Trip, now time.Time) bool {
	if trip == nil || trip.StartDate == nil || trip.Days <= 0 {
		return false
	}
	end := dateOnly(*trip.StartDate).AddDate(0, 0, int(trip.Days))
	return dateOnly(now).After(end) || dateOnly(now).Equal(end)
}

func rangesOverlap(startA, endA, startB, endB int) bool {
	return startA < endB && startB < endA
}

func dayWalkingKm(day aggregate.ItineraryDay) float64 {
	total := 0.0
	for _, item := range day.Items {
		if item.WalkingDistanceKm != nil && *item.WalkingDistanceKm > 0 {
			total += *item.WalkingDistanceKm
		}
	}
	return total
}

func isTransportLike(item aggregate.ItineraryItem) bool {
	return normalizeToken(item.Type) == "transport" ||
		normalizeToken(item.Type) == "transfer" ||
		item.Transfer != nil ||
		normalizeToken(item.Category) == "transport"
}

func activeExpenseTotal(expenses []entity.TripExpense, currency string) float64 {
	total := 0.0
	currency = strings.ToUpper(strings.TrimSpace(currency))
	for _, expense := range expenses {
		if expense.Status != entity.ExpenseStatusActive {
			continue
		}
		if currency == "" || strings.EqualFold(expense.Currency, currency) {
			total += expense.Amount
		}
	}
	return round2(total)
}

func selectedDatesConflict(trip *entity.Trip, response entity.TripAvailabilityResponse) bool {
	if trip.StartDate == nil || trip.Days <= 0 {
		return false
	}
	start := dateOnly(*trip.StartDate)
	end := start.AddDate(0, 0, int(trip.Days)-1)
	for _, unavailable := range response.UnavailableRanges {
		rangeStart, okStart := parseDate(unavailable.StartDate)
		rangeEnd, okEnd := parseDate(unavailable.EndDate)
		if !okStart || !okEnd {
			continue
		}
		if !end.Before(rangeStart) && !rangeEnd.Before(start) {
			return true
		}
	}
	return false
}

func acceptedCollaborators(collaborators []entity.TripCollaborator) []entity.TripCollaborator {
	out := make([]entity.TripCollaborator, 0, len(collaborators))
	for _, collaborator := range collaborators {
		if collaborator.Status == entity.CollaboratorStatusAccepted {
			out = append(out, collaborator)
		}
	}
	return out
}

func latestReminderUpdate(reminders []entity.TripReminder) time.Time {
	var latest time.Time
	for _, reminder := range reminders {
		if reminder.UpdatedAt.After(latest) {
			latest = reminder.UpdatedAt
		}
	}
	return latest
}

func isApprovalActive(approval *entity.TripApprovalFields) bool {
	if approval == nil {
		return false
	}
	status := approvals.Status(approval.Status)
	return status == approvals.StatusPendingApproval || status == approvals.StatusApproved
}

func metadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	switch value := metadata[key].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return false
	}
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := minutes / 60
	remainder := minutes % 60
	if remainder == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d hours %d minutes", hours, remainder)
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
