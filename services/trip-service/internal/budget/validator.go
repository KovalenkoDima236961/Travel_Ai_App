package budget

import (
	"errors"
	"regexp"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

// Cost categories. These mirror the values accepted by the AI Planning Service
// and the web client.
const (
	CategoryFood          = "food"
	CategoryTransport     = "transport"
	CategoryTicket        = "ticket"
	CategoryActivity      = "activity"
	CategoryAccommodation = "accommodation"
	CategoryShopping      = "shopping"
	CategoryOther         = "other"
)

// Cost confidence levels.
const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

// Cost sources. Generated output defaults to SourceAI; manual edits to
// SourceManual.
const (
	SourceAI           = "ai"
	SourceManual       = "manual"
	SourceProvider     = "provider"
	SourceAvailability = "availability"
)

// DefaultCurrency is the local-dev fallback when neither the trip budget nor the
// user's preferred currency is known.
const DefaultCurrency = "EUR"

const maxNoteLength = 300

// currencyPattern matches an uppercase ISO-like 3-letter currency code.
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// categoryOrder is the stable display/serialisation order for byCategory.
var categoryOrder = []string{
	CategoryFood,
	CategoryTransport,
	CategoryTicket,
	CategoryActivity,
	CategoryAccommodation,
	CategoryShopping,
	CategoryOther,
}

var validCategories = toSet(categoryOrder)
var validConfidences = toSet([]string{ConfidenceLow, ConfidenceMedium, ConfidenceHigh})
var validSources = toSet([]string{SourceAI, SourceManual, SourceProvider, SourceAvailability})

// NormalizeEstimatedCost validates and normalizes an item cost estimate in
// place. It returns an error only for the two hard failures the product treats
// as invalid input — a negative amount or a malformed currency code. Softer
// problems are repaired: an unknown category becomes "other", an unknown
// confidence becomes "low", an unknown/empty source becomes defaultSource, and
// an over-length note is truncated.
//
// defaultSource is "ai" when normalizing generated output and "manual" when
// normalizing a user-supplied itinerary edit, so the server backstops clients
// that forget to set it.
func NormalizeEstimatedCost(c *aggregate.EstimatedCost, defaultSource string) error {
	if c == nil {
		return nil
	}

	c.Currency = strings.ToUpper(strings.TrimSpace(c.Currency))
	c.Category = strings.ToLower(strings.TrimSpace(c.Category))
	c.Confidence = strings.ToLower(strings.TrimSpace(c.Confidence))
	c.Source = strings.ToLower(strings.TrimSpace(c.Source))
	c.Note = strings.TrimSpace(c.Note)

	if c.Amount != nil && *c.Amount < 0 {
		return errors.New("amount must be >= 0")
	}
	if c.Currency != "" && !currencyPattern.MatchString(c.Currency) {
		return errors.New("currency must be a 3-letter uppercase code")
	}

	if !validCategories[c.Category] {
		c.Category = CategoryOther
	}
	if !validConfidences[c.Confidence] {
		// A missing confidence with an amount defaults to low; otherwise drop it.
		if c.Amount != nil {
			c.Confidence = ConfidenceLow
		} else {
			c.Confidence = ""
		}
	}
	if !validSources[c.Source] {
		c.Source = defaultSource
	}
	if runes := []rune(c.Note); len(runes) > maxNoteLength {
		c.Note = string(runes[:maxNoteLength])
	}

	return nil
}

// NormalizeBudgetInput validates and normalizes a trip-level budget. A nil
// amount clears the budget (returns nil amount and an empty currency). When an
// amount is present, the currency is upper-cased and, if absent, falls back to
// fallbackCurrency and then DefaultCurrency.
func NormalizeBudgetInput(amount *float64, currency, fallbackCurrency string) (*float64, string, error) {
	if amount == nil {
		return nil, "", nil
	}
	if *amount < 0 {
		return nil, "", errors.New("budget amount must be >= 0")
	}

	normalized := strings.ToUpper(strings.TrimSpace(currency))
	if normalized == "" {
		normalized = strings.ToUpper(strings.TrimSpace(fallbackCurrency))
	}
	if normalized == "" {
		normalized = DefaultCurrency
	}
	if !currencyPattern.MatchString(normalized) {
		return nil, "", errors.New("budget currency must be a 3-letter uppercase code")
	}

	value := *amount
	return &value, normalized, nil
}

func toSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}
