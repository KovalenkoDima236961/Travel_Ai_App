package budget

// Summary is the on-demand budget summary computed from a trip's budget and its
// itinerary JSON. Pointer fields are null when the trip has no budget set;
// arrays are always non-nil so they serialise as [] rather than null.
//
// All monetary fields are rounded to two decimal places.
type Summary struct {
	Currency                 string            `json:"currency"`
	TripBudget               *float64          `json:"tripBudget"`
	EstimatedTotal           float64           `json:"estimatedTotal"`
	Remaining                *float64          `json:"remaining"`
	OverBudgetBy             *float64          `json:"overBudgetBy"`
	MissingEstimateCount     int               `json:"missingEstimateCount"`
	EstimatedItemCount       int               `json:"estimatedItemCount"`
	UnsupportedCurrencyCount int               `json:"unsupportedCurrencyCount"`
	ByDay                    []DaySummary      `json:"byDay"`
	ByCategory               []CategorySummary `json:"byCategory"`
}

// DaySummary is the per-day rollup. DailyBudgetShare and OverDailyBudgetBy are
// only populated when the trip has a budget and a positive day count.
type DaySummary struct {
	DayNumber            int      `json:"dayNumber"`
	EstimatedTotal       float64  `json:"estimatedTotal"`
	MissingEstimateCount int      `json:"missingEstimateCount"`
	DailyBudgetShare     *float64 `json:"dailyBudgetShare,omitempty"`
	OverDailyBudgetBy    *float64 `json:"overDailyBudgetBy,omitempty"`
}

// CategorySummary is the per-category rollup, emitted in a fixed category order
// for stable JSON.
type CategorySummary struct {
	Category       string  `json:"category"`
	EstimatedTotal float64 `json:"estimatedTotal"`
	ItemCount      int     `json:"itemCount"`
}
