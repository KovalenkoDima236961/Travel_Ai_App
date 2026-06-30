package entity

import "time"

// ExchangeRateTable is the canonical latest-rate table returned by exchange
// rate providers. Rates are quoted from Base into each key currency.
type ExchangeRateTable struct {
	Provider     string             `json:"provider"`
	Base         string             `json:"base"`
	Rates        map[string]float64 `json:"rates"`
	AsOf         time.Time          `json:"asOf"`
	FallbackUsed bool               `json:"fallbackUsed"`
}

// CurrencyConversionResult is the canonical point conversion response.
type CurrencyConversionResult struct {
	Provider        string    `json:"provider"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Amount          float64   `json:"amount"`
	ConvertedAmount float64   `json:"convertedAmount"`
	Rate            float64   `json:"rate"`
	AsOf            time.Time `json:"asOf"`
	FallbackUsed    bool      `json:"fallbackUsed"`
}
