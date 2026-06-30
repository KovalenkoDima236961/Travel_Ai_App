package budget

import "time"

// Summary is the on-demand budget summary computed from a trip's budget and its
// itinerary JSON. Pointer fields are null when the trip has no budget set;
// arrays are always non-nil so they serialise as [] rather than null.
//
// All monetary fields are rounded to two decimal places.
type Summary struct {
	Currency                 string                  `json:"currency"`
	TripBudget               *float64                `json:"tripBudget"`
	EstimatedTotal           float64                 `json:"estimatedTotal"`
	AccommodationTotal       *float64                `json:"accommodationTotal,omitempty"`
	Remaining                *float64                `json:"remaining"`
	OverBudgetBy             *float64                `json:"overBudgetBy"`
	MissingEstimateCount     int                     `json:"missingEstimateCount"`
	EstimatedItemCount       int                     `json:"estimatedItemCount"`
	ConvertedItemCount       int                     `json:"convertedItemCount"`
	UnconvertedItemCount     int                     `json:"unconvertedItemCount"`
	UnsupportedCurrencyCount int                     `json:"unsupportedCurrencyCount"`
	OriginalCurrencyTotals   []OriginalCurrencyTotal `json:"originalCurrencyTotals"`
	ConversionWarnings       []ConversionWarning     `json:"conversionWarnings"`
	ExchangeRateInfo         *ExchangeRateInfo       `json:"exchangeRateInfo,omitempty"`
	ByDay                    []DaySummary            `json:"byDay"`
	ByCategory               []CategorySummary       `json:"byCategory"`
}

// DaySummary is the per-day rollup. DailyBudgetShare and OverDailyBudgetBy are
// only populated when the trip has a budget and a positive day count.
type DaySummary struct {
	DayNumber              int                     `json:"dayNumber"`
	EstimatedTotal         float64                 `json:"estimatedTotal"`
	MissingEstimateCount   int                     `json:"missingEstimateCount"`
	OriginalCurrencyTotals []OriginalCurrencyTotal `json:"originalCurrencyTotals"`
	DailyBudgetShare       *float64                `json:"dailyBudgetShare,omitempty"`
	OverDailyBudgetBy      *float64                `json:"overDailyBudgetBy,omitempty"`
}

// CategorySummary is the per-category rollup, emitted in a fixed category order
// for stable JSON.
type CategorySummary struct {
	Category       string  `json:"category"`
	EstimatedTotal float64 `json:"estimatedTotal"`
	ItemCount      int     `json:"itemCount"`
}

type OriginalCurrencyTotal struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

type ConversionWarning struct {
	Currency string   `json:"currency"`
	Amount   *float64 `json:"amount,omitempty"`
	Reason   string   `json:"reason"`
}

type ExchangeRateInfo struct {
	Provider     string    `json:"provider,omitempty"`
	AsOf         time.Time `json:"asOf,omitempty"`
	FallbackUsed bool      `json:"fallbackUsed"`
}

type CurrencyConversionResult struct {
	Provider        string
	From            string
	To              string
	Amount          float64
	ConvertedAmount float64
	Rate            float64
	AsOf            time.Time
	FallbackUsed    bool
}
