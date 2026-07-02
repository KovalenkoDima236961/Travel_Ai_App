package generator

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
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
