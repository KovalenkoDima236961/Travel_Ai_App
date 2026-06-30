package response

import (
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// Budget is the structured trip budget object. It is null on a trip when no
// amount is set.
type Budget struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// NewBudget builds the structured budget object from a trip, or nil when the
// trip has no budget amount.
func NewBudget(t *entity.Trip) *Budget {
	if t == nil || t.BudgetAmount == nil {
		return nil
	}
	currency := t.BudgetCurrency
	if currency == "" {
		currency = budget.DefaultCurrency
	}
	return &Budget{Amount: *t.BudgetAmount, Currency: currency}
}

// BudgetEnvelope is the payload returned by PUT /trips/{id}/budget.
type BudgetEnvelope struct {
	Budget *Budget `json:"budget"`
}

// NewBudgetEnvelope wraps the updated trip's budget.
func NewBudgetEnvelope(t *entity.Trip) BudgetEnvelope {
	return BudgetEnvelope{Budget: NewBudget(t)}
}
