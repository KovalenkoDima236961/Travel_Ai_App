package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type tripTemplateJSON struct {
	SchemaVersion int                    `json:"schemaVersion"`
	DurationDays  int32                  `json:"durationDays"`
	Days          []templateDay          `json:"days"`
	Summary       templateSummary        `json:"summary"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type templateDay struct {
	DayOffset int            `json:"dayOffset"`
	Title     string         `json:"title"`
	Items     []templateItem `json:"items"`
}

type templateItem struct {
	TemplateItemID string                   `json:"templateItemId"`
	Name           string                   `json:"name"`
	Type           string                   `json:"type"`
	Description    string                   `json:"description,omitempty"`
	Time           string                   `json:"time,omitempty"`
	StartTime      string                   `json:"startTime,omitempty"`
	EndTime        string                   `json:"endTime,omitempty"`
	DurationMin    *int                     `json:"durationMinutes,omitempty"`
	Place          *templatePlace           `json:"place,omitempty"`
	EstimatedCost  *aggregate.EstimatedCost `json:"estimatedCost,omitempty"`
	Notes          string                   `json:"notes,omitempty"`
}

type templatePlace struct {
	Name            string   `json:"name"`
	Category        string   `json:"category,omitempty"`
	Address         *string  `json:"address"`
	Lat             *float64 `json:"lat"`
	Lng             *float64 `json:"lng"`
	Provider        *string  `json:"provider"`
	ProviderPlaceID *string  `json:"providerPlaceId"`
}

type templateSummary struct {
	EstimatedTotalAmount *float64 `json:"estimatedTotalAmount,omitempty"`
	Currency             string   `json:"currency,omitempty"`
}

func buildTemplateJSON(
	trip entity.Trip,
	in appdto.SaveTripAsTemplateInput,
) (json.RawMessage, int32, *float64, *string, error) {
	if len(trip.Itinerary) == 0 || strings.EqualFold(strings.TrimSpace(string(trip.Itinerary)), "null") {
		return nil, 0, nil, nil, apperrs.NewInvalidInput("source trip must have an itinerary")
	}
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(trip.Itinerary, &itinerary); err != nil {
		return nil, 0, nil, nil, apperrs.NewInvalidInput("source trip itinerary is invalid")
	}
	if err := validateCurrentItinerary(itinerary); err != nil {
		return nil, 0, nil, nil, apperrs.NewInvalidInput("source trip itinerary is invalid")
	}

	durationDays := trip.Days
	if durationDays <= 0 {
		durationDays = int32(maxItineraryDayNumber(itinerary.Days))
	}
	if durationDays <= 0 {
		return nil, 0, nil, nil, apperrs.NewInvalidInput("source trip duration is invalid")
	}

	currency := strings.ToUpper(strings.TrimSpace(itinerary.Currency))
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(trip.BudgetCurrency))
	}
	if in.DefaultCurrency != nil {
		currency = *in.DefaultCurrency
	}
	if currency == "" {
		currency = defaultCurrency
	}

	days := make([]templateDay, 0, len(itinerary.Days))
	for dayIndex, day := range itinerary.Days {
		templateDay := templateDay{
			DayOffset: max(0, day.Day-1),
			Title:     strings.TrimSpace(day.Title),
			Items:     make([]templateItem, 0, len(day.Items)),
		}
		for itemIndex, item := range day.Items {
			templateDay.Items = append(templateDay.Items, sanitizeTemplateItem(item, currency, dayIndex, itemIndex))
		}
		days = append(days, templateDay)
	}
	sort.SliceStable(days, func(i, j int) bool { return days[i].DayOffset < days[j].DayOffset })

	estimatedAmount, estimatedCurrency := estimateTemplateTotal(itinerary, int(durationDays), currency)
	payload := tripTemplateJSON{
		SchemaVersion: 1,
		DurationDays:  durationDays,
		Days:          days,
		Summary: templateSummary{
			EstimatedTotalAmount: estimatedAmount,
			Currency:             stringPtrValue(estimatedCurrency),
		},
		Metadata: map[string]interface{}{
			"createdFromTripId":    trip.ID.String(),
			"createdFromTripTitle": trip.Destination,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, nil, nil, fmt.Errorf("marshal trip template: %w", err)
	}
	return raw, durationDays, estimatedAmount, estimatedCurrency, nil
}

func sanitizeTemplateItem(item aggregate.ItineraryItem, currency string, dayIndex, itemIndex int) templateItem {
	cost := sanitizeTemplateCost(item.EstimatedCost, currency)
	startTime, endTime := splitTimeRange(item.Time)
	return templateItem{
		TemplateItemID: stableTemplateItemID(dayIndex, itemIndex, item.Name),
		Name:           strings.TrimSpace(item.Name),
		Type:           strings.TrimSpace(item.Type),
		Time:           strings.TrimSpace(item.Time),
		StartTime:      startTime,
		EndTime:        endTime,
		Place:          sanitizeTemplatePlace(item.Place),
		EstimatedCost:  cost,
		Notes:          strings.TrimSpace(item.Note),
	}
}

func sanitizeTemplateCost(cost *aggregate.EstimatedCost, fallbackCurrency string) *aggregate.EstimatedCost {
	if cost == nil {
		return nil
	}
	out := &aggregate.EstimatedCost{
		Amount:     cost.Amount,
		Currency:   strings.ToUpper(strings.TrimSpace(cost.Currency)),
		Category:   strings.TrimSpace(cost.Category),
		Confidence: strings.TrimSpace(cost.Confidence),
		Source:     budget.SourceManual,
		Note:       strings.TrimSpace(cost.Note),
	}
	if out.Currency == "" {
		out.Currency = fallbackCurrency
	}
	if strings.EqualFold(cost.Source, budget.SourceProvider) || strings.EqualFold(cost.Source, budget.SourceAvailability) {
		out.Note = "Copied from template; verify current price."
	}
	if err := budget.NormalizeEstimatedCost(out, budget.SourceManual); err != nil {
		return nil
	}
	return out
}

func sanitizeTemplatePlace(place *aggregate.PlaceRef) *templatePlace {
	if place == nil {
		return nil
	}
	name := strings.TrimSpace(place.Name)
	if name == "" {
		return nil
	}
	return &templatePlace{
		Name:            name,
		Category:        strings.TrimSpace(place.Category),
		Address:         templateStringPtr(place.Address),
		Lat:             place.Latitude,
		Lng:             place.Longitude,
		Provider:        templateStringPtr(place.Provider),
		ProviderPlaceID: templateStringPtr(place.ProviderPlaceID),
	}
}

func estimateTemplateTotal(itinerary aggregate.Itinerary, durationDays int, currency string) (*float64, *string) {
	summary := budget.CalculateBudgetSummary(
		budget.TripBudget{
			Currency: currency,
			Days:     durationDays,
		},
		itinerary,
	)
	if summary.EstimatedItemCount == 0 {
		return nil, nil
	}
	amount := summary.EstimatedTotal
	c := summary.Currency
	return &amount, &c
}

func instantiateTemplateItinerary(
	template *entity.TripTemplate,
	in appdto.CreateTripFromTemplateInput,
	generatedAt time.Time,
) (aggregate.Itinerary, error) {
	var payload tripTemplateJSON
	if err := json.Unmarshal(template.TemplateJSON, &payload); err != nil {
		return aggregate.Itinerary{}, apperrs.NewInvalidInput("template content is invalid")
	}
	if payload.SchemaVersion != 1 {
		return aggregate.Itinerary{}, apperrs.NewInvalidInput("template schema version is not supported")
	}
	if len(payload.Days) == 0 {
		return aggregate.Itinerary{}, apperrs.NewInvalidInput("template has no itinerary days")
	}
	sort.SliceStable(payload.Days, func(i, j int) bool { return payload.Days[i].DayOffset < payload.Days[j].DayOffset })

	currency := in.BudgetCurrency
	if currency == "" && template.DefaultCurrency != nil {
		currency = *template.DefaultCurrency
	}
	if currency == "" {
		currency = payload.Summary.Currency
	}
	if currency == "" {
		currency = defaultCurrency
	}
	itinerary := aggregate.Itinerary{
		Destination: in.Destination,
		Summary:     fmt.Sprintf("Created from template: %s", template.Title),
		Travelers:   *in.Travelers,
		Pace:        in.Pace,
		Currency:    currency,
		Days:        make([]aggregate.ItineraryDay, 0, len(payload.Days)),
		GeneratedAt: generatedAt,
		Source:      "template",
	}
	for _, day := range payload.Days {
		dayNumber := day.DayOffset + 1
		if dayNumber < 1 {
			dayNumber = len(itinerary.Days) + 1
		}
		nextDay := aggregate.ItineraryDay{
			Day:   dayNumber,
			Title: strings.TrimSpace(day.Title),
			Items: make([]aggregate.ItineraryItem, 0, len(day.Items)),
		}
		for _, item := range day.Items {
			nextDay.Items = append(nextDay.Items, instantiateTemplateItem(item, currency))
		}
		itinerary.Days = append(itinerary.Days, nextDay)
	}
	return itinerary, nil
}

func instantiateTemplateItem(item templateItem, currency string) aggregate.ItineraryItem {
	timeValue := strings.TrimSpace(item.Time)
	if timeValue == "" {
		timeValue = strings.TrimSpace(item.StartTime)
		if item.EndTime != "" {
			timeValue += " - " + strings.TrimSpace(item.EndTime)
		}
	}
	note := strings.TrimSpace(item.Notes)
	if note == "" {
		note = strings.TrimSpace(item.Description)
	}
	return aggregate.ItineraryItem{
		Time:          timeValue,
		Type:          strings.TrimSpace(item.Type),
		Name:          strings.TrimSpace(item.Name),
		Note:          note,
		EstimatedCost: instantiateTemplateCost(item.EstimatedCost, currency),
		Place:         instantiateTemplatePlace(item.Place),
	}
}

func instantiateTemplateCost(cost *aggregate.EstimatedCost, fallbackCurrency string) *aggregate.EstimatedCost {
	if cost == nil {
		return nil
	}
	out := &aggregate.EstimatedCost{
		Amount:     cost.Amount,
		Currency:   strings.ToUpper(strings.TrimSpace(cost.Currency)),
		Category:   strings.TrimSpace(cost.Category),
		Confidence: strings.TrimSpace(cost.Confidence),
		Source:     budget.SourceManual,
		Note:       strings.TrimSpace(cost.Note),
	}
	if out.Currency == "" {
		out.Currency = fallbackCurrency
	}
	if out.Note == "" {
		out.Note = "Copied from template; verify current price."
	}
	if err := budget.NormalizeEstimatedCost(out, budget.SourceManual); err != nil {
		return nil
	}
	return out
}

func instantiateTemplatePlace(place *templatePlace) *aggregate.PlaceRef {
	if place == nil || strings.TrimSpace(place.Name) == "" {
		return nil
	}
	return &aggregate.PlaceRef{
		Provider:        stringPtrValue(place.Provider),
		ProviderPlaceID: stringPtrValue(place.ProviderPlaceID),
		Name:            strings.TrimSpace(place.Name),
		Address:         stringPtrValue(place.Address),
		Latitude:        place.Lat,
		Longitude:       place.Lng,
		Category:        strings.TrimSpace(place.Category),
	}
}

func maxItineraryDayNumber(days []aggregate.ItineraryDay) int {
	maxDay := 0
	for _, day := range days {
		if day.Day > maxDay {
			maxDay = day.Day
		}
	}
	return maxDay
}

func stableTemplateItemID(dayIndex, itemIndex int, name string) string {
	namespace := uuid.MustParse("4e89c728-06f9-42b5-95bb-7c1f498a9c0f")
	return uuid.NewSHA1(namespace, []byte(fmt.Sprintf("%d:%d:%s", dayIndex, itemIndex, strings.ToLower(strings.TrimSpace(name))))).String()
}

func splitTimeRange(raw string) (string, string) {
	parts := strings.Split(strings.TrimSpace(raw), "-")
	if len(parts) < 2 {
		return strings.TrimSpace(raw), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
