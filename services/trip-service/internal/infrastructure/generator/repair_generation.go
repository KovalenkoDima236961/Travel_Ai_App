package generator

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aivalidation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

func NewGenerationRepairClient(cfg *config.Config, logger *zap.Logger) (aivalidation.RepairClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.ItineraryGenerator.Mode))
	if mode == "" {
		mode = generatorModeMock
	}
	switch mode {
	case generatorModeMock:
		return &MockGenerationRepairClient{}, nil
	case generatorModeHTTP:
		timeoutSeconds := cfg.ItineraryGenerator.AIPlanningTimeoutSeconds
		if timeoutSeconds <= 0 {
			return nil, fmt.Errorf("AI_PLANNING_TIMEOUT_SECONDS must be greater than 0")
		}
		client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
		return NewAIPlanningHTTPGenerator(cfg.ItineraryGenerator.AIPlanningServiceURL, client, logger)
	default:
		return nil, fmt.Errorf("unknown ITINERARY_GENERATOR_MODE %q (valid values: mock, http)", cfg.ItineraryGenerator.Mode)
	}
}

func (g *AIPlanningHTTPGenerator) RepairGenerationOutput(
	ctx context.Context,
	request aivalidation.RepairGenerationOutputRequest,
) (*aivalidation.RepairGenerationOutputResponse, error) {
	var result aivalidation.RepairGenerationOutputResponse
	if err := g.postJSON(ctx, request.PlanningContext.Trip.ID, "repair-generation-output", request, &result); err != nil {
		return nil, err
	}
	enrichItineraryDefaults(&result.RepairedOutput, request.PlanningContext.Trip)
	return &result, nil
}

func (g *AIPlanningHTTPGenerator) ProviderMode() string {
	return "http"
}

type MockGenerationRepairClient struct{}

func (c *MockGenerationRepairClient) ProviderMode() string {
	return "mock"
}

func (c *MockGenerationRepairClient) RepairGenerationOutput(
	_ context.Context,
	request aivalidation.RepairGenerationOutputRequest,
) (*aivalidation.RepairGenerationOutputResponse, error) {
	repaired := cloneItinerary(request.CurrentOutput)
	changes := make([]aivalidation.RepairChange, 0)
	for _, issue := range request.ValidationIssues {
		switch {
		case strings.HasPrefix(issue.ID, "activity_during_transport"),
			strings.HasPrefix(issue.ID, "activity_before_transport_arrival"):
			if issue.DayNumber != nil && issue.ItemIndex != nil {
				if moveItemAfterTransport(&repaired, request.PlanningContext.Route, *issue.DayNumber, *issue.ItemIndex) {
					changes = append(changes, aivalidation.RepairChange{
						Type:        "moved_activity",
						Description: "Moved activity after selected transport arrival.",
						DayNumber:   issue.DayNumber,
						ItemIndex:   issue.ItemIndex,
					})
				}
			}
		case strings.HasPrefix(issue.ID, "missing_transfer_between_stops"),
			strings.HasPrefix(issue.ID, "transfer_item_missing_or_mismatch"):
			if addMissingTransfer(&repaired, request.PlanningContext.Route, issue.RouteLegID) {
				changes = append(changes, aivalidation.RepairChange{
					Type:        "added_transfer",
					Description: "Added a transfer item for the selected route leg.",
				})
			}
		case issue.ID == "itinerary_day_count_mismatch" || strings.HasPrefix(issue.ID, "itinerary_missing_day"):
			if normalizeDayCount(&repaired, int(request.PlanningContext.Trip.Days), request.PlanningContext.Trip.Destination) {
				changes = append(changes, aivalidation.RepairChange{
					Type:        "normalized_days",
					Description: "Adjusted itinerary days to match the trip duration.",
				})
			}
		case strings.HasPrefix(issue.ID, "activity_wrong_route_stop"), strings.HasPrefix(issue.ID, "day_missing_primary_stop"):
			if assignRouteStops(&repaired, request.PlanningContext.Route) {
				changes = append(changes, aivalidation.RepairChange{
					Type:        "updated_route_stop",
					Description: "Aligned itinerary days with route stops.",
				})
			}
		case issue.Category == aivalidation.CategoryBudget && issue.DayNumber != nil:
			if lowerDayCosts(&repaired, *issue.DayNumber) {
				changes = append(changes, aivalidation.RepairChange{
					Type:        "reduced_cost",
					Description: "Reduced estimated cost on the affected day.",
					DayNumber:   issue.DayNumber,
				})
			}
		case issue.Category == aivalidation.CategorySchema:
			if repairRequiredFields(&repaired, request.PlanningContext.Trip.Destination) {
				changes = append(changes, aivalidation.RepairChange{
					Type:        "filled_required_fields",
					Description: "Filled missing required itinerary fields.",
				})
			}
		}
	}
	if len(changes) == 0 {
		repairRequiredFields(&repaired, request.PlanningContext.Trip.Destination)
		normalizeDayCount(&repaired, int(request.PlanningContext.Trip.Days), request.PlanningContext.Trip.Destination)
		changes = append(changes, aivalidation.RepairChange{
			Type:        "reviewed_output",
			Description: "Reviewed itinerary and applied deterministic normalization.",
		})
	}
	return &aivalidation.RepairGenerationOutputResponse{
		RepairedOutput: repaired,
		ChangesMade:    changes,
		Warnings:       []string{},
	}, nil
}

func cloneItinerary(input aggregate.Itinerary) aggregate.Itinerary {
	out := input
	out.Days = append([]aggregate.ItineraryDay(nil), input.Days...)
	for dayIndex := range out.Days {
		out.Days[dayIndex].Items = append([]aggregate.ItineraryItem(nil), out.Days[dayIndex].Items...)
	}
	return out
}

func moveItemAfterTransport(itinerary *aggregate.Itinerary, route *aggregate.TripRoute, dayNumber, itemIndex int) bool {
	if route == nil {
		return false
	}
	for dayIndex := range itinerary.Days {
		day := &itinerary.Days[dayIndex]
		if day.Day != dayNumber || itemIndex < 0 || itemIndex >= len(day.Items) {
			continue
		}
		for _, leg := range route.Legs {
			if leg.SelectedTransportOption == nil || leg.ToStopID != day.PrimaryStopID {
				continue
			}
			arrival := strings.TrimSpace(leg.SelectedTransportOption.ArrivalTime)
			if arrival == "" {
				continue
			}
			minutes, ok := parseRepairHHMM(arrival)
			if !ok {
				continue
			}
			minutes += 60
			if minutes > 23*60+30 {
				minutes = 23*60 + 30
			}
			day.Items[itemIndex].Time = formatRepairHHMM(minutes)
			if day.Items[itemIndex].EndTime != "" {
				day.Items[itemIndex].EndTime = formatRepairHHMM(min(minutes+90, 23*60+59))
			}
			return true
		}
	}
	return false
}

func addMissingTransfer(itinerary *aggregate.Itinerary, route *aggregate.TripRoute, routeLegID string) bool {
	if route == nil {
		return false
	}
	for _, leg := range route.Legs {
		if routeLegID != "" && leg.ID != routeLegID {
			continue
		}
		if hasRepairTransfer(*itinerary, leg.ID) {
			return false
		}
		targetDay := findRepairDayForStop(itinerary, leg.ToStopID)
		if targetDay == nil {
			targetDay = findRepairDayForStop(itinerary, leg.FromStopID)
		}
		if targetDay == nil {
			continue
		}
		start := "09:00"
		end := ""
		if leg.SelectedTransportOption != nil {
			if leg.SelectedTransportOption.DepartureTime != "" {
				start = leg.SelectedTransportOption.DepartureTime
			}
			end = leg.SelectedTransportOption.ArrivalTime
		}
		mode := leg.Mode
		if leg.SelectedTransportOption != nil && leg.SelectedTransportOption.Mode != "" {
			mode = leg.SelectedTransportOption.Mode
		}
		targetDay.Items = append([]aggregate.ItineraryItem{{
			Time:          start,
			EndTime:       end,
			Type:          "transport",
			TransportMode: mode,
			Name:          "Transfer to " + routeLegDestination(leg),
			Note:          "Added during AI generation validation repair.",
			Transfer: &aggregate.TransferDetails{
				LegID:                    leg.ID,
				From:                     routeLegOrigin(leg),
				To:                       routeLegDestination(leg),
				Mode:                     mode,
				EstimatedDurationMinutes: leg.EstimatedDurationMinutes,
				EstimatedDistanceKm:      leg.EstimatedDistanceKm,
				EstimatedCost:            leg.EstimatedCost,
				BookingRequired:          leg.SelectedTransportOption != nil,
			},
		}}, targetDay.Items...)
		return true
	}
	return false
}

func normalizeDayCount(itinerary *aggregate.Itinerary, expected int, destination string) bool {
	if expected <= 0 || len(itinerary.Days) == expected {
		return false
	}
	changed := false
	for len(itinerary.Days) > expected {
		itinerary.Days = itinerary.Days[:len(itinerary.Days)-1]
		changed = true
	}
	for len(itinerary.Days) < expected {
		dayNumber := len(itinerary.Days) + 1
		itinerary.Days = append(itinerary.Days, aggregate.ItineraryDay{
			Day:   dayNumber,
			Title: fmt.Sprintf("Day %d buffer in %s", dayNumber, destinationOrFallback(destination)),
			Items: []aggregate.ItineraryItem{{
				Time:          "10:00",
				Type:          "rest",
				Name:          "Flexible planning buffer",
				Note:          "Added during AI generation validation repair.",
				EstimatedCost: zeroRepairCost(itinerary.Currency),
			}},
		})
		changed = true
	}
	for index := range itinerary.Days {
		if itinerary.Days[index].Day != index+1 {
			itinerary.Days[index].Day = index + 1
			changed = true
		}
	}
	return changed
}

func assignRouteStops(itinerary *aggregate.Itinerary, route *aggregate.TripRoute) bool {
	if route == nil || len(route.Stops) == 0 {
		return false
	}
	changed := false
	for index := range itinerary.Days {
		stop := route.Stops[min(index, len(route.Stops)-1)]
		if itinerary.Days[index].PrimaryStopID == "" {
			itinerary.Days[index].PrimaryStopID = stop.ID
			changed = true
		}
		if itinerary.Days[index].LocationName == "" {
			itinerary.Days[index].LocationName = stopNameForRepair(stop)
			changed = true
		}
	}
	return changed
}

func lowerDayCosts(itinerary *aggregate.Itinerary, dayNumber int) bool {
	for dayIndex := range itinerary.Days {
		if itinerary.Days[dayIndex].Day != dayNumber {
			continue
		}
		for itemIndex := range itinerary.Days[dayIndex].Items {
			cost := itinerary.Days[dayIndex].Items[itemIndex].EstimatedCost
			if cost == nil || cost.Amount == nil || *cost.Amount <= 0 {
				continue
			}
			newAmount := *cost.Amount * 0.7
			cost.Amount = &newAmount
			cost.Confidence = "medium"
			cost.Source = "ai"
			itinerary.Days[dayIndex].Items[itemIndex].Note = appendRepairNote(itinerary.Days[dayIndex].Items[itemIndex].Note, "Validation repair lowered the estimated cost.")
			return true
		}
	}
	return false
}

func repairRequiredFields(itinerary *aggregate.Itinerary, destination string) bool {
	changed := false
	if itinerary.Destination == "" {
		itinerary.Destination = destinationOrFallback(destination)
		changed = true
	}
	if itinerary.Currency == "" {
		itinerary.Currency = defaultCurrency
		changed = true
	}
	if itinerary.GeneratedAt.IsZero() {
		itinerary.GeneratedAt = time.Now().UTC()
		changed = true
	}
	if itinerary.Source == "" {
		itinerary.Source = "validation-repair"
		changed = true
	}
	for dayIndex := range itinerary.Days {
		if itinerary.Days[dayIndex].Day <= 0 {
			itinerary.Days[dayIndex].Day = dayIndex + 1
			changed = true
		}
		if itinerary.Days[dayIndex].Title == "" {
			itinerary.Days[dayIndex].Title = fmt.Sprintf("Day %d", itinerary.Days[dayIndex].Day)
			changed = true
		}
		for itemIndex := range itinerary.Days[dayIndex].Items {
			item := &itinerary.Days[dayIndex].Items[itemIndex]
			if item.Time == "" {
				item.Time = "10:00"
				changed = true
			}
			if item.Type == "" {
				item.Type = "activity"
				changed = true
			}
			if item.Name == "" {
				item.Name = "Reviewable itinerary item"
				changed = true
			}
		}
	}
	return changed
}

func zeroRepairCost(currency string) *aggregate.EstimatedCost {
	if strings.TrimSpace(currency) == "" {
		currency = defaultCurrency
	}
	amount := 0.0
	return &aggregate.EstimatedCost{
		Amount:     &amount,
		Currency:   strings.ToUpper(currency),
		Category:   "other",
		Confidence: "high",
		Source:     "ai",
	}
}

func hasRepairTransfer(itinerary aggregate.Itinerary, routeLegID string) bool {
	for _, day := range itinerary.Days {
		for _, item := range day.Items {
			if item.Transfer != nil && item.Transfer.LegID == routeLegID {
				return true
			}
		}
	}
	return false
}

func findRepairDayForStop(itinerary *aggregate.Itinerary, stopID string) *aggregate.ItineraryDay {
	for index := range itinerary.Days {
		if itinerary.Days[index].PrimaryStopID == stopID {
			return &itinerary.Days[index]
		}
	}
	return nil
}

func routeLegOrigin(leg aggregate.RouteLeg) string {
	if strings.TrimSpace(leg.FromName) != "" {
		return leg.FromName
	}
	return leg.FromStopID
}

func routeLegDestination(leg aggregate.RouteLeg) string {
	if strings.TrimSpace(leg.ToName) != "" {
		return leg.ToName
	}
	return leg.ToStopID
}

func stopNameForRepair(stop aggregate.RouteStop) string {
	if stop.City != "" {
		return stop.City
	}
	if stop.Destination != "" {
		return stop.Destination
	}
	return stop.ID
}

func destinationOrFallback(destination string) string {
	if strings.TrimSpace(destination) == "" {
		return "destination"
	}
	return strings.TrimSpace(destination)
}

func appendRepairNote(note, extra string) string {
	if strings.TrimSpace(note) == "" {
		return extra
	}
	if strings.Contains(note, extra) {
		return note
	}
	return note + " " + extra
}

func parseRepairHHMM(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if len(value) != 5 || value[2] != ':' {
		return 0, false
	}
	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func formatRepairHHMM(minutes int) string {
	if minutes < 0 {
		minutes = 0
	}
	if minutes > 23*60+59 {
		minutes = 23*60 + 59
	}
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
