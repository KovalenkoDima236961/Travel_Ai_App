package calendarsync

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/calendarclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const DefaultTimeZone = "Europe/Bratislava"

type BuildInput struct {
	Trip     *entity.Trip
	TripURL  string
	TimeZone string
}

type BuildResult struct {
	Items   []calendarclient.SyncItem
	Skipped int
}

func BuildEvents(input BuildInput) (*BuildResult, error) {
	if input.Trip == nil {
		return nil, fmt.Errorf("trip is required")
	}
	if input.Trip.StartDate == nil {
		return nil, fmt.Errorf("trip startDate is required")
	}
	if len(input.Trip.Itinerary) == 0 {
		return nil, fmt.Errorf("trip itinerary is required")
	}
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(input.Trip.Itinerary, &itinerary); err != nil {
		return nil, fmt.Errorf("decode itinerary: %w", err)
	}
	timeZone := strings.TrimSpace(input.TimeZone)
	if timeZone == "" {
		timeZone = DefaultTimeZone
	}
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		loc = time.UTC
	}

	result := &BuildResult{Items: make([]calendarclient.SyncItem, 0)}
	for dayIndex, day := range itinerary.Days {
		dayNumber := day.Day
		if dayNumber <= 0 {
			dayNumber = dayIndex + 1
		}
		date := time.Date(
			input.Trip.StartDate.Year(),
			input.Trip.StartDate.Month(),
			input.Trip.StartDate.Day(),
			0,
			0,
			0,
			0,
			loc,
		).AddDate(0, 0, dayNumber-1)
		for itemIndex, item := range day.Items {
			startClock, endClock, ok := parseTimeRange(item.Time)
			if !ok {
				result.Skipped++
				continue
			}
			start := time.Date(date.Year(), date.Month(), date.Day(), startClock.hour, startClock.minute, 0, 0, loc)
			var end time.Time
			if endClock != nil {
				end = time.Date(date.Year(), date.Month(), date.Day(), endClock.hour, endClock.minute, 0, 0, loc)
				if !end.After(start) {
					end = end.Add(24 * time.Hour)
				}
			} else {
				end = start.Add(defaultDuration(item.Type))
			}
			result.Items = append(result.Items, calendarclient.SyncItem{
				SyncKey:     fmt.Sprintf("day-%d-item-%d", dayNumber, itemIndex),
				DayNumber:   dayNumber,
				ItemIndex:   itemIndex,
				Title:       strings.TrimSpace(item.Name),
				Description: buildDescription(item, input.TripURL, input.Trip.BudgetCurrency),
				Location:    itemLocation(item, input.Trip.Destination),
				MapURL:      itemMapURL(item),
				Start:       start,
				End:         end,
			})
		}
	}
	return result, nil
}

type clock struct {
	hour   int
	minute int
}

var timeTokenRe = regexp.MustCompile(`(?i)\b(\d{1,2}):(\d{2})\s*(AM|PM)?\b`)

func parseTimeRange(raw string) (clock, *clock, bool) {
	normalized := strings.NewReplacer("–", "-", "—", "-", "−", "-").Replace(strings.TrimSpace(raw))
	if normalized == "" {
		return clock{}, nil, false
	}
	matches := timeTokenRe.FindAllStringSubmatch(normalized, 2)
	if len(matches) == 0 {
		return clock{}, nil, false
	}
	start, ok := parseClockMatch(matches[0])
	if !ok {
		return clock{}, nil, false
	}
	if len(matches) < 2 {
		return start, nil, true
	}
	end, ok := parseClockMatch(matches[1])
	if !ok {
		return start, nil, true
	}
	return start, &end, true
}

func parseClockMatch(match []string) (clock, bool) {
	if len(match) < 4 {
		return clock{}, false
	}
	hour, minute, ok := parseHourMinute(match[1], match[2])
	if !ok {
		return clock{}, false
	}
	period := strings.ToUpper(strings.TrimSpace(match[3]))
	if period == "AM" {
		if hour == 12 {
			hour = 0
		}
	} else if period == "PM" {
		if hour < 12 {
			hour += 12
		}
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return clock{}, false
	}
	return clock{hour: hour, minute: minute}, true
}

func parseHourMinute(hourPart, minutePart string) (int, int, bool) {
	var hour, minute int
	if _, err := fmt.Sscanf(hourPart+":"+minutePart, "%d:%d", &hour, &minute); err != nil {
		return 0, 0, false
	}
	return hour, minute, true
}

func defaultDuration(itemType string) time.Duration {
	switch strings.ToLower(strings.TrimSpace(itemType)) {
	case "food", "restaurant", "cafe":
		return 90 * time.Minute
	case "transport":
		return 30 * time.Minute
	case "rest", "break", "free_time":
		return time.Hour
	default:
		return time.Hour
	}
}

func itemLocation(item aggregate.ItineraryItem, fallback string) string {
	if item.Place != nil {
		if strings.TrimSpace(item.Place.Address) != "" {
			return strings.TrimSpace(item.Place.Address)
		}
		if strings.TrimSpace(item.Place.Name) != "" {
			return strings.TrimSpace(item.Place.Name)
		}
	}
	return strings.TrimSpace(fallback)
}

func itemMapURL(item aggregate.ItineraryItem) string {
	if item.Place == nil {
		return ""
	}
	return strings.TrimSpace(item.Place.MapURL)
}

func buildDescription(item aggregate.ItineraryItem, tripURL, currency string) string {
	parts := make([]string, 0, 4)
	if note := strings.TrimSpace(item.Note); note != "" {
		parts = append(parts, note)
	}
	if mapURL := itemMapURL(item); mapURL != "" {
		parts = append(parts, "Map: "+mapURL)
	}
	if item.EstimatedCost != nil {
		parts = append(parts, fmt.Sprintf("Estimated cost: %.2f %s", *item.EstimatedCost, strings.TrimSpace(currency)))
	}
	if strings.TrimSpace(tripURL) != "" {
		parts = append(parts, "Trip: "+strings.TrimSpace(tripURL))
	}
	return strings.Join(parts, "\n\n")
}
