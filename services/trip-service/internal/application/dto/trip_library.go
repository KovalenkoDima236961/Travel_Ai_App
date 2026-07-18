package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type TripLibrarySort string

const (
	TripLibrarySortRecentlyUpdated    TripLibrarySort = "recently_updated"
	TripLibrarySortTripDateDesc       TripLibrarySort = "trip_date_desc"
	TripLibrarySortTripDateAsc        TripLibrarySort = "trip_date_asc"
	TripLibrarySortDestination        TripLibrarySort = "destination"
	TripLibrarySortBudgetDesc         TripLibrarySort = "budget_desc"
	TripLibrarySortBudgetAsc          TripLibrarySort = "budget_asc"
	TripLibrarySortCompletionRateDesc TripLibrarySort = "completion_rate_desc"
	TripLibrarySortRecapCreatedDesc   TripLibrarySort = "recap_created_desc"
)

type TripLibraryFilters struct {
	Query         string
	Lifecycle     string
	WorkspaceID   *uuid.UUID
	Year          *int
	Destination   string
	Country       string
	TripType      string
	TravelStyle   string
	TransportMode string
	BudgetMin     *float64
	BudgetMax     *float64
	Currency      string
	HasRecap      *bool
	HasTemplate   *bool
	HasExpenses   *bool
	Archived      *bool
	Sort          TripLibrarySort
	Limit         int
	Cursor        string
}

type ArchiveTripInput struct {
	Reason string
}

type ArchiveTripResult struct {
	TripID     uuid.UUID
	ArchivedAt *time.Time
	Lifecycle  entity.TripLifecycle
}

type LibraryMoney struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// TripLibrarySummary is intentionally compact. It contains no raw receipt,
// comment, calendar, share, provider, or AI payloads.
type TripLibrarySummary struct {
	RecapStatus    *entity.TripRecapStatus
	RecapCreatedAt *time.Time
	TemplateID     *uuid.UUID
	ExpenseTotals  []LibraryMoney
	HasExpenses    bool
	PlannedCount   int
	DoneCount      int
	MissedItems    []string
	Lessons        []string
}

type TripLibraryItem struct {
	Trip               *entity.Trip
	Lifecycle          entity.TripLifecycle
	Recap              TripLibraryRecap
	Template           TripLibraryTemplate
	Budget             TripLibraryBudget
	Completion         TripLibraryCompletion
	Route              TripLibraryRoute
	Actions            []string
	HasExpenses        bool
	InsightLessons     []string
	InsightMissedItems []string
}

type TripLibraryRecap struct {
	HasRecap  bool
	Status    string
	CreatedAt *time.Time
}

type TripLibraryTemplate struct {
	HasTemplate bool
	TemplateID  *uuid.UUID
}

type TripLibraryBudget struct {
	PlannedTotal    *LibraryMoney
	ActualTotal     *LibraryMoney
	Variance        *LibraryMoney
	MixedCurrencies bool
}

type TripLibraryCompletion struct {
	PlannedItemCount int
	DoneItemCount    int
	CompletionRate   float64
}

type TripLibraryRoute struct {
	TransportModes []string
	StopCount      int
}

type TripLibraryResult struct {
	Items          []TripLibraryItem
	NextCursor     string
	AvailableYears []int
	Destinations   []string
	Total          int
	Completed      int
	Archived       int
	WithRecaps     int
	WithTemplates  int
}

type TripLibraryInsights struct {
	TripCount          int
	CompletedTripCount int
	ArchivedTripCount  int
	TotalTravelDays    int
	CountriesVisited   int
	TopDestinations    []TripLibraryCount
	TopCountries       []TripLibraryCount
	AverageTripBudget  *LibraryMoney
	AverageActualSpend *LibraryMoney
	MixedCurrencies    bool
	UnderBudgetTrips   int
	OverBudgetTrips    int
	TransportModes     []TripLibraryCount
	TravelStyles       []TripLibraryCount
	TripRecapCount     int
	CommonLessons      []string
	TemplatesCreated   int
	CommonlyMissed     []TripLibraryCount
}

type TripLibraryCount struct {
	Label string
	Count int
}
