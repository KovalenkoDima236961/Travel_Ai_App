package response

import (
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
)

type TripArchiveResult struct {
	TripID     uuid.UUID  `json:"tripId"`
	ArchivedAt *time.Time `json:"archivedAt"`
	Lifecycle  string     `json:"lifecycle"`
}

type TripLibraryTrip struct {
	ID               uuid.UUID  `json:"id"`
	Destination      string     `json:"destination"`
	StartDate        *string    `json:"startDate,omitempty"`
	Days             int32      `json:"days"`
	TripType         string     `json:"tripType"`
	WorkspaceID      *uuid.UUID `json:"workspaceId,omitempty"`
	ArchivedAt       *time.Time `json:"archivedAt,omitempty"`
	ArchivedByUserID *uuid.UUID `json:"archivedByUserId,omitempty"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type TripLibraryItem struct {
	Trip       TripLibraryTrip       `json:"trip"`
	Lifecycle  string                `json:"lifecycle"`
	Recap      TripLibraryRecap      `json:"recap"`
	Template   TripLibraryTemplate   `json:"template"`
	Budget     TripLibraryBudget     `json:"budget"`
	Completion TripLibraryCompletion `json:"completion"`
	Route      TripLibraryRoute      `json:"route"`
	Actions    []string              `json:"actions"`
}

type TripLibraryRecap struct {
	HasRecap  bool       `json:"hasRecap"`
	Status    string     `json:"status,omitempty"`
	Href      string     `json:"href,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}
type TripLibraryTemplate struct {
	HasTemplate bool       `json:"hasTemplate"`
	TemplateID  *uuid.UUID `json:"templateId,omitempty"`
}
type TripLibraryMoney struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}
type TripLibraryBudget struct {
	PlannedTotal    *TripLibraryMoney `json:"plannedTotal,omitempty"`
	ActualTotal     *TripLibraryMoney `json:"actualTotal,omitempty"`
	Variance        *TripLibraryMoney `json:"variance,omitempty"`
	MixedCurrencies bool              `json:"mixedCurrencies,omitempty"`
}
type TripLibraryCompletion struct {
	PlannedItemCount int     `json:"plannedItemCount"`
	DoneItemCount    int     `json:"doneItemCount"`
	CompletionRate   float64 `json:"completionRate"`
}
type TripLibraryRoute struct {
	TransportModes []string `json:"transportModes"`
	StopCount      int      `json:"stopCount"`
}
type TripLibraryFilters struct {
	AvailableYears        []int    `json:"availableYears"`
	AvailableDestinations []string `json:"availableDestinations"`
}
type TripLibrarySummary struct {
	Total         int `json:"total"`
	Completed     int `json:"completed"`
	Archived      int `json:"archived"`
	WithRecaps    int `json:"withRecaps"`
	WithTemplates int `json:"withTemplates"`
}
type TripLibrary struct {
	Items      []TripLibraryItem  `json:"items"`
	NextCursor string             `json:"nextCursor,omitempty"`
	Filters    TripLibraryFilters `json:"filters"`
	Summary    TripLibrarySummary `json:"summary"`
}

type TripLibraryInsights struct {
	Summary struct {
		TripCount             int `json:"tripCount"`
		CompletedTripCount    int `json:"completedTripCount"`
		ArchivedTripCount     int `json:"archivedTripCount"`
		TotalTravelDays       int `json:"totalTravelDays"`
		CountriesVisitedCount int `json:"countriesVisitedCount"`
	} `json:"summary"`
	TopDestinations []TripLibraryCount `json:"topDestinations"`
	TopCountries    []TripLibraryCount `json:"topCountries"`
	Budget          struct {
		AverageTripBudget    *TripLibraryMoney `json:"averageTripBudget,omitempty"`
		AverageActualSpend   *TripLibraryMoney `json:"averageActualSpend,omitempty"`
		UnderBudgetTripCount int               `json:"underBudgetTripCount"`
		OverBudgetTripCount  int               `json:"overBudgetTripCount"`
		MixedCurrencies      bool              `json:"mixedCurrencies,omitempty"`
	} `json:"budget"`
	TransportModes []TripLibraryCount `json:"transportModes"`
	TravelStyles   []TripLibraryCount `json:"travelStyles"`
	Recaps         struct {
		TripRecapCount int      `json:"tripRecapCount"`
		CommonLessons  []string `json:"commonLessons"`
	} `json:"recaps"`
	Templates struct {
		TemplatesCreatedFromTrips int `json:"templatesCreatedFromTrips"`
	} `json:"templates"`
	Checklists struct {
		CommonlyMissedItems []TripLibraryCount `json:"commonlyMissedItems"`
	} `json:"checklists"`
}
type TripLibraryCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

func NewTripArchiveResult(value appdto.ArchiveTripResult) TripArchiveResult {
	return TripArchiveResult{TripID: value.TripID, ArchivedAt: value.ArchivedAt, Lifecycle: string(value.Lifecycle)}
}
func NewTripLibrary(value appdto.TripLibraryResult) TripLibrary {
	items := make([]TripLibraryItem, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, newTripLibraryItem(item))
	}
	return TripLibrary{Items: items, NextCursor: value.NextCursor, Filters: TripLibraryFilters{AvailableYears: value.AvailableYears, AvailableDestinations: value.Destinations}, Summary: TripLibrarySummary{Total: value.Total, Completed: value.Completed, Archived: value.Archived, WithRecaps: value.WithRecaps, WithTemplates: value.WithTemplates}}
}
func newTripLibraryItem(value appdto.TripLibraryItem) TripLibraryItem {
	trip := value.Trip
	var startDate *string
	if trip.StartDate != nil {
		formatted := trip.StartDate.Format("2006-01-02")
		startDate = &formatted
	}
	recap := TripLibraryRecap{HasRecap: value.Recap.HasRecap, Status: value.Recap.Status, CreatedAt: value.Recap.CreatedAt}
	if recap.HasRecap {
		recap.Href = "/trips/" + trip.ID.String() + "/recap"
	}
	return TripLibraryItem{Trip: TripLibraryTrip{ID: trip.ID, Destination: trip.Destination, StartDate: startDate, Days: trip.Days, TripType: trip.TripType, WorkspaceID: trip.WorkspaceID, ArchivedAt: trip.ArchivedAt, ArchivedByUserID: trip.ArchivedByUserID, UpdatedAt: trip.UpdatedAt}, Lifecycle: string(value.Lifecycle), Recap: recap, Template: TripLibraryTemplate{HasTemplate: value.Template.HasTemplate, TemplateID: value.Template.TemplateID}, Budget: newTripLibraryBudget(value.Budget), Completion: TripLibraryCompletion{PlannedItemCount: value.Completion.PlannedItemCount, DoneItemCount: value.Completion.DoneItemCount, CompletionRate: value.Completion.CompletionRate}, Route: TripLibraryRoute{TransportModes: value.Route.TransportModes, StopCount: value.Route.StopCount}, Actions: value.Actions}
}
func newTripLibraryBudget(value appdto.TripLibraryBudget) TripLibraryBudget {
	return TripLibraryBudget{PlannedTotal: newTripLibraryMoney(value.PlannedTotal), ActualTotal: newTripLibraryMoney(value.ActualTotal), Variance: newTripLibraryMoney(value.Variance), MixedCurrencies: value.MixedCurrencies}
}
func newTripLibraryMoney(value *appdto.LibraryMoney) *TripLibraryMoney {
	if value == nil {
		return nil
	}
	return &TripLibraryMoney{Amount: value.Amount, Currency: value.Currency}
}
func NewTripLibraryInsights(value appdto.TripLibraryInsights) TripLibraryInsights {
	result := TripLibraryInsights{TopDestinations: newTripLibraryCounts(value.TopDestinations), TopCountries: newTripLibraryCounts(value.TopCountries), TransportModes: newTripLibraryCounts(value.TransportModes), TravelStyles: newTripLibraryCounts(value.TravelStyles)}
	result.Summary.TripCount, result.Summary.CompletedTripCount, result.Summary.ArchivedTripCount, result.Summary.TotalTravelDays, result.Summary.CountriesVisitedCount = value.TripCount, value.CompletedTripCount, value.ArchivedTripCount, value.TotalTravelDays, value.CountriesVisited
	result.Budget.AverageTripBudget, result.Budget.AverageActualSpend, result.Budget.UnderBudgetTripCount, result.Budget.OverBudgetTripCount, result.Budget.MixedCurrencies = newTripLibraryMoney(value.AverageTripBudget), newTripLibraryMoney(value.AverageActualSpend), value.UnderBudgetTrips, value.OverBudgetTrips, value.MixedCurrencies
	result.Recaps.TripRecapCount, result.Recaps.CommonLessons = value.TripRecapCount, value.CommonLessons
	result.Templates.TemplatesCreatedFromTrips = value.TemplatesCreated
	result.Checklists.CommonlyMissedItems = newTripLibraryCounts(value.CommonlyMissed)
	return result
}
func newTripLibraryCounts(values []appdto.TripLibraryCount) []TripLibraryCount {
	result := make([]TripLibraryCount, 0, len(values))
	for _, value := range values {
		result = append(result, TripLibraryCount{Label: value.Label, Count: value.Count})
	}
	return result
}
