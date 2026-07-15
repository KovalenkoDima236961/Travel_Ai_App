package aivalidation

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

type Validator struct {
	cfg Config
}

func NewValidator(cfg Config) *Validator {
	return &Validator{cfg: NormalizeConfig(cfg)}
}

func (v *Validator) Validate(ctx context.Context, input ValidationInput) (ValidationResult, error) {
	started := time.Now()
	if !v.cfg.Enabled {
		result := ValidationResult{
			Valid:         true,
			SaveAllowed:   true,
			Issues:        []ValidationIssue{},
			Warnings:      []string{},
			QualityStatus: StatusNotValidated,
		}
		recordValidation(input.GenerationType, result.QualityStatus, result.Issues, time.Since(started))
		return result, nil
	}

	builder := issueBuilder{}
	builder.add(v.validateSchema(input)...)
	builder.add(v.validateDayCount(input)...)
	builder.add(v.validateTimeOrdering(input)...)
	builder.add(v.validateRoute(input)...)
	builder.add(v.validateTransport(input)...)
	builder.add(v.validateOpeningHours(input)...)
	builder.add(v.validateWeather(input)...)
	builder.add(v.validateBudget(input)...)
	builder.add(v.validatePolicy(input)...)
	builder.add(v.validateGroupPreferences(input)...)
	builder.add(v.validateAccommodation(input)...)
	builder.add(v.validatePlaceData(input)...)

	issues := builder.issues()
	blocking := filterIssues(issues, func(issue ValidationIssue) bool {
		return isSaveBlockingIssue(issue, v.cfg)
	})
	repairable := filterIssues(issues, func(issue ValidationIssue) bool {
		return issue.Fixability == FixableByAI
	})
	warnings := warningsFromIssues(issues)
	status := qualityStatusForValidation(issues, len(blocking) == 0)
	result := ValidationResult{
		Valid:            len(blocking) == 0,
		SaveAllowed:      len(blocking) == 0,
		Issues:           issues,
		BlockingIssues:   blocking,
		RepairableIssues: repairable,
		Warnings:         warnings,
		QualityStatus:    status,
	}
	recordValidation(input.GenerationType, result.QualityStatus, result.Issues, time.Since(started))
	return result, nil
}

func (v *Validator) validateSchema(input ValidationInput) []ValidationIssue {
	itinerary := input.Itinerary
	issues := make([]ValidationIssue, 0)
	if len(itinerary.Days) == 0 {
		issues = append(issues, issue("schema_missing_required_field:days", CategorySchema, SeverityCritical,
			"Missing itinerary days", "The itinerary must include a non-empty days list.", NotFixable))
		return issues
	}
	if strings.TrimSpace(itinerary.Destination) == "" && strings.TrimSpace(input.Trip.Destination) == "" {
		issues = append(issues, issue("schema_missing_required_field:destination", CategorySchema, SeverityCritical,
			"Missing destination", "The itinerary must include a destination.", FixableByAI))
	}
	if itinerary.Travelers < 0 {
		issues = append(issues, issue("schema_invalid_field:travelers", CategorySchema, SeverityCritical,
			"Invalid traveler count", "The traveler count cannot be negative.", FixableByAI))
	}
	for dayIndex, day := range itinerary.Days {
		dayNumber := day.Day
		if dayNumber <= 0 {
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("schema_invalid_day_number:%d", dayIndex),
				CategorySchema,
				SeverityCritical,
				"Invalid day number",
				"Day numbers must be positive integers.",
				FixableByAI,
				&dayNumber,
				nil,
			))
		}
		if strings.TrimSpace(day.Title) == "" {
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("schema_missing_required_field:day_%d:title", dayNumber),
				CategorySchema,
				SeverityCritical,
				"Missing day title",
				"Each itinerary day must include a title.",
				FixableByAI,
				&dayNumber,
				nil,
			))
		}
		if day.Items == nil {
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("schema_missing_required_field:day_%d:items", dayNumber),
				CategorySchema,
				SeverityCritical,
				"Missing itinerary items",
				"Each day must include an items list.",
				FixableByAI,
				&dayNumber,
				nil,
			))
			continue
		}
		for itemIndex, item := range day.Items {
			index := itemIndex
			if strings.TrimSpace(item.Name) == "" {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("schema_missing_required_field:day_%d:item_%d:name", dayNumber, itemIndex),
					CategorySchema,
					SeverityCritical,
					"Missing item name",
					"Each itinerary item must include a name.",
					FixableByAI,
					&dayNumber,
					&index,
				))
			}
			if strings.TrimSpace(item.Type) == "" {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("schema_missing_required_field:day_%d:item_%d:type", dayNumber, itemIndex),
					CategorySchema,
					SeverityCritical,
					"Missing item type",
					"Each itinerary item must include a type.",
					FixableByAI,
					&dayNumber,
					&index,
				))
			}
			if strings.TrimSpace(item.Time) != "" && !validHHMM(item.Time) {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("schema_invalid_time:day_%d:item_%d:time", dayNumber, itemIndex),
					CategorySchema,
					SeverityCritical,
					"Invalid start time",
					"Item start times must use HH:MM 24-hour format.",
					FixableByAI,
					&dayNumber,
					&index,
				))
			}
			if strings.TrimSpace(item.EndTime) != "" && !validHHMM(item.EndTime) {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("schema_invalid_time:day_%d:item_%d:end_time", dayNumber, itemIndex),
					CategorySchema,
					SeverityCritical,
					"Invalid end time",
					"Item end times must use HH:MM 24-hour format.",
					FixableByAI,
					&dayNumber,
					&index,
				))
			}
			if item.EstimatedCost != nil {
				if item.EstimatedCost.Amount != nil && *item.EstimatedCost.Amount < 0 {
					issues = append(issues, issueWithLocation(
						fmt.Sprintf("schema_invalid_cost:day_%d:item_%d", dayNumber, itemIndex),
						CategorySchema,
						SeverityCritical,
						"Invalid estimated cost",
						"Estimated costs cannot be negative.",
						FixableByAI,
						&dayNumber,
						&index,
					))
				}
				if c := strings.TrimSpace(item.EstimatedCost.Currency); c != "" && !validCurrency(c) {
					issues = append(issues, issueWithLocation(
						fmt.Sprintf("schema_invalid_cost_currency:day_%d:item_%d", dayNumber, itemIndex),
						CategorySchema,
						SeverityCritical,
						"Invalid estimated cost currency",
						"Estimated cost currency must be a three-letter code.",
						FixableByAI,
						&dayNumber,
						&index,
					))
				}
			}
			if item.Place != nil {
				if item.Place.Latitude != nil && (*item.Place.Latitude < -90 || *item.Place.Latitude > 90) {
					issues = append(issues, issueWithLocation(
						fmt.Sprintf("schema_invalid_place:day_%d:item_%d:latitude", dayNumber, itemIndex),
						CategorySchema,
						SeverityCritical,
						"Invalid place latitude",
						"Place latitude must be between -90 and 90.",
						FixableByAI,
						&dayNumber,
						&index,
					))
				}
				if item.Place.Longitude != nil && (*item.Place.Longitude < -180 || *item.Place.Longitude > 180) {
					issues = append(issues, issueWithLocation(
						fmt.Sprintf("schema_invalid_place:day_%d:item_%d:longitude", dayNumber, itemIndex),
						CategorySchema,
						SeverityCritical,
						"Invalid place longitude",
						"Place longitude must be between -180 and 180.",
						FixableByAI,
						&dayNumber,
						&index,
					))
				}
			}
		}
	}
	return issues
}

func (v *Validator) validateDayCount(input ValidationInput) []ValidationIssue {
	itinerary := input.Itinerary
	expected := input.Context.ExpectedDayCount
	if expected <= 0 {
		expected = int(input.Trip.Days)
	}
	issues := make([]ValidationIssue, 0)
	if expected > 0 && len(itinerary.Days) != expected {
		issues = append(issues, issue("itinerary_day_count_mismatch", CategoryItinerary, SeverityCritical,
			"Day count does not match trip duration",
			fmt.Sprintf("The trip is %d day(s), but the itinerary has %d day(s).", expected, len(itinerary.Days)),
			FixableByAI))
	}
	seen := make(map[int]int, len(itinerary.Days))
	for index, day := range itinerary.Days {
		if day.Day <= 0 {
			continue
		}
		if previous, ok := seen[day.Day]; ok {
			dayNumber := day.Day
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("itinerary_duplicate_day:%d", day.Day),
				CategoryItinerary,
				SeverityCritical,
				"Duplicate itinerary day",
				fmt.Sprintf("Day %d appears more than once at positions %d and %d.", day.Day, previous+1, index+1),
				FixableByAI,
				&dayNumber,
				nil,
			))
		}
		seen[day.Day] = index
	}
	maxDay := len(itinerary.Days)
	if expected > maxDay {
		maxDay = expected
	}
	for dayNumber := 1; dayNumber <= maxDay; dayNumber++ {
		if _, ok := seen[dayNumber]; !ok {
			n := dayNumber
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("itinerary_missing_day:%d", dayNumber),
				CategoryItinerary,
				SeverityCritical,
				"Missing itinerary day",
				fmt.Sprintf("Day %d is missing from the itinerary.", dayNumber),
				FixableByAI,
				&n,
				nil,
			))
		}
	}
	return issues
}

func (v *Validator) validateTimeOrdering(input ValidationInput) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	for _, day := range input.Itinerary.Days {
		spans := make([]itemSpan, 0, len(day.Items))
		for itemIndex, item := range day.Items {
			start, startOK := parseHHMM(item.Time)
			end, endOK := itemEndMinute(item, start)
			index := itemIndex
			dayNumber := day.Day
			if strings.TrimSpace(item.Time) != "" && startOK && endOK && end <= start {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("item_invalid_time_order:day_%d:item_%d", day.Day, itemIndex),
					CategoryTime,
					SeverityCritical,
					"Item end time is before start time",
					fmt.Sprintf("%s ends before it starts.", itemName(item)),
					FixableByAI,
					&dayNumber,
					&index,
				))
				continue
			}
			if startOK && endOK {
				spans = append(spans, itemSpan{dayNumber: day.Day, itemIndex: itemIndex, start: start, end: end, item: item})
			}
		}
		sort.SliceStable(spans, func(i, j int) bool {
			if spans[i].start == spans[j].start {
				return spans[i].end < spans[j].end
			}
			return spans[i].start < spans[j].start
		})
		for i := 1; i < len(spans); i++ {
			prev := spans[i-1]
			current := spans[i]
			if current.start >= prev.end || overlapAllowed(prev.item, current.item) {
				continue
			}
			dayNumber := day.Day
			index := current.itemIndex
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("itinerary_item_overlap:day_%d:item_%d:item_%d", day.Day, prev.itemIndex, current.itemIndex),
				CategoryTime,
				SeverityCritical,
				"Overlapping itinerary items",
				fmt.Sprintf("%s overlaps %s.", itemName(current.item), itemName(prev.item)),
				FixableByAI,
				&dayNumber,
				&index,
			))
		}
	}
	return issues
}

func (v *Validator) validateRoute(input ValidationInput) []ValidationIssue {
	route := input.Trip.Route
	if route == nil || len(route.Stops) == 0 {
		return nil
	}
	issues := make([]ValidationIssue, 0)
	stops := make(map[string]aggregate.RouteStop, len(route.Stops))
	for _, stop := range route.Stops {
		stops[strings.TrimSpace(stop.ID)] = stop
	}
	legsByPair := make(map[string]aggregate.RouteLeg, len(route.Legs))
	for _, leg := range route.Legs {
		legsByPair[leg.FromStopID+"->"+leg.ToStopID] = leg
	}

	for _, day := range input.Itinerary.Days {
		if strings.TrimSpace(day.PrimaryStopID) == "" {
			dayNumber := day.Day
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("day_missing_primary_stop:%d", day.Day),
				CategoryRoute,
				SeverityHigh,
				"Day is missing route stop",
				"Multi-destination itinerary days should reference a route stop.",
				FixableByAI,
				&dayNumber,
				nil,
			))
			continue
		}
		stop, ok := stops[day.PrimaryStopID]
		if !ok {
			dayNumber := day.Day
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("day_stop_not_in_route:%d:%s", day.Day, day.PrimaryStopID),
				CategoryRoute,
				SeverityCritical,
				"Day references unknown route stop",
				"The itinerary references a stop that is not in the selected route.",
				FixableByAI,
				&dayNumber,
				nil,
			))
			continue
		}
		expected := stopName(stop)
		if !samePlaceHint(day.LocationName, expected) && strings.TrimSpace(day.LocationName) != "" {
			dayNumber := day.Day
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("activity_wrong_route_stop:day_%d", day.Day),
				CategoryRoute,
				SeverityHigh,
				"Day location conflicts with route stop",
				fmt.Sprintf("Day %d is assigned to %s but its location is %s.", day.Day, expected, day.LocationName),
				FixableByAI,
				&dayNumber,
				nil,
			))
		}
		for itemIndex, item := range day.Items {
			if isTransportLike(item) || item.Place == nil {
				continue
			}
			placeText := strings.TrimSpace(item.Place.Name + " " + item.Place.Address)
			if placeText == "" || samePlaceHint(placeText, expected) {
				continue
			}
			if stop.City != "" && strings.Contains(strings.ToLower(placeText), strings.ToLower(stop.City)) {
				continue
			}
			dayNumber := day.Day
			index := itemIndex
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("activity_wrong_route_stop:day_%d:item_%d", day.Day, itemIndex),
				CategoryRoute,
				SeverityHigh,
				"Activity may be in the wrong city",
				fmt.Sprintf("%s does not appear to match route stop %s.", itemName(item), expected),
				FixableByAI,
				&dayNumber,
				&index,
			))
		}
	}

	orderedDays := sortedDays(input.Itinerary.Days)
	for i := 1; i < len(orderedDays); i++ {
		prev := orderedDays[i-1]
		current := orderedDays[i]
		if prev.PrimaryStopID == "" || current.PrimaryStopID == "" || prev.PrimaryStopID == current.PrimaryStopID {
			continue
		}
		leg, hasLeg := legsByPair[prev.PrimaryStopID+"->"+current.PrimaryStopID]
		if !hasLeg {
			dayNumber := current.Day
			issues = append(issues, issueWithLocation(
				fmt.Sprintf("impossible_cross_city_schedule:day_%d", current.Day),
				CategoryRoute,
				SeverityCritical,
				"Impossible route jump",
				"The itinerary changes route stops without a matching route leg.",
				FixableByAI,
				&dayNumber,
				nil,
			))
			continue
		}
		if !hasTransferForLeg(prev, current, leg.ID) {
			dayNumber := current.Day
			issues = append(issues, issueWithRouteLeg(
				fmt.Sprintf("missing_transfer_between_stops:%s", leg.ID),
				CategoryRoute,
				SeverityHigh,
				"Missing transfer between route stops",
				fmt.Sprintf("The itinerary moves from %s to %s without a transfer item.", routeEndpointName(leg.FromName, prev.LocationName), routeEndpointName(leg.ToName, current.LocationName)),
				FixableByAI,
				&dayNumber,
				nil,
				leg.ID,
			))
		}
	}
	return issues
}

func (v *Validator) validateTransport(input ValidationInput) []ValidationIssue {
	route := input.Trip.Route
	if route == nil {
		return nil
	}
	issues := make([]ValidationIssue, 0)
	for _, leg := range route.Legs {
		selected := leg.SelectedTransportOption
		if selected == nil {
			continue
		}
		if strings.TrimSpace(selected.Mode) != "" && strings.TrimSpace(leg.Mode) != "" &&
			normalizeToken(selected.Mode) != normalizeToken(leg.Mode) {
			issues = append(issues, issueWithRouteLeg(
				"transport_mode_mismatch:"+leg.ID,
				CategoryTransport,
				SeverityHigh,
				"Selected transport mode does not match route leg",
				"The selected transport option uses a different mode than the route leg.",
				FixableByAI,
				nil,
				nil,
				leg.ID,
			))
		}
		departure, arrival, ok := selectedTransportInterval(*selected)
		if !ok {
			issues = append(issues, issueWithRouteLeg(
				"transport_option_invalid_time:"+leg.ID,
				CategoryTransport,
				SeverityCritical,
				"Selected transport time is invalid",
				"Selected transport must include valid departure and arrival dates and times.",
				FixableByUser,
				nil,
				nil,
				leg.ID,
			))
			continue
		}
		if !arrival.After(departure) {
			issues = append(issues, issueWithRouteLeg(
				"transport_option_invalid_time_order:"+leg.ID,
				CategoryTransport,
				SeverityCritical,
				"Selected transport arrives before it departs",
				"Selected transport arrival must be after departure.",
				FixableByUser,
				nil,
				nil,
				leg.ID,
			))
			continue
		}
		if selected.DurationMinutes > 0 {
			actual := int(arrival.Sub(departure).Minutes())
			if math.Abs(float64(actual-selected.DurationMinutes)) > 45 {
				issues = append(issues, issueWithRouteLeg(
					"transport_duration_mismatch:"+leg.ID,
					CategoryTransport,
					SeverityWarning,
					"Selected transport duration differs from schedule",
					"The selected transport duration does not match its departure and arrival times.",
					FixableByUser,
					nil,
					nil,
					leg.ID,
				))
			}
		}
		matchingTransfer := false
		for _, day := range input.Itinerary.Days {
			for itemIndex, item := range day.Items {
				if item.Transfer != nil && strings.TrimSpace(item.Transfer.LegID) == leg.ID {
					matchingTransfer = true
				}
				itemStart, itemEnd, ok := absoluteItemInterval(day, item)
				if !ok || isTransportLike(item) {
					continue
				}
				if intervalsOverlap(itemStart, itemEnd, departure, arrival) {
					dayNumber := day.Day
					index := itemIndex
					issues = append(issues, issueWithRouteLeg(
						fmt.Sprintf("activity_during_transport:day_%d:item_%d:%s", day.Day, itemIndex, leg.ID),
						CategoryTransport,
						SeverityCritical,
						"Activity overlaps selected transport",
						fmt.Sprintf("%s is scheduled during selected transport.", itemName(item)),
						FixableByAI,
						&dayNumber,
						&index,
						leg.ID,
					))
				}
				if itemStart.Before(arrival) && day.PrimaryStopID == leg.ToStopID && !isTransportLike(item) && sameDate(itemStart, arrival) {
					dayNumber := day.Day
					index := itemIndex
					issues = append(issues, issueWithRouteLeg(
						fmt.Sprintf("activity_before_transport_arrival:day_%d:item_%d:%s", day.Day, itemIndex, leg.ID),
						CategoryTransport,
						SeverityCritical,
						"Activity starts before transport arrival",
						fmt.Sprintf("%s starts before arrival at the destination stop.", itemName(item)),
						FixableByAI,
						&dayNumber,
						&index,
						leg.ID,
					))
				}
			}
		}
		if !matchingTransfer {
			issues = append(issues, issueWithRouteLeg(
				"transfer_item_missing_or_mismatch:"+leg.ID,
				CategoryTransport,
				SeverityHigh,
				"Selected transport is missing from itinerary",
				"The itinerary should include a transfer item for selected transport.",
				FixableByAI,
				nil,
				nil,
				leg.ID,
			))
		}
	}
	return issues
}

func (v *Validator) validateOpeningHours(input ValidationInput) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	for _, day := range input.Itinerary.Days {
		dayDate, hasDate := parseDate(day.Date)
		for itemIndex, item := range day.Items {
			if isTransportLike(item) || item.Place == nil {
				continue
			}
			if len(item.Place.OpeningHours) == 0 {
				if likelyNeedsOpeningHours(item) {
					dayNumber := day.Day
					index := itemIndex
					issues = append(issues, issueWithLocation(
						fmt.Sprintf("opening_hours_unknown:day_%d:item_%d", day.Day, itemIndex),
						CategoryOpeningHours,
						SeverityWarning,
						"Opening hours unknown",
						"The place opening hours could not be verified.",
						FixableByUser,
						&dayNumber,
						&index,
					))
				}
				continue
			}
			start, ok := parseHHMM(item.Time)
			if !ok || !hasDate {
				continue
			}
			weekday := int(dayDate.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			if !withinOpeningHours(item.Place.OpeningHours, weekday, start) {
				dayNumber := day.Day
				index := itemIndex
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("place_likely_closed:day_%d:item_%d", day.Day, itemIndex),
					CategoryOpeningHours,
					SeverityHigh,
					"Place may be closed",
					fmt.Sprintf("%s is scheduled outside known opening hours.", itemName(item)),
					FixableByAI,
					&dayNumber,
					&index,
				))
			}
		}
	}
	return issues
}

func (v *Validator) validateWeather(input ValidationInput) []ValidationIssue {
	if input.WeatherForecast == nil {
		return nil
	}
	byDate := make(map[string]weathercontext.WeatherDay, len(input.WeatherForecast.Days))
	for _, day := range input.WeatherForecast.Days {
		byDate[day.Date] = day
	}
	issues := make([]ValidationIssue, 0)
	for _, day := range input.Itinerary.Days {
		weatherDay, ok := byDate[day.Date]
		if !ok {
			continue
		}
		for itemIndex, item := range day.Items {
			if !isOutdoorItem(item) {
				continue
			}
			severity := SeverityWarning
			title := "Outdoor activity may be affected by weather"
			id := fmt.Sprintf("weather_risk_outdoor_activity:day_%d:item_%d", day.Day, itemIndex)
			if isHikingItem(item) {
				id = fmt.Sprintf("weather_risk_hiking:day_%d:item_%d", day.Day, itemIndex)
				title = "Hiking may be affected by severe weather"
			} else if isCampingItem(item) {
				id = fmt.Sprintf("weather_risk_camping:day_%d:item_%d", day.Day, itemIndex)
				title = "Camping may be affected by severe weather"
			}
			if severeWeather(weatherDay) {
				severity = SeverityHigh
			} else if !weatherRisk(weatherDay) {
				continue
			}
			dayNumber := day.Day
			index := itemIndex
			issues = append(issues, issueWithLocation(
				id,
				CategoryWeather,
				severity,
				title,
				"Weather context suggests this outdoor item should be reviewed. This is not a safety guarantee.",
				FixableByAI,
				&dayNumber,
				&index,
			))
		}
	}
	return issues
}

func (v *Validator) validateBudget(input ValidationInput) []ValidationIssue {
	summary := input.BudgetSummary
	if summary == nil {
		calculated := budget.CalculateBudgetSummary(budget.TripBudget{
			Amount:        input.Trip.BudgetAmount,
			Currency:      input.Trip.BudgetCurrency,
			Days:          int(input.Trip.Days),
			Accommodation: input.Trip.Accommodation,
			Route:         input.Trip.Route,
		}, input.Itinerary)
		summary = &calculated
	}
	issues := make([]ValidationIssue, 0)
	if summary.OverBudgetBy != nil && *summary.OverBudgetBy > 0 {
		severity := SeverityHigh
		fixability := FixableByAI
		if input.PlanningConstraints != nil && input.PlanningConstraints.Budget != nil &&
			strings.EqualFold(input.PlanningConstraints.Budget.Strictness, "strict") {
			severity = SeverityCritical
		}
		issues = append(issues, issue("generated_budget_exceeded", CategoryBudget, severity,
			"Generated itinerary exceeds budget",
			fmt.Sprintf("Estimated total is over budget by %.2f %s.", *summary.OverBudgetBy, summary.Currency),
			fixability))
	}
	if summary.MissingEstimateCount > 0 {
		issues = append(issues, issue("missing_estimated_costs", CategoryBudget, SeverityWarning,
			"Some budget estimates are missing",
			fmt.Sprintf("%d paid item(s) do not have estimated costs.", summary.MissingEstimateCount),
			FixableByAI))
	}
	if input.GenerationType == GenerationTypeBudgetOptimizationDay && summary.OverBudgetBy != nil && *summary.OverBudgetBy > 0 {
		issues = append(issues, issue("budget_optimization_no_savings", CategoryBudget, SeverityHigh,
			"Budget optimization did not reduce enough cost",
			"The budget optimization output still exceeds the available budget.",
			FixableByAI))
	}
	return issues
}

func (v *Validator) validatePolicy(input ValidationInput) []ValidationIssue {
	if input.PolicyEvaluation == nil {
		return nil
	}
	evaluation := input.PolicyEvaluation
	issues := make([]ValidationIssue, 0)
	for _, result := range evaluation.Results {
		if result.Status != workspacepolicies.ResultViolation {
			continue
		}
		severity := SeverityWarning
		fixability := FixableByAI
		id := "policy_warning_violation:" + result.RuleKey
		title := result.Title
		if result.Severity == workspacepolicies.SeverityBlocking {
			severity = SeverityBlocking
			id = "policy_blocking_violation:" + result.RuleKey
		}
		issue := ValidationIssue{
			ID:          id,
			Category:    CategoryPolicy,
			Severity:    severity,
			Title:       title,
			Description: result.Message,
			Fixability:  fixability,
			RuleKey:     result.RuleKey,
		}
		if len(result.AffectedItems) > 0 {
			issue.DayNumber = result.AffectedItems[0].DayNumber
			issue.ItemIndex = result.AffectedItems[0].ItemIndex
		}
		issues = append(issues, issue)
	}
	if evaluation.Status == workspacepolicies.EvaluationBlocking && len(issues) == 0 {
		issues = append(issues, issue("policy_blocking_violation", CategoryPolicy, SeverityBlocking,
			"Generated itinerary violates workspace policy",
			"The policy evaluator reported a blocking violation.",
			FixableByAI))
	}
	return issues
}

func (v *Validator) validateGroupPreferences(input ValidationInput) []ValidationIssue {
	if input.PlanningConstraints == nil || input.PlanningConstraints.GroupPreferences == nil {
		return nil
	}
	prefs := input.PlanningConstraints.GroupPreferences
	issues := make([]ValidationIssue, 0)
	itineraryText := normalizeText(itineraryNames(input.Itinerary))
	for _, must := range prefs.MustHaveItems {
		if strings.TrimSpace(must.Name) == "" || strings.Contains(itineraryText, normalizeText(must.Name)) {
			continue
		}
		dayNumber := must.DayNumber
		index := must.ItemIndex
		issues = append(issues, issueWithLocation(
			"group_must_have_missing:"+normalizeID(must.Name),
			CategoryGroupPreferences,
			SeverityHigh,
			"Must-have group item is missing",
			fmt.Sprintf("%s was marked as important by collaborators.", must.Name),
			FixableByAI,
			&dayNumber,
			&index,
		))
	}
	for _, skip := range prefs.SkipCandidates {
		if strings.TrimSpace(skip.Name) == "" || !strings.Contains(itineraryText, normalizeText(skip.Name)) {
			continue
		}
		dayNumber := skip.DayNumber
		index := skip.ItemIndex
		issues = append(issues, issueWithLocation(
			"group_disliked_item_included:"+normalizeID(skip.Name),
			CategoryGroupPreferences,
			SeverityWarning,
			"Disliked group item is included",
			fmt.Sprintf("%s was marked as a skip candidate by collaborators.", skip.Name),
			FixableByAI,
			&dayNumber,
			&index,
		))
	}
	if input.PlanningConstraints.GroupAvailability != nil && input.PlanningConstraints.GroupAvailability.SelectedDateOption != nil {
		selected := input.PlanningConstraints.GroupAvailability.SelectedDateOption
		if selected.DurationDays > 0 && len(input.Itinerary.Days) != selected.DurationDays {
			issues = append(issues, issue("selected_date_constraint_violated", CategoryGroupPreferences, SeverityCritical,
				"Selected group dates were contradicted",
				"The itinerary day count does not match the selected group date option.",
				FixableByAI))
		}
	}
	return issues
}

func (v *Validator) validateAccommodation(input ValidationInput) []ValidationIssue {
	accommodation := input.Trip.Accommodation
	if accommodation == nil {
		return nil
	}
	issues := make([]ValidationIssue, 0)
	if accommodation.CheckInDate != "" && input.Trip.StartDate != nil {
		start := input.Trip.StartDate.Format("2006-01-02")
		if accommodation.CheckInDate > start {
			issues = append(issues, issue("accommodation_dates_mismatch:check_in", CategoryAccommodation, SeverityWarning,
				"Accommodation check-in starts after trip begins",
				"Early trip activities may need accommodation or luggage context.",
				FixableByUser))
		}
	}
	if accommodation.CheckOutDate != "" && input.Trip.StartDate != nil && input.Trip.Days > 0 {
		end := input.Trip.StartDate.AddDate(0, 0, int(input.Trip.Days)).Format("2006-01-02")
		if accommodation.CheckOutDate < end {
			issues = append(issues, issue("accommodation_dates_mismatch:check_out", CategoryAccommodation, SeverityWarning,
				"Accommodation check-out is before trip ends",
				"Late trip activities may need accommodation or luggage context.",
				FixableByUser))
		}
	}
	firstDayHasContext := false
	lastDayHasContext := false
	for _, day := range input.Itinerary.Days {
		for _, item := range day.Items {
			text := normalizeToken(item.Type + " " + item.Name)
			if strings.Contains(text, "check_in") || strings.Contains(text, "hotel") || strings.Contains(text, "accommodation") {
				if day.Day == 1 {
					firstDayHasContext = true
				}
			}
			if strings.Contains(text, "check_out") || strings.Contains(text, "luggage") {
				if day.Day == len(input.Itinerary.Days) {
					lastDayHasContext = true
				}
			}
		}
	}
	if accommodation.CheckInDate != "" && !firstDayHasContext {
		issues = append(issues, issue("accommodation_context_ignored:check_in", CategoryAccommodation, SeverityInfo,
			"Accommodation check-in is not reflected",
			"The first day may need check-in or luggage timing.",
			NonBlocking))
	}
	if accommodation.CheckOutDate != "" && !lastDayHasContext {
		issues = append(issues, issue("accommodation_context_ignored:check_out", CategoryAccommodation, SeverityInfo,
			"Accommodation check-out is not reflected",
			"The final day may need check-out or luggage timing.",
			NonBlocking))
	}
	return issues
}

func (v *Validator) validatePlaceData(input ValidationInput) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	for _, day := range input.Itinerary.Days {
		for itemIndex, item := range day.Items {
			if isTransportLike(item) || !keyActivity(item) {
				continue
			}
			dayNumber := day.Day
			index := itemIndex
			if item.PlaceEnrichment != nil && item.PlaceEnrichment.Confidence > 0 && item.PlaceEnrichment.Confidence < 0.55 {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("low_confidence_place:day_%d:item_%d", day.Day, itemIndex),
					CategoryPlace,
					SeverityWarning,
					"Low-confidence place match",
					fmt.Sprintf("%s has a low-confidence place match.", itemName(item)),
					FixableByUser,
					&dayNumber,
					&index,
				))
			}
			if item.Place == nil {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("missing_place_for_key_activity:day_%d:item_%d", day.Day, itemIndex),
					CategoryPlace,
					SeverityWarning,
					"Key activity is missing a place",
					fmt.Sprintf("%s does not have attached place metadata.", itemName(item)),
					FixableByUser,
					&dayNumber,
					&index,
				))
				continue
			}
			if item.Place.Latitude == nil || item.Place.Longitude == nil {
				issues = append(issues, issueWithLocation(
					fmt.Sprintf("missing_place_coordinates:day_%d:item_%d", day.Day, itemIndex),
					CategoryPlace,
					SeverityInfo,
					"Place coordinates are missing",
					fmt.Sprintf("%s may not render accurately on the map.", itemName(item)),
					FixableByUser,
					&dayNumber,
					&index,
				))
			}
		}
	}
	return issues
}

type issueBuilder struct {
	byID map[string]ValidationIssue
}

func (b *issueBuilder) add(issues ...ValidationIssue) {
	if b.byID == nil {
		b.byID = make(map[string]ValidationIssue)
	}
	for _, issue := range issues {
		if strings.TrimSpace(issue.ID) == "" {
			continue
		}
		if existing, ok := b.byID[issue.ID]; ok {
			if severityRank(issue.Severity) > severityRank(existing.Severity) {
				b.byID[issue.ID] = issue
			}
			continue
		}
		b.byID[issue.ID] = issue
	}
}

func (b *issueBuilder) issues() []ValidationIssue {
	if len(b.byID) == 0 {
		return []ValidationIssue{}
	}
	out := make([]ValidationIssue, 0, len(b.byID))
	for _, issue := range b.byID {
		out = append(out, issue)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if severityRank(out[i].Severity) == severityRank(out[j].Severity) {
			return out[i].ID < out[j].ID
		}
		return severityRank(out[i].Severity) > severityRank(out[j].Severity)
	})
	return out
}
