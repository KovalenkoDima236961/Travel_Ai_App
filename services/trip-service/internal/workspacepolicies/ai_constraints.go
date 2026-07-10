package workspacepolicies

import (
	"encoding/json"
	"fmt"
	"strings"
)

func BuildAIConstraints(policy *Policy) *AIConstraints {
	if policy == nil {
		return nil
	}
	r := policy.Rules.Rules
	lines := make([]string, 0, 13)
	if r.RequireTripBudget.Enabled {
		lines = append(lines, "Include realistic cost estimates so the trip budget can be reviewed.")
	}
	if r.MaxTripBudget.Enabled {
		lines = append(lines, fmt.Sprintf(
			"The total estimated trip cost should not exceed %.2f %s.",
			r.MaxTripBudget.Amount, r.MaxTripBudget.Currency,
		))
	}
	if r.MaxDailyBudget.Enabled {
		lines = append(lines, fmt.Sprintf(
			"Each day's estimated cost should stay at or below %.2f %s.",
			r.MaxDailyBudget.Amount, r.MaxDailyBudget.Currency,
		))
	}
	if r.MaxItemCost.Enabled {
		suffix := ""
		if len(r.MaxItemCost.Categories) > 0 {
			suffix = " for " + strings.Join(r.MaxItemCost.Categories, ", ")
		}
		lines = append(lines, fmt.Sprintf(
			"Avoid individual item costs above %.2f %s%s.",
			r.MaxItemCost.Amount, r.MaxItemCost.Currency, suffix,
		))
	}
	if r.MaxAccommodationTotal.Enabled {
		lines = append(lines, fmt.Sprintf(
			"Accommodation total should stay at or below %.2f %s.",
			r.MaxAccommodationTotal.Amount, r.MaxAccommodationTotal.Currency,
		))
	}
	if r.MaxAccommodationPerNight.Enabled {
		lines = append(lines, fmt.Sprintf(
			"Accommodation should stay at or below %.2f %s per night.",
			r.MaxAccommodationPerNight.Amount, r.MaxAccommodationPerNight.Currency,
		))
	}
	if r.RequireCostSplitting.Enabled {
		lines = append(lines, "Keep cost estimates explicit so costs can be split between travelers.")
	}
	if r.RequireAvailabilityForTicketedItems.Enabled {
		lines = append(lines, "Prefer availability-checkable ticketed items and mark availability as unchecked.")
	}
	if r.MaxWalkingKmPerDay.Enabled {
		lines = append(lines, fmt.Sprintf(
			"Keep estimated walking distance at or below %.2f km per day.",
			r.MaxWalkingKmPerDay.Km,
		))
	}
	if r.NoLateActivitiesAfter.Enabled {
		lines = append(lines, "Avoid activities after "+r.NoLateActivitiesAfter.Time+".")
	}
	if r.RequiredRestTimePerDay.Enabled {
		lines = append(lines, fmt.Sprintf(
			"Include at least %d minutes of rest or free time per day.",
			r.RequiredRestTimePerDay.Minutes,
		))
	}
	if r.PreferredTransportModes.Enabled && len(r.PreferredTransportModes.Modes) > 0 {
		lines = append(lines, "Prefer "+strings.Join(r.PreferredTransportModes.Modes, " and ")+".")
	}
	if r.MaxTransferHoursPerDay.Enabled {
		lines = append(lines, fmt.Sprintf(
			"Keep transfer time at or below %.2f hours per day.",
			r.MaxTransferHoursPerDay.Hours,
		))
	}
	if r.DisallowedTransportModes.Enabled && len(r.DisallowedTransportModes.Modes) > 0 {
		lines = append(lines, "Do not use "+strings.Join(r.DisallowedTransportModes.Modes, " or ")+".")
	}
	if r.DisallowedActivityTypes.Enabled && len(r.DisallowedActivityTypes.Types) > 0 {
		lines = append(lines, "Do not include "+strings.Join(r.DisallowedActivityTypes.Types, " or ")+".")
	}
	if len(lines) == 0 {
		return nil
	}
	raw, _ := json.Marshal(policy.Rules)
	return &AIConstraints{Summary: strings.Join(lines, "\n"), Rules: raw}
}
