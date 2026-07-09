package generator

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/templateadaptation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triprepair"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
)

// MockItineraryGenerator produces a deterministic, interest-aware sample plan
// locally.
type MockItineraryGenerator struct {
	logger *zap.Logger
}

// NewMockItineraryGenerator constructs the mock generator.
func NewMockItineraryGenerator(logger *zap.Logger) *MockItineraryGenerator {
	return &MockItineraryGenerator{logger: logger}
}

// Generate builds a sample itinerary derived from the trip's destination,
// interests, pace and duration.
func (g *MockItineraryGenerator) Generate(_ context.Context, input application.GenerateItineraryInput) (*aggregate.Itinerary, error) {
	trip := input.Trip
	g.logger.Info("generating mock itinerary",
		zap.String("trip_id", trip.ID.String()),
		zap.String("destination", trip.Destination),
		zap.Int32("days", trip.Days),
		zap.Bool("user_context_loaded", input.UserProfile != nil || input.UserPreferences != nil),
	)

	interests := trip.Interests
	if len(interests) == 0 {
		interests = []string{"sightseeing"}
	}

	currency := mockCurrency(input)

	days := make([]aggregate.ItineraryDay, 0, trip.Days)
	for i := int32(0); i < trip.Days; i++ {
		focus := interests[int(i)%len(interests)]
		days = append(days, aggregate.ItineraryDay{
			Day:   int(i) + 1,
			Title: fmt.Sprintf("Day %d in %s — %s", i+1, trip.Destination, titleCase(focus)),
			Items: []aggregate.ItineraryItem{
				{
					// Intentionally left without an estimate to exercise the
					// missing-estimate path in the budget summary.
					Time: "09:00",
					Type: "activity",
					Name: fmt.Sprintf("Explore %s highlights", trip.Destination),
					Note: weatherAwareNote(
						fmt.Sprintf("focused on %s", focus),
						weatherForDay(input.WeatherForecast, int(i)+1),
					),
				},
				{
					Time:          "11:00",
					Type:          "viewpoint",
					Name:          "Free scenic viewpoint",
					EstimatedCost: mockCost(0, "other", currency, "high", "Free to visit"),
				},
				{
					Time:          "13:00",
					Type:          "meal",
					Name:          "Lunch at a local spot",
					EstimatedCost: mockCost(15, "food", currency, "medium", "Approximate lunch price"),
				},
				{
					Time: "15:00",
					Type: "ticket",
					Name: fmt.Sprintf("%s city museum", titleCase(trip.Destination)),
					Note: "Provider price enrichment can fill this likely ticketed stop.",
				},
				{
					Time:          "17:30",
					Type:          "transport",
					Name:          "Day transit pass",
					EstimatedCost: mockCost(8, "transport", currency, "low", "Public transport estimate"),
				},
				{
					Time:          "19:30",
					Type:          "meal",
					Name:          "Dinner recommendation",
					EstimatedCost: mockCost(24, "food", currency, "low", "Mid-range dinner estimate"),
				},
			},
		})
	}

	summary := fmt.Sprintf("A %d-day %s trip to %s for %d traveler(s).",
		trip.Days, trip.Pace, trip.Destination, trip.Travelers)
	if input.UserProfile != nil || input.UserPreferences != nil {
		summary += " Personalized with saved traveler context."
	}

	return &aggregate.Itinerary{
		Destination: trip.Destination,
		Summary:     summary,
		Travelers:   trip.Travelers,
		Pace:        trip.Pace,
		Currency:    trip.BudgetCurrency,
		TotalBudget: trip.BudgetAmount,
		Days:        days,
		GeneratedAt: time.Now().UTC(),
		Source:      "mock-local-generator",
	}, nil
}

// RegenerateDay returns a deterministic replacement for a single existing day.
func (g *MockItineraryGenerator) RegenerateDay(_ context.Context, input application.RegenerateDayInput) (*aggregate.ItineraryDay, error) {
	if input.DayNumber < 1 {
		return nil, fmt.Errorf("day number must be at least 1")
	}

	trip := input.Trip
	g.logger.Info("regenerating mock itinerary day",
		zap.String("trip_id", trip.ID.String()),
		zap.Int("day_number", input.DayNumber),
		zap.Bool("instruction_present", input.Instruction != ""),
	)

	focus := "refreshed local highlights"
	if input.Instruction != "" {
		focus = "instruction-aware local highlights"
	}
	weatherDay := weatherForDay(input.WeatherForecast, input.DayNumber)

	return &aggregate.ItineraryDay{
		Day:   input.DayNumber,
		Title: fmt.Sprintf("Regenerated day %d in %s", input.DayNumber, trip.Destination),
		Items: []aggregate.ItineraryItem{
			{
				Time: "10:00",
				Type: "activity",
				Name: fmt.Sprintf("Updated %s experience", trip.Destination),
				Note: weatherAwareNote(
					fmt.Sprintf("A deterministic mock replacement focused on %s.", focus),
					weatherDay,
				),
			},
			{
				Time:          "13:00",
				Type:          "food",
				Name:          "Updated local lunch",
				Note:          "Keeps the partial regeneration flow predictable in mock mode.",
				EstimatedCost: mockCost(15, "food", mockTripCurrency(trip), "medium", "Approximate lunch price"),
			},
			{
				Time: "16:00",
				Type: "rest",
				Name: "Flexible neighborhood break",
				Note: "Leaves room to adjust the rest of the itinerary manually.",
			},
		},
	}, nil
}

// RegenerateItem returns a deterministic replacement item for the selected
// zero-based item index.
func (g *MockItineraryGenerator) RegenerateItem(_ context.Context, input application.RegenerateItemInput) (*aggregate.ItineraryItem, error) {
	if input.DayNumber < 1 {
		return nil, fmt.Errorf("day number must be at least 1")
	}
	if input.ItemIndex < 0 {
		return nil, fmt.Errorf("item index must be >= 0")
	}

	trip := input.Trip
	g.logger.Info("regenerating mock itinerary item",
		zap.String("trip_id", trip.ID.String()),
		zap.Int("day_number", input.DayNumber),
		zap.Int("item_index", input.ItemIndex),
		zap.Bool("instruction_present", input.Instruction != ""),
	)

	weatherDay := weatherForDay(input.WeatherForecast, input.DayNumber)

	return &aggregate.ItineraryItem{
		Time:          "12:30",
		Type:          "food",
		Name:          fmt.Sprintf("Updated local option %d", input.ItemIndex+1),
		Note:          weatherAwareNote(fmt.Sprintf("Mock replacement for day %d in %s.", input.DayNumber, trip.Destination), weatherDay),
		EstimatedCost: mockCost(12, "food", mockTripCurrency(trip), "medium", "Approximate meal price"),
	}, nil
}

// OptimizeBudgetDay returns a deterministic reviewable cheaper-day proposal.
func (g *MockItineraryGenerator) OptimizeBudgetDay(_ context.Context, input budgetoptimization.OptimizeDayInput) (*budgetoptimization.ProposalContent, error) {
	if input.DayNumber < 1 {
		return nil, fmt.Errorf("day number must be at least 1")
	}

	trip := input.Trip
	g.logger.Info("optimizing mock itinerary day budget",
		zap.String("trip_id", trip.ID.String()),
		zap.Int("day_number", input.DayNumber),
		zap.Bool("instruction_present", input.Instruction != ""),
	)

	proposed := input.CurrentDay
	oldIndex := mostExpensiveItemIndex(proposed.Items)
	if oldIndex < 0 {
		oldIndex = 0
	}
	oldItem := proposed.Items[oldIndex]
	currency := input.BudgetContext.Currency
	if currency == "" {
		currency = mockTripCurrency(trip)
	}

	cheapAmount := 8.0
	if oldItem.EstimatedCost != nil && oldItem.EstimatedCost.Amount != nil && *oldItem.EstimatedCost.Amount > 0 {
		cheapAmount = maxFloat(0, *oldItem.EstimatedCost.Amount-35)
	}
	proposed.Items[oldIndex] = aggregate.ItineraryItem{
		Time: oldItem.Time,
		Type: "activity",
		Name: fmt.Sprintf("Budget-friendly %s alternative", trip.Destination),
		Note: "Mock budget optimization keeps the day theme while lowering estimated cost.",
		EstimatedCost: &aggregate.EstimatedCost{
			Amount:     &cheapAmount,
			Currency:   currency,
			Category:   "activity",
			Confidence: "medium",
			Source:     "ai",
			Note:       "Approximate budget alternative estimate",
		},
	}

	baseTotal := input.BudgetContext.DayEstimatedTotal
	proposedTotal := mockDayTotal(proposed, currency)
	savings := maxFloat(1, baseTotal-proposedTotal)
	if proposedTotal <= 0 || proposedTotal >= baseTotal {
		proposedTotal = maxFloat(0, baseTotal-savings)
	}

	return &budgetoptimization.ProposalContent{
		Summary:                   fmt.Sprintf("Reduced estimated Day %d cost by about %.0f %s with one cheaper alternative.", input.DayNumber, savings, currency),
		Scope:                     budgetoptimization.ScopeDay,
		DayNumber:                 input.DayNumber,
		Currency:                  currency,
		BaseDayEstimatedTotal:     baseTotal,
		ProposedDayEstimatedTotal: proposedTotal,
		EstimatedSavingsAmount:    savings,
		Confidence:                budgetoptimization.ConfidenceMedium,
		Changes: []budgetoptimization.ProposalChange{{
			Type:                   budgetoptimization.ChangeReplaceItem,
			OldItemIndex:           intPtr(oldIndex),
			OldItemName:            oldItem.Name,
			NewItemName:            proposed.Items[oldIndex].Name,
			Reason:                 "Replaces the most expensive stop with a lower-cost option.",
			EstimatedSavingsAmount: &savings,
			Currency:               currency,
		}},
		PreservedItems: []budgetoptimization.PreservedItem{{
			ItemIndex: 0,
			ItemName:  proposed.Items[0].Name,
			Reason:    "Preserved to keep the basic day structure.",
		}},
		Tradeoffs:   []string{"The replacement is less premium but keeps the day practical."},
		Warnings:    []string{"Savings and prices are approximate estimates for review."},
		ProposedDay: proposed,
	}, nil
}

// AdaptTemplate produces a deterministic template adaptation: it renames items
// with a "<destination> version of ..." prefix, preserves the day/item
// structure, and trims/extends the plan to the target duration. It mirrors the
// AI Planning Service mock adapter so mock mode stays predictable in tests.
func (g *MockItineraryGenerator) AdaptTemplate(_ context.Context, input templateadaptation.AdaptInput) (*templateadaptation.AdaptResult, error) {
	target := input.Target
	if target.DurationDays < 1 {
		return nil, fmt.Errorf("target duration must be at least 1")
	}
	destination := strings.TrimSpace(target.Destination)
	currency := defaultCurrency
	if target.Budget != nil && strings.TrimSpace(target.Budget.Currency) != "" {
		currency = strings.ToUpper(strings.TrimSpace(target.Budget.Currency))
	}

	sourceDays := append([]templateadaptation.TemplateDay(nil), input.Template.Days...)
	sortTemplateDays(sourceDays)

	days := make([]aggregate.ItineraryDay, 0, target.DurationDays)
	usedSource := len(sourceDays)
	if usedSource > target.DurationDays {
		usedSource = target.DurationDays
	}
	for i := 0; i < usedSource; i++ {
		days = append(days, mockAdaptDay(sourceDays[i], i, destination, currency))
	}
	for i := usedSource; i < target.DurationDays; i++ {
		days = append(days, aggregate.ItineraryDay{
			Day:   i + 1,
			Title: fmt.Sprintf("Flexible exploration day in %s", destination),
			Items: []aggregate.ItineraryItem{{
				Time: "10:00",
				Type: "activity",
				Name: fmt.Sprintf("Flexible exploration in %s", destination),
				Note: "Added to extend the template to the requested duration; customize freely.",
			}},
		})
	}

	summary := templateadaptation.Summary{
		SourceDurationDays: input.Template.DurationDays,
		TargetDurationDays: target.DurationDays,
		PreservedStructure: input.Constraints.PreserveStructure,
		ChangedDestination: true,
		MajorChanges:       []string{fmt.Sprintf("Adapted the template structure to %s.", destination)},
		Warnings: []string{
			"Estimated prices are approximate and should be verified.",
			"Availability and opening hours must be checked before booking.",
		},
	}
	if target.DurationDays < len(sourceDays) {
		summary.MajorChanges = append(summary.MajorChanges,
			fmt.Sprintf("Trimmed the plan from %d to %d day(s).", len(sourceDays), target.DurationDays))
	} else if target.DurationDays > len(sourceDays) {
		summary.MajorChanges = append(summary.MajorChanges,
			fmt.Sprintf("Extended the plan from %d to %d day(s).", len(sourceDays), target.DurationDays))
	}

	return &templateadaptation.AdaptResult{
		Itinerary: aggregate.Itinerary{
			Destination: destination,
			Summary:     fmt.Sprintf("%s trip adapted from template", destination),
			Travelers:   int32(target.Travelers),
			Pace:        target.Pace,
			Currency:    currency,
			Days:        days,
			GeneratedAt: time.Now().UTC(),
			Source:      "ai_template_adaptation",
		},
		Summary: summary,
	}, nil
}

func (g *MockItineraryGenerator) RepairItinerary(_ context.Context, input triprepair.Input) (*triprepair.ProposalContent, error) {
	trip := input.Trip
	g.logger.Info("repairing mock itinerary",
		zap.String("trip_id", trip.ID.String()),
		zap.String("repair_mode", string(input.Constraints.RepairMode)),
	)

	repaired := input.CurrentItinerary
	currency := mockTripCurrency(trip)
	beforeTotal := itineraryTotal(repaired, currency)
	changes := make([]triprepair.Change, 0)
	maxChanges := 10
	if input.Constraints.Constraints.MaxChangedItems != nil {
		maxChanges = *input.Constraints.Constraints.MaxChangedItems
	}

	changed := false
	if input.Constraints.RepairMode == triprepair.RepairModeAddRestTime {
		changed = addMockRestBlocks(&repaired, currency, &changes, maxChanges)
	} else {
		changed = reduceMostExpensiveMockItem(&repaired, trip.Destination, currency, &changes)
	}
	if !changed {
		addMockRestBlocks(&repaired, currency, &changes, maxChanges)
	}
	afterTotal := itineraryTotal(repaired, currency)

	changedCount := 0
	addedCount := 0
	movedCount := 0
	for _, change := range changes {
		switch change.Type {
		case "item_added":
			addedCount++
		case "item_moved":
			movedCount++
		default:
			changedCount++
		}
	}

	issues := make([]string, 0, len(input.Issues))
	for _, issue := range input.Issues {
		issues = append(issues, issue.Type)
	}
	content := &triprepair.ProposalContent{
		RepairedItinerary: repaired,
		RepairSummary: triprepair.Summary{
			RepairMode:          input.Constraints.RepairMode,
			ChangedItemCount:    changedCount,
			AddedItemCount:      addedCount,
			MovedItemCount:      movedCount,
			EstimatedCostBefore: &triprepair.Money{Amount: beforeTotal, Currency: currency},
			EstimatedCostAfter:  &triprepair.Money{Amount: afterTotal, Currency: currency},
			MajorChanges:        majorChangeReasons(changes),
			IssuesAddressed:     issues,
			IssuesRemaining:     []string{"availability_unchecked"},
			Warnings: []string{
				"Availability must be checked again after repair.",
				"Cost estimates are approximate.",
			},
		},
		Changes: changes,
		Validation: triprepair.Validation{
			Valid:    true,
			Warnings: []string{"Review the repaired itinerary before applying."},
		},
	}
	content.Diff = triprepair.BuildDiff(input.CurrentItinerary, repaired)
	return content, nil
}

func mockAdaptDay(source templateadaptation.TemplateDay, index int, destination, currency string) aggregate.ItineraryDay {
	items := make([]aggregate.ItineraryItem, 0, len(source.Items))
	for _, item := range source.Items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		items = append(items, mockAdaptItem(item, destination, currency))
	}
	if len(items) == 0 {
		items = append(items, aggregate.ItineraryItem{
			Time: "10:00",
			Type: "activity",
			Name: fmt.Sprintf("Flexible exploration in %s", destination),
			Note: "Customize this day for the destination.",
		})
	}
	title := strings.TrimSpace(source.Title)
	if title == "" {
		title = fmt.Sprintf("Day %d in %s", index+1, destination)
	} else if !strings.Contains(strings.ToLower(title), strings.ToLower(destination)) {
		title = fmt.Sprintf("%s (%s)", title, destination)
	}
	return aggregate.ItineraryDay{Day: index + 1, Title: title, Items: items}
}

func mockAdaptItem(item templateadaptation.TemplateItem, destination, currency string) aggregate.ItineraryItem {
	timeValue := strings.TrimSpace(item.Time)
	if timeValue == "" {
		timeValue = strings.TrimSpace(item.StartTime)
		if end := strings.TrimSpace(item.EndTime); end != "" {
			timeValue = strings.TrimSpace(timeValue + " - " + end)
		}
	}
	note := strings.TrimSpace(item.Notes)
	if note == "" {
		note = strings.TrimSpace(item.Description)
	}
	if note == "" {
		note = fmt.Sprintf("Adapted from the template for %s; verify details.", destination)
	}
	var cost *aggregate.EstimatedCost
	if item.EstimatedCost != nil && item.EstimatedCost.Amount != nil {
		c := *item.EstimatedCost
		if strings.TrimSpace(c.Currency) == "" {
			c.Currency = currency
		}
		c.Source = "ai"
		c.Note = "Estimated from the template; verify current price."
		cost = &c
	}
	var place *aggregate.PlaceRef
	if item.Place != nil && strings.TrimSpace(item.Place.Name) != "" {
		place = &aggregate.PlaceRef{
			Name:     strings.TrimSpace(destination + " " + item.Place.Name),
			Category: strings.TrimSpace(item.Place.Category),
		}
	}
	return aggregate.ItineraryItem{
		Time:          timeValue,
		Type:          strings.TrimSpace(item.Type),
		Name:          fmt.Sprintf("%s version of %s", destination, strings.TrimSpace(item.Name)),
		Note:          note,
		EstimatedCost: cost,
		Place:         place,
	}
}

func sortTemplateDays(days []templateadaptation.TemplateDay) {
	for i := 1; i < len(days); i++ {
		for j := i; j > 0 && days[j-1].DayOffset > days[j].DayOffset; j-- {
			days[j-1], days[j] = days[j], days[j-1]
		}
	}
}

func weatherForDay(forecast *weathercontext.WeatherForecast, dayNumber int) *weathercontext.WeatherDay {
	if forecast == nil || dayNumber < 1 || dayNumber > len(forecast.Days) {
		return nil
	}
	return &forecast.Days[dayNumber-1]
}

func weatherAwareNote(base string, weatherDay *weathercontext.WeatherDay) string {
	if weatherDay == nil {
		return base
	}
	if weatherDay.PrecipitationChance >= 60 {
		return base + " Rain is likely, so include an indoor backup such as a museum or cafe."
	}
	if weatherDay.TemperatureMaxC >= 32 {
		return base + " High heat expected; avoid long outdoor walks at midday."
	}
	return base
}

func mostExpensiveItemIndex(items []aggregate.ItineraryItem) int {
	index := -1
	maxAmount := -1.0
	for i := range items {
		cost := items[i].EstimatedCost
		if cost == nil || cost.Amount == nil {
			continue
		}
		if *cost.Amount > maxAmount {
			maxAmount = *cost.Amount
			index = i
		}
	}
	return index
}

func mockDayTotal(day aggregate.ItineraryDay, currency string) float64 {
	total := 0.0
	for _, item := range day.Items {
		if item.EstimatedCost == nil || item.EstimatedCost.Amount == nil {
			continue
		}
		itemCurrency := strings.ToUpper(strings.TrimSpace(item.EstimatedCost.Currency))
		if itemCurrency == "" {
			itemCurrency = currency
		}
		if itemCurrency == currency {
			total += *item.EstimatedCost.Amount
		}
	}
	return total
}

func itineraryTotal(itinerary aggregate.Itinerary, currency string) float64 {
	total := 0.0
	for _, day := range itinerary.Days {
		total += mockDayTotal(day, currency)
	}
	return total
}

func reduceMostExpensiveMockItem(
	itinerary *aggregate.Itinerary,
	destination string,
	currency string,
	changes *[]triprepair.Change,
) bool {
	bestDay := -1
	bestItem := -1
	bestAmount := -1.0
	for dayIndex := range itinerary.Days {
		for itemIndex := range itinerary.Days[dayIndex].Items {
			cost := itinerary.Days[dayIndex].Items[itemIndex].EstimatedCost
			if cost == nil || cost.Amount == nil || *cost.Amount <= bestAmount {
				continue
			}
			bestDay = dayIndex
			bestItem = itemIndex
			bestAmount = *cost.Amount
		}
	}
	if bestDay < 0 || bestItem < 0 {
		return false
	}
	item := &itinerary.Days[bestDay].Items[bestItem]
	before := compactMockRepairItem(*item)
	newAmount := maxFloat(0, bestAmount-35)
	item.Type = "activity"
	item.Name = fmt.Sprintf("Budget-friendly %s alternative", destination)
	item.Note = "Mock AI repair lowers estimated cost while keeping the itinerary reviewable."
	item.EstimatedCost = mockCost(newAmount, "activity", currency, "medium", "Approximate repaired estimate")
	dayNumber := itinerary.Days[bestDay].Day
	itemIndex := bestItem
	*changes = append(*changes, triprepair.Change{
		Type:      "item_replaced",
		DayNumber: &dayNumber,
		ItemIndex: &itemIndex,
		Before:    before,
		After:     compactMockRepairItem(*item),
		Reason:    "Reduce budget and policy risk with a lower-cost alternative.",
	})
	return true
}

func addMockRestBlocks(
	itinerary *aggregate.Itinerary,
	currency string,
	changes *[]triprepair.Change,
	maxChanges int,
) bool {
	added := false
	for dayIndex := range itinerary.Days {
		if len(*changes) >= maxChanges {
			return added
		}
		if dayHasRest(itinerary.Days[dayIndex]) {
			continue
		}
		rest := aggregate.ItineraryItem{
			Time:          "15:00",
			Type:          "rest",
			Name:          "AI repair rest break",
			Note:          "Adds downtime for review before approval.",
			EstimatedCost: mockCost(0, "other", currency, "high", "Free rest block"),
		}
		itinerary.Days[dayIndex].Items = append(itinerary.Days[dayIndex].Items, rest)
		dayNumber := itinerary.Days[dayIndex].Day
		itemIndex := len(itinerary.Days[dayIndex].Items) - 1
		*changes = append(*changes, triprepair.Change{
			Type:      "item_added",
			DayNumber: &dayNumber,
			ItemIndex: &itemIndex,
			After:     compactMockRepairItem(rest),
			Reason:    "Add rest time.",
		})
		added = true
	}
	return added
}

func dayHasRest(day aggregate.ItineraryDay) bool {
	for _, item := range day.Items {
		normalized := strings.ToLower(strings.TrimSpace(item.Type))
		if normalized == "rest" || normalized == "break" || normalized == "free_time" {
			return true
		}
	}
	return false
}

func compactMockRepairItem(item aggregate.ItineraryItem) map[string]any {
	return map[string]any{
		"time":          item.Time,
		"type":          item.Type,
		"name":          item.Name,
		"estimatedCost": item.EstimatedCost,
	}
}

func majorChangeReasons(changes []triprepair.Change) []string {
	out := make([]string, 0, len(changes))
	for _, change := range changes {
		if strings.TrimSpace(change.Reason) != "" {
			out = append(out, change.Reason)
		}
	}
	return out
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func intPtr(v int) *int {
	return &v
}

// titleCase upper-cases the first rune of s.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// mockCurrency resolves the currency for sample estimates: trip budget currency,
// then the user's preferred currency, then the package default.
func mockCurrency(input application.GenerateItineraryInput) string {
	if c := strings.TrimSpace(input.Trip.BudgetCurrency); c != "" {
		return strings.ToUpper(c)
	}
	if input.UserProfile != nil {
		if c := strings.TrimSpace(input.UserProfile.PreferredCurrency); c != "" {
			return strings.ToUpper(c)
		}
	}
	return defaultCurrency
}

// mockTripCurrency resolves the currency for regeneration sample estimates from
// the trip alone (regeneration inputs do not carry budget currency separately).
func mockTripCurrency(trip entity.Trip) string {
	if c := strings.TrimSpace(trip.BudgetCurrency); c != "" {
		return strings.ToUpper(c)
	}
	return defaultCurrency
}

// mockCost builds a structured sample estimate with source "ai".
func mockCost(amount float64, category, currency, confidence, note string) *aggregate.EstimatedCost {
	value := amount
	return &aggregate.EstimatedCost{
		Amount:     &value,
		Currency:   currency,
		Category:   category,
		Confidence: confidence,
		Source:     "ai",
		Note:       note,
	}
}
