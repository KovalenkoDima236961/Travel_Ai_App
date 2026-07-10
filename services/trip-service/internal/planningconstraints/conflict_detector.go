package planningconstraints

import (
	"fmt"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

func DetectConflicts(c *PlanningConstraints) {
	if c == nil {
		return
	}
	c.Warnings = []Issue{}
	c.Blockers = []Issue{}
	detectDateConflicts(c)
	detectTransportConflicts(c)
	detectRouteConflicts(c)
	detectBudgetConflicts(c)
	detectStyleConflicts(c)
}

func detectTransportConflicts(c *PlanningConstraints) {
	for _, preferred := range c.Transport.PreferredModes {
		if contains(c.Transport.AvoidModes, preferred) {
			c.addIssue(Issue{
				Type:     "transport_mode_preferred_and_avoided",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s is both preferred and avoided.", preferred),
				Source:   "user_preferences",
				Affected: map[string]any{"mode": preferred},
				SuggestedActions: []SuggestedAction{
					{Type: "remove_disallowed_mode", Label: "Remove the duplicate transport preference"},
				},
			})
		}
	}
	if !c.Transport.CarAvailable {
		for _, mode := range append([]string{}, c.Transport.PreferredModes...) {
			if mode == aggregate.TransportModeCar || mode == aggregate.TransportModeRentalCar {
				c.addIssue(Issue{
					Type:     "car_selected_without_car_available",
					Severity: SeverityWarning,
					Message:  "Car travel is selected, but carAvailable is false.",
					Source:   "request",
					Affected: map[string]any{"mode": mode},
					SuggestedActions: []SuggestedAction{
						{Type: "enable_car_available", Label: "Mark a car as available"},
						{Type: "change_transport_mode", Label: "Choose train, bus, or public transport"},
					},
				})
			}
		}
	}
	if c.Route == nil {
		return
	}
	for _, leg := range c.Route.Legs {
		mode := aggregate.NormalizeRouteToken(leg.Mode)
		if contains(c.Transport.DisallowedModes, mode) {
			c.addIssue(Issue{
				Type:     "transport_mode_disallowed",
				Severity: SeverityBlocking,
				Message:  fmt.Sprintf("%s is selected, but workspace policy disallows it.", mode),
				Source:   "workspace_policy",
				Affected: map[string]any{"mode": mode, "legId": leg.ID},
				SuggestedActions: []SuggestedAction{
					{Type: "change_transport_mode", Label: "Choose an allowed transport mode"},
					{Type: "open_route_builder", Label: "Open route builder"},
				},
			})
		}
		if (mode == aggregate.TransportModeCar || mode == aggregate.TransportModeRentalCar) && !c.Transport.CarAvailable {
			c.addIssue(Issue{
				Type:     "car_route_without_car_available",
				Severity: SeverityWarning,
				Message:  "A route leg uses car travel, but no car is marked as available.",
				Source:   "route",
				Affected: map[string]any{"mode": mode, "legId": leg.ID},
				SuggestedActions: []SuggestedAction{
					{Type: "enable_car_available", Label: "Mark a car as available"},
				},
			})
		}
		if mode == aggregate.TransportModeFlight && leg.EstimatedDistanceKm != nil && *leg.EstimatedDistanceKm < 400 && contains(c.Transport.PreferredModes, aggregate.TransportModeTrain) {
			c.addIssue(Issue{
				Type:     "short_flight_with_train_preferred",
				Severity: SeverityInfo,
				Message:  "A short flight is selected while train travel is preferred.",
				Source:   "route",
				Affected: map[string]any{"mode": mode, "legId": leg.ID, "distanceKm": *leg.EstimatedDistanceKm},
				SuggestedActions: []SuggestedAction{
					{Type: "change_transport_mode", Label: "Compare train or bus"},
				},
			})
		}
	}
}

func detectRouteConflicts(c *PlanningConstraints) {
	if c.Route == nil {
		return
	}
	duration := c.Dates.DurationDays
	if duration > 0 {
		if len(c.Route.Stops) > duration {
			c.addIssue(Issue{
				Type:     "too_many_stops_for_duration",
				Severity: SeverityWarning,
				Message:  "The route has more stops than trip days.",
				Source:   "route",
				Affected: map[string]any{"stopCount": len(c.Route.Stops), "durationDays": duration},
				SuggestedActions: []SuggestedAction{
					{Type: "reduce_stops", Label: "Reduce stops"},
					{Type: "increase_duration", Label: "Increase duration"},
				},
			})
		} else if len(c.Route.Stops) >= duration && duration <= 3 {
			c.addIssue(Issue{
				Type:     "dense_short_route",
				Severity: SeverityWarning,
				Message:  "This is a dense route for a short trip.",
				Source:   "route",
				Affected: map[string]any{"stopCount": len(c.Route.Stops), "durationDays": duration},
				SuggestedActions: []SuggestedAction{
					{Type: "reduce_stops", Label: "Reduce stops"},
					{Type: "lower_pace", Label: "Use a slower pace"},
				},
			})
		}
	}
	for _, leg := range c.Route.Legs {
		if leg.EstimatedDurationMinutes == nil {
			c.addIssue(Issue{
				Type:     "route_leg_missing_estimate",
				Severity: SeverityInfo,
				Message:  "A route leg is missing a duration estimate.",
				Source:   "route",
				Affected: map[string]any{"legId": leg.ID},
				SuggestedActions: []SuggestedAction{
					{Type: "open_route_builder", Label: "Add route estimate"},
				},
			})
			continue
		}
		if c.Transport.MaxTransferHoursPerDay != nil && *leg.EstimatedDurationMinutes > *c.Transport.MaxTransferHoursPerDay*60 {
			severity := SeverityWarning
			source := "route"
			if c.WorkspacePolicy != nil && contains(c.WorkspacePolicy.BlockingRules, "maxTransferHoursPerDay") {
				severity = SeverityBlocking
				source = "workspace_policy"
			}
			c.addIssue(Issue{
				Type:     "transfer_exceeds_max_hours",
				Severity: severity,
				Message:  "A route transfer exceeds the configured maximum transfer time per day.",
				Source:   source,
				Affected: map[string]any{"legId": leg.ID, "durationMinutes": *leg.EstimatedDurationMinutes, "maxHours": *c.Transport.MaxTransferHoursPerDay},
				SuggestedActions: []SuggestedAction{
					{Type: "change_transport_mode", Label: "Choose a shorter transfer"},
					{Type: "increase_duration", Label: "Add a buffer day"},
				},
			})
		}
	}
	if contains(c.TripStyles, "island_hopping") {
		hasBoat := false
		for _, mode := range append(append([]string{}, c.Transport.PreferredModes...), c.Transport.AllowedModes...) {
			if mode == aggregate.TransportModeBoat || mode == aggregate.TransportModeFerry {
				hasBoat = true
				break
			}
		}
		if !hasBoat {
			c.addIssue(Issue{
				Type:     "island_hopping_without_boat_mode",
				Severity: SeverityInfo,
				Message:  "Island hopping usually needs ferry or boat transport.",
				Source:   "trip_style",
				SuggestedActions: []SuggestedAction{
					{Type: "change_transport_mode", Label: "Allow ferry or boat"},
				},
			})
		}
	}
}

func detectBudgetConflicts(c *PlanningConstraints) {
	if c.Budget == nil || c.Budget.Amount == nil {
		c.addIssue(Issue{
			Type:     "budget_missing",
			Severity: SeverityInfo,
			Message:  "No budget amount is configured; AI will treat budget loosely.",
			Source:   "request",
			SuggestedActions: []SuggestedAction{
				{Type: "open_budget_settings", Label: "Set a budget"},
			},
		})
		return
	}
	if c.WorkspacePolicy != nil && contains(c.WorkspacePolicy.BlockingRules, "maxTripBudget") && c.WorkspacePolicy.Rules != nil {
		var doc struct {
			Rules struct {
				MaxTripBudget struct {
					Enabled  bool    `json:"enabled"`
					Severity string  `json:"severity"`
					Amount   float64 `json:"amount"`
					Currency string  `json:"currency"`
				} `json:"maxTripBudget"`
			} `json:"rules"`
		}
		if err := jsonUnmarshal(c.WorkspacePolicy.Rules, &doc); err == nil &&
			doc.Rules.MaxTripBudget.Enabled &&
			strings.EqualFold(doc.Rules.MaxTripBudget.Currency, c.Budget.Currency) &&
			*c.Budget.Amount > doc.Rules.MaxTripBudget.Amount {
			c.addIssue(Issue{
				Type:     "workspace_max_trip_budget_exceeded",
				Severity: SeverityBlocking,
				Message:  "The requested budget exceeds the workspace maximum trip budget.",
				Source:   "workspace_policy",
				Affected: map[string]any{"amount": *c.Budget.Amount, "currency": c.Budget.Currency, "maxAmount": doc.Rules.MaxTripBudget.Amount},
				SuggestedActions: []SuggestedAction{
					{Type: "increase_budget", Label: "Review workspace budget policy"},
					{Type: "adjust_workspace_policy", Label: "Adjust workspace policy"},
				},
			})
		}
	}
	if c.Route != nil && *c.Budget.Amount > 0 {
		totalTransfer := 0.0
		for _, leg := range c.Route.Legs {
			if leg.EstimatedCost != nil && leg.EstimatedCost.Amount != nil && strings.EqualFold(leg.EstimatedCost.Currency, c.Budget.Currency) {
				totalTransfer += *leg.EstimatedCost.Amount
			}
		}
		if totalTransfer > *c.Budget.Amount*0.5 {
			c.addIssue(Issue{
				Type:     "transfer_cost_high_share_of_budget",
				Severity: SeverityWarning,
				Message:  "Estimated route transfer costs use more than half of the budget.",
				Source:   "budget",
				Affected: map[string]any{"transferCost": totalTransfer, "budget": *c.Budget.Amount, "currency": c.Budget.Currency},
				SuggestedActions: []SuggestedAction{
					{Type: "open_budget_settings", Label: "Review budget"},
					{Type: "change_transport_mode", Label: "Use cheaper transport"},
				},
			})
		}
	}
	if c.Budget.Strictness == "strict" && contains(c.TripStyles, "luxury") {
		c.addIssue(Issue{
			Type:     "strict_budget_luxury_style",
			Severity: SeverityWarning,
			Message:  "Strict budget conflicts with luxury trip style.",
			Source:   "budget",
			SuggestedActions: []SuggestedAction{
				{Type: "increase_budget", Label: "Increase budget"},
				{Type: "disable_hiking", Label: "Remove luxury style"},
			},
		})
	}
}

func detectDateConflicts(c *PlanningConstraints) {
	if c.Dates.StartDate == "" || c.Dates.EndDate == "" {
		return
	}
	start, startErr := time.Parse("2006-01-02", c.Dates.StartDate)
	end, endErr := time.Parse("2006-01-02", c.Dates.EndDate)
	if startErr != nil || endErr != nil {
		c.addIssue(Issue{
			Type:     "invalid_dates",
			Severity: SeverityBlocking,
			Message:  "Trip dates must use YYYY-MM-DD format.",
			Source:   "request",
			SuggestedActions: []SuggestedAction{
				{Type: "open_preferences", Label: "Fix dates"},
			},
		})
		return
	}
	if start.After(end) {
		c.addIssue(Issue{
			Type:     "start_date_after_end_date",
			Severity: SeverityBlocking,
			Message:  "Start date is after end date.",
			Source:   "request",
			Affected: map[string]any{"startDate": c.Dates.StartDate, "endDate": c.Dates.EndDate},
			SuggestedActions: []SuggestedAction{
				{Type: "open_preferences", Label: "Fix dates"},
			},
		})
	}
	if c.Dates.DurationDays > 0 {
		expected := int(end.Sub(start).Hours()/24) + 1
		if expected != c.Dates.DurationDays {
			c.addIssue(Issue{
				Type:     "duration_mismatch",
				Severity: SeverityWarning,
				Message:  "Duration does not match the date range.",
				Source:   "request",
				Affected: map[string]any{"durationDays": c.Dates.DurationDays, "dateRangeDays": expected},
				SuggestedActions: []SuggestedAction{
					{Type: "increase_duration", Label: "Adjust duration"},
				},
			})
		}
	}
}

func detectStyleConflicts(c *PlanningConstraints) {
	if contains(c.TripStyles, "hiking") && c.Walking.MaxKmPerDay != nil && *c.Walking.MaxKmPerDay <= 5 && !c.Walking.AllowLongHikes {
		c.addIssue(Issue{
			Type:     "hiking_low_walking_conflict",
			Severity: SeverityWarning,
			Message:  "Hiking conflicts with a low walking preference.",
			Source:   "trip_style",
			SuggestedActions: []SuggestedAction{
				{Type: "disable_hiking", Label: "Remove hiking style"},
				{Type: "lower_pace", Label: "Keep hikes short and optional"},
			},
		})
	}
	if contains(c.TripStyles, "hiking") && c.Pace == "packed" {
		c.addIssue(Issue{
			Type:     "hiking_packed_pace",
			Severity: SeverityWarning,
			Message:  "Hiking with a packed pace can make the itinerary unrealistic.",
			Source:   "trip_style",
			SuggestedActions: []SuggestedAction{
				{Type: "lower_pace", Label: "Use a relaxed or balanced pace"},
			},
		})
	}
	if contains(c.TripStyles, "camping") && contains(c.Accommodation.AvoidTypes, "campsite") {
		c.addIssue(Issue{
			Type:     "camping_campsite_avoided",
			Severity: SeverityWarning,
			Message:  "Camping style conflicts with avoiding campsites.",
			Source:   "accommodation",
			SuggestedActions: []SuggestedAction{
				{Type: "change_accommodation_type", Label: "Allow campsite or cabin"},
			},
		})
	}
	if contains(c.TripStyles, "camping") && !c.Accommodation.CampingAllowed {
		c.addIssue(Issue{
			Type:     "camping_without_accommodation_hint",
			Severity: SeverityWarning,
			Message:  "Camping style is selected but camping accommodation is not allowed.",
			Source:   "accommodation",
			SuggestedActions: []SuggestedAction{
				{Type: "change_accommodation_type", Label: "Allow camping accommodation"},
			},
		})
	}
	for _, avoid := range c.Avoid {
		for _, interest := range append(append([]string{}, c.Interests...), c.MustHave...) {
			if normalizeToken(avoid) != "" && normalizeToken(avoid) == normalizeToken(interest) {
				c.addIssue(Issue{
					Type:     "avoid_conflicts_with_interest",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%q appears in both avoid and interest/must-have lists.", interest),
					Source:   "preferences",
					Affected: map[string]any{"value": interest},
					SuggestedActions: []SuggestedAction{
						{Type: "open_preferences", Label: "Review preferences"},
					},
				})
			}
		}
	}
	if c.Accessibility.LowWalkingRequired && contains(c.TripStyles, "hiking") {
		c.addIssue(Issue{
			Type:     "hiking_low_walking_required",
			Severity: SeverityWarning,
			Message:  "Hiking conflicts with low-walking accessibility requirements.",
			Source:   "accessibility",
			SuggestedActions: []SuggestedAction{
				{Type: "disable_hiking", Label: "Remove hiking style"},
			},
		})
	}
}

func (c *PlanningConstraints) addIssue(issue Issue) {
	if issue.SuggestedActions == nil {
		issue.SuggestedActions = []SuggestedAction{}
	}
	if issue.Severity == SeverityBlocking {
		c.Blockers = append(c.Blockers, issue)
		return
	}
	c.Warnings = append(c.Warnings, issue)
}
