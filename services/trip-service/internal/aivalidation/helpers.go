package aivalidation

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
)

type itemSpan struct {
	dayNumber int
	itemIndex int
	start     int
	end       int
	item      aggregate.ItineraryItem
}

func issue(id string, category IssueCategory, severity IssueSeverity, title, description string, fixability IssueFixability) ValidationIssue {
	return ValidationIssue{
		ID:          id,
		Category:    category,
		Severity:    severity,
		Title:       title,
		Description: description,
		Fixability:  fixability,
	}
}

func issueWithLocation(id string, category IssueCategory, severity IssueSeverity, title, description string, fixability IssueFixability, dayNumber *int, itemIndex *int) ValidationIssue {
	out := issue(id, category, severity, title, description, fixability)
	out.DayNumber = dayNumber
	out.ItemIndex = itemIndex
	return out
}

func issueWithRouteLeg(id string, category IssueCategory, severity IssueSeverity, title, description string, fixability IssueFixability, dayNumber *int, itemIndex *int, routeLegID string) ValidationIssue {
	out := issueWithLocation(id, category, severity, title, description, fixability, dayNumber, itemIndex)
	out.RouteLegID = routeLegID
	return out
}

func filterIssues(issues []ValidationIssue, fn func(ValidationIssue) bool) []ValidationIssue {
	out := make([]ValidationIssue, 0)
	for _, issue := range issues {
		if fn(issue) {
			out = append(out, issue)
		}
	}
	return out
}

func issueIDs(issues []ValidationIssue) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		out = append(out, issue.ID)
	}
	sort.Strings(out)
	return out
}

func issueIDSet(issues []ValidationIssue) map[string]ValidationIssue {
	out := make(map[string]ValidationIssue, len(issues))
	for _, issue := range issues {
		out[issue.ID] = issue
	}
	return out
}

func warningsFromIssues(issues []ValidationIssue) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, issue := range issues {
		if issue.Severity != SeverityWarning && issue.Severity != SeverityInfo {
			continue
		}
		value := strings.TrimSpace(issue.Title)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isSaveBlockingIssue(issue ValidationIssue, cfg Config) bool {
	if issue.Severity == SeverityBlocking {
		return true
	}
	if issue.ID == "itinerary_day_count_mismatch" || strings.HasPrefix(issue.ID, "itinerary_missing_day") || strings.HasPrefix(issue.ID, "itinerary_duplicate_day") {
		return issue.Severity == SeverityCritical
	}
	if issue.Severity != SeverityCritical {
		return false
	}
	switch issue.Category {
	case CategorySchema:
		return cfg.BlockOnSchemaErrors
	case CategoryPolicy:
		return cfg.BlockOnPolicyBlockers
	case CategoryRoute, CategoryTransport:
		return cfg.BlockOnCriticalRouteErrors
	case CategoryBudget:
		return cfg.BlockOnBudgetErrors
	default:
		return true
	}
}

func qualityStatusForValidation(issues []ValidationIssue, saveAllowed bool) GenerationQualityStatus {
	if len(issues) == 0 {
		return StatusValidated
	}
	for _, issue := range issues {
		if issue.Category == CategorySchema && issue.Severity == SeverityCritical {
			return StatusSchemaInvalid
		}
		if issue.Category == CategoryPolicy && issue.Severity == SeverityBlocking {
			return StatusBlockedByPolicy
		}
	}
	if saveAllowed {
		return StatusValidatedWithWarnings
	}
	return StatusBlockedByCriticalIssues
}

func severityRank(severity IssueSeverity) int {
	switch severity {
	case SeverityBlocking:
		return 5
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func parseHHMM(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if len(value) != 5 || value[2] != ':' {
		return 0, false
	}
	for _, index := range []int{0, 1, 3, 4} {
		if value[index] < '0' || value[index] > '9' {
			return 0, false
		}
	}
	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	if hour > 23 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func validHHMM(value string) bool {
	_, ok := parseHHMM(value)
	return ok
}

func itemEndMinute(item aggregate.ItineraryItem, start int) (int, bool) {
	if end, ok := parseHHMM(item.EndTime); ok {
		return end, true
	}
	if item.DurationMinutes != nil && *item.DurationMinutes > 0 {
		return start + *item.DurationMinutes, true
	}
	if isTransportLike(item) {
		return start + 90, true
	}
	return start + 60, true
}

func overlapAllowed(left, right aggregate.ItineraryItem) bool {
	leftType := normalizeToken(left.Type)
	rightType := normalizeToken(right.Type)
	if leftType == "note" || rightType == "note" {
		return true
	}
	if strings.Contains(leftType, "meal") && strings.Contains(rightType, "meal") {
		return true
	}
	return false
}

func validCurrency(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 3 {
		return false
	}
	for _, r := range value {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func normalizeText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func normalizeID(value string) string {
	value = normalizeToken(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return '_'
	}, value)
	return strings.Trim(value, "_")
}

func itemName(item aggregate.ItineraryItem) string {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return "This item"
	}
	return name
}

func isTransportLike(item aggregate.ItineraryItem) bool {
	token := normalizeToken(item.Type + " " + item.TransportMode)
	if item.Transfer != nil {
		return true
	}
	for _, part := range []string{"transport", "transfer", "train", "bus", "flight", "ferry", "boat", "taxi", "car", "metro", "public_transport"} {
		if strings.Contains(token, part) {
			return true
		}
	}
	return false
}

func isOutdoorItem(item aggregate.ItineraryItem) bool {
	token := normalizeToken(item.Type + " " + item.Category + " " + item.Name + " " + item.Note)
	for _, part := range []string{"walk", "hike", "hiking", "outdoor", "park", "beach", "camp", "camping", "bike", "viewpoint", "scenic"} {
		if strings.Contains(token, part) {
			return true
		}
	}
	return false
}

func isHikingItem(item aggregate.ItineraryItem) bool {
	token := normalizeToken(item.Type + " " + item.Category + " " + item.Name)
	return strings.Contains(token, "hiking") || strings.Contains(token, "hike")
}

func isCampingItem(item aggregate.ItineraryItem) bool {
	token := normalizeToken(item.Type + " " + item.Category + " " + item.Name)
	return strings.Contains(token, "camping") || strings.Contains(token, "camp")
}

func likelyNeedsOpeningHours(item aggregate.ItineraryItem) bool {
	token := normalizeToken(item.Type + " " + item.Category + " " + item.Name)
	for _, part := range []string{"food", "restaurant", "cafe", "museum", "ticket", "attraction", "tour", "shop", "market", "activity"} {
		if strings.Contains(token, part) {
			return true
		}
	}
	return false
}

func keyActivity(item aggregate.ItineraryItem) bool {
	token := normalizeToken(item.Type + " " + item.Category + " " + item.Name)
	if isTransportLike(item) {
		return false
	}
	for _, part := range []string{"activity", "museum", "ticket", "attraction", "tour", "food", "restaurant", "landmark", "place"} {
		if strings.Contains(token, part) {
			return true
		}
	}
	return false
}

func stopName(stop aggregate.RouteStop) string {
	for _, value := range []string{stop.City, stop.Destination, stop.Country, stop.ID} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "route stop"
}

func samePlaceHint(value, expected string) bool {
	value = normalizeText(value)
	expected = normalizeText(expected)
	if value == "" || expected == "" {
		return true
	}
	return strings.Contains(value, expected) || strings.Contains(expected, value)
}

func sortedDays(days []aggregate.ItineraryDay) []aggregate.ItineraryDay {
	out := append([]aggregate.ItineraryDay(nil), days...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Day < out[j].Day })
	return out
}

func hasTransferForLeg(left, right aggregate.ItineraryDay, legID string) bool {
	for _, day := range []aggregate.ItineraryDay{left, right} {
		for _, item := range day.Items {
			if item.Transfer != nil && strings.TrimSpace(item.Transfer.LegID) == legID {
				return true
			}
			if strings.TrimSpace(legID) == "" && isTransportLike(item) {
				return true
			}
		}
	}
	return false
}

func routeEndpointName(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "a stop"
}

func parseDate(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	return parsed, err == nil
}

func selectedTransportInterval(option aggregate.SelectedTransportOption) (time.Time, time.Time, bool) {
	departure, ok := parseDateTime(option.DepartureDate, option.DepartureTime)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	arrival, ok := parseDateTime(option.ArrivalDate, option.ArrivalTime)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	return departure, arrival, true
}

func parseDateTime(dateValue, timeValue string) (time.Time, bool) {
	if strings.TrimSpace(dateValue) == "" || strings.TrimSpace(timeValue) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02 15:04", strings.TrimSpace(dateValue)+" "+strings.TrimSpace(timeValue))
	return parsed, err == nil
}

func absoluteItemInterval(day aggregate.ItineraryDay, item aggregate.ItineraryItem) (time.Time, time.Time, bool) {
	startMinute, ok := parseHHMM(item.Time)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	endMinute, ok := itemEndMinute(item, startMinute)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	dateValue, ok := parseDate(day.Date)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	start := dateValue.Add(time.Duration(startMinute) * time.Minute)
	end := dateValue.Add(time.Duration(endMinute) * time.Minute)
	return start, end, end.After(start)
}

func intervalsOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func sameDate(left, right time.Time) bool {
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

func withinOpeningHours(hours []aggregate.OpeningHoursInterval, weekday int, itemStart int) bool {
	for _, interval := range hours {
		if interval.DayOfWeek != weekday {
			continue
		}
		open, openOK := parseHHMM(interval.Open)
		close, closeOK := parseHHMM(interval.Close)
		if !openOK || !closeOK {
			continue
		}
		if close < open {
			if itemStart >= open || itemStart <= close {
				return true
			}
			continue
		}
		if itemStart >= open && itemStart <= close {
			return true
		}
	}
	return false
}

func weatherRisk(day weathercontext.WeatherDay) bool {
	condition := normalizeToken(day.Condition + " " + day.Summary + " " + strings.Join(day.Warnings, " "))
	return day.PrecipitationChance >= 70 ||
		day.TemperatureMaxC >= 33 ||
		day.TemperatureMinC <= -5 ||
		strings.Contains(condition, "rain") ||
		strings.Contains(condition, "storm") ||
		strings.Contains(condition, "snow")
}

func severeWeather(day weathercontext.WeatherDay) bool {
	condition := normalizeToken(day.Condition + " " + day.Summary + " " + strings.Join(day.Warnings, " "))
	return day.PrecipitationChance >= 85 ||
		day.TemperatureMaxC >= 38 ||
		day.TemperatureMinC <= -10 ||
		day.WindSpeedKph >= 60 ||
		strings.Contains(condition, "severe") ||
		strings.Contains(condition, "storm") ||
		strings.Contains(condition, "thunder") ||
		strings.Contains(condition, "snow")
}

func itineraryNames(itinerary aggregate.Itinerary) string {
	parts := make([]string, 0)
	for _, day := range itinerary.Days {
		parts = append(parts, day.Title)
		for _, item := range day.Items {
			parts = append(parts, item.Name, item.Note)
		}
	}
	return strings.Join(parts, " ")
}

func joinIssueIDs(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", ids)
}
