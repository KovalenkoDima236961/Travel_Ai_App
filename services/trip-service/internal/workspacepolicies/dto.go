package workspacepolicies

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	timePattern     = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d$`)
)

const maxRuleArrayLength = 30

func ValidateInput(input *UpsertInput) error {
	if input == nil {
		return fmt.Errorf("policy is required")
	}
	input.Name = strings.TrimSpace(input.Name)
	if length := len([]rune(input.Name)); length < 2 || length > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		if len([]rune(value)) > 500 {
			return fmt.Errorf("description must be at most 500 characters")
		}
		if value == "" {
			input.Description = nil
		} else {
			input.Description = &value
		}
	}
	if input.Rules.SchemaVersion != SchemaVersion {
		return fmt.Errorf("rules.schemaVersion must be 1")
	}
	rules := &input.Rules.Rules
	baseRules := []Rule{
		rules.RequireTripBudget,
		rules.MaxTripBudget.Rule,
		rules.MaxDailyBudget.Rule,
		rules.MaxItemCost.Rule,
		rules.MaxAccommodationTotal.Rule,
		rules.MaxAccommodationPerNight.Rule,
		rules.RequireCostSplitting,
		rules.RequireAvailabilityForTicketedItems,
		rules.MaxWalkingKmPerDay.Rule,
		rules.NoLateActivitiesAfter.Rule,
		rules.RequiredRestTimePerDay.Rule,
		rules.PreferredTransportModes.Rule,
		rules.DisallowedActivityTypes.Rule,
	}
	for _, rule := range baseRules {
		if !rule.Severity.Valid() {
			return fmt.Errorf("every rule severity must be info, warning, or blocking")
		}
	}
	for _, rule := range []*MoneyRule{
		&rules.MaxTripBudget,
		&rules.MaxDailyBudget,
		&rules.MaxItemCost.MoneyRule,
		&rules.MaxAccommodationTotal,
		&rules.MaxAccommodationPerNight,
	} {
		rule.Currency = strings.ToUpper(strings.TrimSpace(rule.Currency))
		if rule.Amount < 0 {
			return fmt.Errorf("policy amounts must be greater than or equal to 0")
		}
		if rule.Enabled && !currencyPattern.MatchString(rule.Currency) {
			return fmt.Errorf("enabled money rules require a 3-letter uppercase currency")
		}
	}
	rules.MaxItemCost.Categories = normalizeList(rules.MaxItemCost.Categories)
	rules.PreferredTransportModes.Modes = normalizeList(rules.PreferredTransportModes.Modes)
	rules.DisallowedActivityTypes.Types = normalizeList(rules.DisallowedActivityTypes.Types)
	for _, values := range [][]string{
		rules.MaxItemCost.Categories,
		rules.PreferredTransportModes.Modes,
		rules.DisallowedActivityTypes.Types,
	} {
		if len(values) > maxRuleArrayLength {
			return fmt.Errorf("policy rule arrays may contain at most %d values", maxRuleArrayLength)
		}
	}
	if rules.MaxWalkingKmPerDay.Enabled && rules.MaxWalkingKmPerDay.Km <= 0 {
		return fmt.Errorf("maxWalkingKmPerDay.km must be greater than 0")
	}
	rules.NoLateActivitiesAfter.Time = strings.TrimSpace(rules.NoLateActivitiesAfter.Time)
	if rules.NoLateActivitiesAfter.Enabled &&
		!timePattern.MatchString(rules.NoLateActivitiesAfter.Time) {
		return fmt.Errorf("noLateActivitiesAfter.time must be HH:mm")
	}
	if rules.RequiredRestTimePerDay.Minutes < 0 {
		return fmt.Errorf("requiredRestTimePerDay.minutes must be greater than or equal to 0")
	}
	return nil
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalizeToken(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}
