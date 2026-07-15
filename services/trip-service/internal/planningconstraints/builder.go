package planningconstraints

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

const (
	defaultCurrency = "EUR"
	defaultLanguage = "en"
	defaultPace     = "balanced"
)

type BuildInput struct {
	UserID                     uuid.UUID
	Trip                       *entity.Trip
	WorkspaceID                *uuid.UUID
	Source                     Source
	Request                    RequestOverride
	UserContext                usercontext.UserContext
	WorkspacePolicy            *workspacepolicies.Policy
	GroupPreferences           *GroupPreferences
	GroupAvailability          *GroupAvailability
	PreviousTrips              []entity.Trip
	IncludePreviousTripSignals bool
	IncludeRoute               bool
}

func Build(input BuildInput) PlanningConstraints {
	workspaceID := input.WorkspaceID
	if workspaceID == nil && input.Trip != nil {
		workspaceID = input.Trip.WorkspaceID
	}
	scope := "personal"
	if workspaceID != nil {
		scope = "workspace"
	}

	constraints := PlanningConstraints{
		SchemaVersion: SchemaVersion,
		Language:      language(input),
		Scope:         scope,
		WorkspaceID:   cloneUUIDPtr(workspaceID),
		Source:        input.Source,
		Profile:       profile(input.UserContext.Profile),
		Dates:         dates(input),
		Travelers:     travelers(input),
		Pace:          pace(input),
		Walking:       walking(input),
		Transport:     transport(input),
		TripStyles:    tripStyles(input),
		Accommodation: accommodation(input),
		Interests:     interests(input),
		Avoid:         avoid(input),
		MustHave:      cleanStrings(input.Request.MustHave),
		Accessibility: accessibility(input.Request.Accessibility),
		Food:          food(input),
		Prompt:        prompt(input.Request.Prompt),
		Warnings:      []Issue{},
		Blockers:      []Issue{},
	}
	if b := budget(input); b != nil {
		constraints.Budget = b
	}
	routeSnapshot := route(input)
	if routeSnapshot != nil && input.IncludeRoute {
		constraints.Route = routeSnapshot
	}
	if selections := transportSelections(routeSnapshot); len(selections) > 0 {
		constraints.TransportSelections = selections
	}
	if policy := workspacePolicy(input.WorkspacePolicy); policy != nil {
		constraints.WorkspacePolicy = policy
	}
	if input.GroupPreferences != nil && strings.TrimSpace(input.GroupPreferences.Summary) != "" {
		constraints.GroupPreferences = input.GroupPreferences
	}
	if input.GroupAvailability != nil {
		constraints.GroupAvailability = input.GroupAvailability
	}
	if input.IncludePreviousTripSignals {
		constraints.PreviousTripSignals = previousTripSignals(input.PreviousTrips)
	}
	DetectConflicts(&constraints)
	return constraints
}

func SummaryFor(c PlanningConstraints) Summary {
	budget := "not provided"
	if c.Budget != nil {
		if c.Budget.Amount != nil {
			budget = formatAmount(*c.Budget.Amount) + " " + c.Budget.Currency
		} else if c.Budget.Currency != "" {
			budget = c.Budget.Currency
		}
	}
	transportSummary := "No transport preference"
	if len(c.Transport.PreferredModes) > 0 || len(c.Transport.AvoidModes) > 0 || len(c.Transport.DisallowedModes) > 0 {
		parts := make([]string, 0, 3)
		if len(c.Transport.PreferredModes) > 0 {
			parts = append(parts, "Prefer "+strings.Join(c.Transport.PreferredModes, ", "))
		}
		if len(c.Transport.AvoidModes) > 0 {
			parts = append(parts, "avoid "+strings.Join(c.Transport.AvoidModes, ", "))
		}
		if len(c.Transport.DisallowedModes) > 0 {
			parts = append(parts, "disallow "+strings.Join(c.Transport.DisallowedModes, ", "))
		}
		transportSummary = strings.Join(parts, "; ")
	}
	if len(c.TransportSelections) > 0 {
		transportSummary = strings.TrimSpace(transportSummary)
		if transportSummary == "" || transportSummary == "No transport preference" {
			transportSummary = formatSelectedTransportCount(len(c.TransportSelections))
		} else {
			transportSummary += "; " + formatSelectedTransportCount(len(c.TransportSelections))
		}
	}
	return Summary{
		Language:             languageName(c.Language),
		Budget:               budget,
		Pace:                 c.Pace,
		Transport:            transportSummary,
		TripStyles:           append([]string(nil), c.TripStyles...),
		WorkspacePolicyRules: workspaceRuleCount(c.WorkspacePolicy),
		WarningCount:         len(c.Warnings),
		BlockerCount:         len(c.Blockers),
	}
}

func ToAIContext(c *PlanningConstraints) *AIContext {
	if c == nil {
		return nil
	}
	summary := SummaryFor(*c)
	parts := []string{
		"Output language: " + summary.Language,
		"Budget: " + summary.Budget,
		"Pace: " + c.Pace,
		"Transport: " + summary.Transport,
	}
	if c.Walking.MaxKmPerDay != nil {
		parts = append(parts, "Max walking: "+formatAmount(*c.Walking.MaxKmPerDay)+" km/day")
	}
	if len(c.TripStyles) > 0 {
		parts = append(parts, "Trip styles: "+strings.Join(c.TripStyles, ", "))
	}
	if c.WorkspacePolicy != nil && strings.TrimSpace(c.WorkspacePolicy.Summary) != "" {
		parts = append(parts, "Workspace policy: "+strings.TrimSpace(c.WorkspacePolicy.Summary))
	}
	if c.GroupPreferences != nil && strings.TrimSpace(c.GroupPreferences.Summary) != "" {
		parts = append(parts, "Group preferences: "+strings.TrimSpace(c.GroupPreferences.Summary))
	}
	if c.GroupAvailability != nil && c.GroupAvailability.SelectedDateOption != nil {
		selected := c.GroupAvailability.SelectedDateOption
		parts = append(parts, "Group dates: "+selected.StartDate+" to "+selected.EndDate)
	}
	if len(c.TransportSelections) > 0 {
		parts = append(parts, "Selected transport: "+formatSelectedTransportCount(len(c.TransportSelections))+"; plan around departure and arrival times and do not imply booking is confirmed unless status proves it.")
	}
	return &AIContext{
		PlanningConstraints: c,
		ConstraintSummary:   strings.Join(parts, "\n"),
		Warnings:            append([]Issue(nil), c.Warnings...),
		Blockers:            append([]Issue(nil), c.Blockers...),
	}
}

func language(input BuildInput) string {
	candidates := []string{}
	if input.Request.OutputLanguage != "" {
		candidates = append(candidates, input.Request.OutputLanguage)
	}
	if input.Request.Language != "" {
		candidates = append(candidates, input.Request.Language)
	}
	if input.Trip != nil && input.Trip.CreationMetadata != nil {
		if value, _ := input.Trip.CreationMetadata["outputLanguage"].(string); value != "" {
			candidates = append(candidates, value)
		}
	}
	if input.UserContext.Profile != nil {
		candidates = append(candidates, input.UserContext.Profile.PreferredLanguage)
	}
	candidates = append(candidates, defaultLanguage)
	for _, candidate := range candidates {
		switch normalized := strings.ToLower(strings.TrimSpace(candidate)); normalized {
		case "en", "es", "uk", "fr":
			return normalized
		}
	}
	return defaultLanguage
}

func profile(profile *usercontext.UserProfile) Profile {
	out := Profile{PreferredCurrency: defaultCurrency}
	if profile == nil {
		return out
	}
	out.HomeCity = stringPtrValue(profile.HomeCity)
	out.HomeCountry = stringPtrValue(profile.HomeCountry)
	if currency := normalizeCurrency(profile.PreferredCurrency); currency != "" {
		out.PreferredCurrency = currency
	}
	return out
}

func budget(input BuildInput) *Budget {
	if input.Request.Budget != nil {
		return &Budget{
			Amount:     cloneFloat64Ptr(input.Request.Budget.Amount),
			Currency:   currencyOrDefault(input.Request.Budget.Currency, input.UserContext.Profile),
			Strictness: strictnessOrDefault(input.Request.Budget.Strictness),
		}
	}
	if input.Trip != nil && (input.Trip.BudgetAmount != nil || strings.TrimSpace(input.Trip.BudgetCurrency) != "") {
		return &Budget{
			Amount:     cloneFloat64Ptr(input.Trip.BudgetAmount),
			Currency:   currencyOrDefault(input.Trip.BudgetCurrency, input.UserContext.Profile),
			Strictness: "target",
		}
	}
	return &Budget{Currency: currencyOrDefault("", input.UserContext.Profile), Strictness: "loose"}
}

func dates(input BuildInput) Dates {
	out := Dates{Flexibility: dateFlexibility(input.Request.DateFlexibility)}
	if input.Request.StartDate != "" {
		out.StartDate = strings.TrimSpace(input.Request.StartDate)
	}
	if input.Request.EndDate != "" {
		out.EndDate = strings.TrimSpace(input.Request.EndDate)
	}
	if input.Request.DurationDays != nil {
		out.DurationDays = *input.Request.DurationDays
	}
	if input.Trip != nil {
		if out.StartDate == "" && input.Trip.StartDate != nil {
			out.StartDate = input.Trip.StartDate.Format("2006-01-02")
		}
		if out.DurationDays == 0 {
			out.DurationDays = int(input.Trip.Days)
		}
	}
	if out.EndDate == "" && out.StartDate != "" && out.DurationDays > 0 {
		if parsed, err := time.Parse("2006-01-02", out.StartDate); err == nil {
			out.EndDate = parsed.AddDate(0, 0, out.DurationDays-1).Format("2006-01-02")
		}
	}
	if input.GroupAvailability != nil && input.GroupAvailability.SelectedDateOption != nil {
		selected := input.GroupAvailability.SelectedDateOption
		out.StartDate = selected.StartDate
		out.EndDate = selected.EndDate
		out.DurationDays = selected.DurationDays
		out.Flexibility = "fixed"
	}
	return out
}

func travelers(input BuildInput) Travelers {
	out := Travelers{Count: 1}
	if input.Request.Travelers != nil {
		if input.Request.Travelers.Count != nil && *input.Request.Travelers.Count > 0 {
			out.Count = *input.Request.Travelers.Count
		}
		out.Type = strings.TrimSpace(input.Request.Travelers.Type)
		return out
	}
	if input.Trip != nil && input.Trip.Travelers > 0 {
		out.Count = input.Trip.Travelers
	}
	return out
}

func pace(input BuildInput) string {
	for _, candidate := range []string{input.Request.Pace, tripPace(input.Trip), preferencesPace(input.UserContext.Preferences), defaultPace} {
		switch normalized := strings.ToLower(strings.TrimSpace(candidate)); normalized {
		case "relaxed", "balanced", "packed", "intensive":
			if normalized == "intensive" {
				return "packed"
			}
			return normalized
		}
	}
	return defaultPace
}

func walking(input BuildInput) Walking {
	out := Walking{AllowLongHikes: true}
	if prefs := input.UserContext.Preferences; prefs != nil && prefs.MaxWalkingKmPerDay != nil {
		out.MaxKmPerDay = cloneFloat64Ptr(prefs.MaxWalkingKmPerDay)
		out.AllowLongHikes = *prefs.MaxWalkingKmPerDay > 8
	}
	if policy := input.WorkspacePolicy; policy != nil && policy.Rules.Rules.MaxWalkingKmPerDay.Enabled {
		limit := policy.Rules.Rules.MaxWalkingKmPerDay.Km
		if out.MaxKmPerDay == nil || limit < *out.MaxKmPerDay {
			out.MaxKmPerDay = &limit
		}
		if limit <= 8 {
			out.AllowLongHikes = false
		}
	}
	if input.Request.Walking != nil {
		if input.Request.Walking.MaxKmPerDay != nil {
			out.MaxKmPerDay = cloneFloat64Ptr(input.Request.Walking.MaxKmPerDay)
		}
		if input.Request.Walking.AllowLongHikes != nil {
			out.AllowLongHikes = *input.Request.Walking.AllowLongHikes
		}
	}
	return out
}

func transport(input BuildInput) Transport {
	preferred := cleanModes(input.Request.TransportModesPreferred())
	avoid := cleanModes(input.Request.TransportModesAvoid())
	if len(preferred) == 0 && input.Request.Route == nil && input.Trip != nil && input.Trip.Route != nil {
		preferred = cleanModes(input.Trip.Route.Preferences.PreferredModes)
	}
	if len(avoid) == 0 && input.Request.Route == nil && input.Trip != nil && input.Trip.Route != nil {
		avoid = cleanModes(input.Trip.Route.Preferences.AvoidModes)
	}
	if len(preferred) == 0 && input.UserContext.Preferences != nil {
		preferred = cleanModes(input.UserContext.Preferences.PreferredTransport)
	}
	if len(preferred) == 0 && input.WorkspacePolicy != nil && input.WorkspacePolicy.Rules.Rules.PreferredTransportModes.Enabled {
		preferred = cleanModes(input.WorkspacePolicy.Rules.Rules.PreferredTransportModes.Modes)
	}
	disallowed := []string{}
	if input.WorkspacePolicy != nil && input.WorkspacePolicy.Rules.Rules.DisallowedTransportModes.Enabled {
		disallowed = append(disallowed, input.WorkspacePolicy.Rules.Rules.DisallowedTransportModes.Modes...)
	}
	if input.Request.Transport != nil {
		disallowed = append(disallowed, input.Request.Transport.DisallowedModes...)
	}
	disallowed = cleanModes(disallowed)
	carAvailable := false
	if input.Trip != nil && input.Trip.Route != nil {
		carAvailable = input.Trip.Route.Preferences.CarAvailable
	}
	if input.Request.Route != nil {
		carAvailable = input.Request.Route.Preferences.CarAvailable
	}
	if input.Request.Transport != nil && input.Request.Transport.CarAvailable != nil {
		carAvailable = *input.Request.Transport.CarAvailable
	}
	var maxTransfer *int
	if input.Trip != nil && input.Trip.Route != nil {
		maxTransfer = cloneIntPtr(input.Trip.Route.Preferences.MaxTransferHoursPerDay)
	}
	if input.Request.Route != nil {
		maxTransfer = cloneIntPtr(input.Request.Route.Preferences.MaxTransferHoursPerDay)
	}
	if input.WorkspacePolicy != nil && input.WorkspacePolicy.Rules.Rules.MaxTransferHoursPerDay.Enabled {
		policyMax := int(input.WorkspacePolicy.Rules.Rules.MaxTransferHoursPerDay.Hours)
		maxTransfer = &policyMax
	}
	if input.Request.Transport != nil && input.Request.Transport.MaxTransferHoursPerDay != nil {
		maxTransfer = cloneIntPtr(input.Request.Transport.MaxTransferHoursPerDay)
	}
	allowed := cleanModes(nil)
	if input.Request.Transport != nil && len(input.Request.Transport.AllowedModes) > 0 {
		allowed = cleanModes(input.Request.Transport.AllowedModes)
	} else {
		allowed = allAllowedModes(disallowed)
	}
	return Transport{
		PreferredModes:         preferred,
		AllowedModes:           allowed,
		AvoidModes:             avoid,
		DisallowedModes:        disallowed,
		CarAvailable:           carAvailable,
		MaxTransferHoursPerDay: maxTransfer,
	}
}

func tripStyles(input BuildInput) []string {
	if len(input.Request.TripStyles) > 0 {
		return cleanTokens(input.Request.TripStyles, aggregate.SupportedTripStyles)
	}
	if input.Request.Route != nil && len(input.Request.Route.Preferences.TripStyles) > 0 {
		return cleanTokens(input.Request.Route.Preferences.TripStyles, aggregate.SupportedTripStyles)
	}
	if input.Trip != nil && input.Trip.Route != nil && len(input.Trip.Route.Preferences.TripStyles) > 0 {
		return cleanTokens(input.Trip.Route.Preferences.TripStyles, aggregate.SupportedTripStyles)
	}
	if input.UserContext.Preferences != nil {
		return cleanTokens(input.UserContext.Preferences.TravelStyles, aggregate.SupportedTripStyles)
	}
	return []string{}
}

func accommodation(input BuildInput) Accommodation {
	out := Accommodation{PreferredTypes: []string{}, AvoidTypes: []string{}}
	if input.UserContext.Preferences != nil {
		out.PreferredTypes = cleanTokens(input.UserContext.Preferences.AccommodationStyle, aggregate.SupportedAccommodationHints)
	}
	if input.Trip != nil && input.Trip.Accommodation != nil {
		out.PreferredTypes = appendUnique(out.PreferredTypes, string(input.Trip.Accommodation.Type))
	}
	if input.Request.Accommodation != nil {
		if len(input.Request.Accommodation.PreferredTypes) > 0 {
			out.PreferredTypes = cleanTokens(input.Request.Accommodation.PreferredTypes, aggregate.SupportedAccommodationHints)
		}
		out.AvoidTypes = cleanTokens(input.Request.Accommodation.AvoidTypes, aggregate.SupportedAccommodationHints)
		if input.Request.Accommodation.CampingAllowed != nil {
			out.CampingAllowed = *input.Request.Accommodation.CampingAllowed
		}
	}
	if contains(out.PreferredTypes, "campsite") || contains(out.PreferredTypes, "campervan") || contains(out.PreferredTypes, "cabin") {
		out.CampingAllowed = true
	}
	return out
}

func interests(input BuildInput) []string {
	if len(input.Request.Interests) > 0 {
		return cleanStrings(input.Request.Interests)
	}
	if input.Trip != nil {
		return cleanStrings(input.Trip.Interests)
	}
	return []string{}
}

func avoid(input BuildInput) []string {
	out := cleanStrings(input.Request.Avoid)
	if len(out) == 0 && input.UserContext.Preferences != nil {
		out = cleanStrings(input.UserContext.Preferences.Avoid)
	}
	return out
}

func accessibility(value *Accessibility) Accessibility {
	if value == nil {
		return Accessibility{}
	}
	value.Notes = strings.TrimSpace(value.Notes)
	return *value
}

func food(input BuildInput) Food {
	out := Food{Preferences: []string{}, DietaryRestrictions: []string{}}
	if input.UserContext.Preferences != nil {
		out.Preferences = cleanStrings(input.UserContext.Preferences.FoodPreferences)
		out.DietaryRestrictions = cleanStrings(input.UserContext.Preferences.DietaryRestrictions)
	}
	if input.Request.Food != nil {
		if len(input.Request.Food.Preferences) > 0 {
			out.Preferences = cleanStrings(input.Request.Food.Preferences)
		}
		if len(input.Request.Food.DietaryRestrictions) > 0 {
			out.DietaryRestrictions = cleanStrings(input.Request.Food.DietaryRestrictions)
		}
	}
	return out
}

func route(input BuildInput) *Route {
	source := input.Request.Route
	if source == nil && input.Trip != nil {
		source = input.Trip.Route
	}
	if source == nil {
		return nil
	}
	clean := aggregate.PublicRoute(source)
	if clean == nil {
		return nil
	}
	tripType := input.Request.TripType
	if tripType == "" && input.Trip != nil {
		tripType = input.Trip.TripType
	}
	if tripType == "" && len(clean.Stops) > 1 {
		tripType = entity.TripTypeMultiDestination
	}
	return &Route{
		TripType:       tripType,
		Origin:         clean.Origin,
		Stops:          clean.Stops,
		Legs:           clean.Legs,
		ReturnToOrigin: clean.ReturnToOrigin,
		Preferences:    clean.Preferences,
	}
}

func transportSelections(route *Route) []TransportSelection {
	if route == nil {
		return nil
	}
	out := make([]TransportSelection, 0, len(route.Legs))
	for _, leg := range route.Legs {
		if leg.SelectedTransportOption == nil {
			continue
		}
		option := leg.SelectedTransportOption
		selection := TransportSelection{
			RouteLegID:      leg.ID,
			Mode:            option.Mode,
			Provider:        option.Provider,
			OperatorName:    option.OperatorName,
			ServiceName:     option.ServiceName,
			DepartureDate:   option.DepartureDate,
			DepartureTime:   option.DepartureTime,
			ArrivalDate:     option.ArrivalDate,
			ArrivalTime:     option.ArrivalTime,
			DurationMinutes: option.DurationMinutes,
			Status:          option.Status,
			Confidence:      option.Confidence,
		}
		if option.EstimatedPrice != nil {
			price := *option.EstimatedPrice
			selection.EstimatedPrice = &price
		}
		out = append(out, selection)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func formatSelectedTransportCount(count int) string {
	if count == 1 {
		return "1 selected route leg"
	}
	return fmt.Sprintf("%d selected route legs", count)
}

func workspacePolicy(policy *workspacepolicies.Policy) *WorkspacePolicy {
	if policy == nil {
		return nil
	}
	rules := policy.Rules.Rules
	blocking := []string{}
	warnings := []string{}
	addRule := func(key string, rule workspacepolicies.Rule) {
		if !rule.Enabled {
			return
		}
		if rule.Severity == workspacepolicies.SeverityBlocking {
			blocking = append(blocking, key)
			return
		}
		if rule.Severity == workspacepolicies.SeverityWarning {
			warnings = append(warnings, key)
		}
	}
	addRule("requireTripBudget", rules.RequireTripBudget)
	addRule("maxTripBudget", rules.MaxTripBudget.Rule)
	addRule("maxDailyBudget", rules.MaxDailyBudget.Rule)
	addRule("maxWalkingKmPerDay", rules.MaxWalkingKmPerDay.Rule)
	addRule("preferredTransportModes", rules.PreferredTransportModes.Rule)
	addRule("maxTransferHoursPerDay", rules.MaxTransferHoursPerDay.Rule)
	addRule("disallowedTransportModes", rules.DisallowedTransportModes.Rule)
	raw, _ := json.Marshal(policy.Rules)
	constraints := workspacepolicies.BuildAIConstraints(policy)
	summary := ""
	if constraints != nil {
		summary = constraints.Summary
	}
	return &WorkspacePolicy{
		PolicyID:      policy.ID.String(),
		Summary:       summary,
		BlockingRules: blocking,
		WarningRules:  warnings,
		Rules:         raw,
	}
}

func previousTripSignals(trips []entity.Trip) *PreviousTripSignals {
	if len(trips) == 0 {
		return nil
	}
	if len(trips) > 20 {
		trips = trips[:20]
	}
	destinations := make([]string, 0, len(trips))
	styleCounts := map[string]int{}
	totalDays := 0
	budgetAmounts := map[string][]float64{}
	for _, trip := range trips {
		if destination := strings.TrimSpace(trip.Destination); destination != "" {
			destinations = appendUnique(destinations, destination)
		}
		if trip.Days > 0 {
			totalDays += int(trip.Days)
		}
		for _, interest := range trip.Interests {
			styleCounts[normalizeToken(interest)]++
		}
		if trip.BudgetAmount != nil && strings.TrimSpace(trip.BudgetCurrency) != "" {
			currency := normalizeCurrency(trip.BudgetCurrency)
			budgetAmounts[currency] = append(budgetAmounts[currency], *trip.BudgetAmount)
		}
	}
	styles := make([]string, 0, len(styleCounts))
	for style := range styleCounts {
		if style != "" {
			styles = append(styles, style)
		}
	}
	sort.Slice(styles, func(i, j int) bool { return styleCounts[styles[i]] > styleCounts[styles[j]] })
	if len(styles) > 8 {
		styles = styles[:8]
	}
	typicalDays := 0
	if totalDays > 0 {
		typicalDays = totalDays / len(trips)
	}
	var typicalBudget *Budget
	for currency, amounts := range budgetAmounts {
		if len(amounts) == 0 {
			continue
		}
		total := 0.0
		for _, amount := range amounts {
			total += amount
		}
		avg := total / float64(len(amounts))
		typicalBudget = &Budget{Amount: &avg, Currency: currency, Strictness: "target"}
		break
	}
	return &PreviousTripSignals{
		VisitedDestinations: destinations,
		LikedStyles:         styles,
		TypicalDurationDays: typicalDays,
		TypicalBudget:       typicalBudget,
	}
}

func prompt(value *Prompt) *Prompt {
	if value == nil {
		return nil
	}
	out := *value
	out.UserPrompt = strings.TrimSpace(out.UserPrompt)
	out.RefinementInstruction = strings.TrimSpace(out.RefinementInstruction)
	out.QuickChips = cleanStrings(out.QuickChips)
	if out.UserPrompt == "" && out.RefinementInstruction == "" && len(out.QuickChips) == 0 {
		return nil
	}
	return &out
}

func (r RequestOverride) TransportModesPreferred() []string {
	if r.Transport != nil && len(r.Transport.PreferredModes) > 0 {
		return r.Transport.PreferredModes
	}
	if r.Route != nil {
		return r.Route.Preferences.PreferredModes
	}
	return nil
}

func (r RequestOverride) TransportModesAvoid() []string {
	if r.Transport != nil && len(r.Transport.AvoidModes) > 0 {
		return r.Transport.AvoidModes
	}
	if r.Route != nil {
		return r.Route.Preferences.AvoidModes
	}
	return nil
}

func tripPace(trip *entity.Trip) string {
	if trip == nil {
		return ""
	}
	return trip.Pace
}

func preferencesPace(prefs *usercontext.UserPreferences) string {
	if prefs == nil {
		return ""
	}
	return prefs.Pace
}
