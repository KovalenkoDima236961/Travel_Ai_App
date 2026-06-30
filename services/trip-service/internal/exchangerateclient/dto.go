package exchangerateclient

import (
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
)

type ExchangeRateTable struct {
	Provider     string             `json:"provider"`
	Base         string             `json:"base"`
	Rates        map[string]float64 `json:"rates"`
	AsOf         time.Time          `json:"asOf"`
	FallbackUsed bool               `json:"fallbackUsed"`
}

type CurrencyConversionResult = budget.CurrencyConversionResult
