package trip

import (
	"time"

	"github.com/google/uuid"
)

// CreateTripRequest is the JSON body accepted by POST /trips. Validation tags
// are enforced by the project's validation package in the handler.
type CreateTripRequest struct {
	Destination    string   `json:"destination" validate:"required"`
	StartDate      string   `json:"startDate" validate:"omitempty,datetime=2006-01-02"`
	Days           int32    `json:"days" validate:"required,gte=1,lte=30"`
	BudgetAmount   *float64 `json:"budgetAmount" validate:"omitempty,gte=0"`
	BudgetCurrency string   `json:"budgetCurrency" validate:"omitempty,len=3"`
	Travelers      int32    `json:"travelers" validate:"required,gte=1"`
	Interests      []string `json:"interests" validate:"omitempty,dive,required"`
	Pace           string   `json:"pace" validate:"omitempty,oneof=relaxed balanced packed"`
}

func (r CreateTripRequest) toInput() CreateTripInput {
	return CreateTripInput{
		Destination:    r.Destination,
		StartDate:      r.StartDate,
		Days:           r.Days,
		BudgetAmount:   r.BudgetAmount,
		BudgetCurrency: r.BudgetCurrency,
		Travelers:      r.Travelers,
		Interests:      r.Interests,
		Pace:           r.Pace,
	}
}

// CreateTripInput is the validated, service-level representation of a request.
type CreateTripInput struct {
	Destination    string
	StartDate      string
	Days           int32
	BudgetAmount   *float64
	BudgetCurrency string
	Travelers      int32
	Interests      []string
	Pace           string
}

// TripResponse is the JSON representation returned to clients.
type TripResponse struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"userId,omitempty"`
	Destination    string     `json:"destination"`
	StartDate      *string    `json:"startDate,omitempty"`
	Days           int32      `json:"days"`
	BudgetAmount   *float64   `json:"budgetAmount,omitempty"`
	BudgetCurrency string     `json:"budgetCurrency"`
	Travelers      int32      `json:"travelers"`
	Interests      []string   `json:"interests"`
	Pace           string     `json:"pace"`
	Status         Status     `json:"status"`
	Itinerary      any        `json:"itinerary,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// newTripResponse maps a domain Trip to its API representation.
func newTripResponse(t *Trip) TripResponse {
	resp := TripResponse{
		ID:             t.ID,
		UserID:         t.UserID,
		Destination:    t.Destination,
		Days:           t.Days,
		BudgetAmount:   t.BudgetAmount,
		BudgetCurrency: t.BudgetCurrency,
		Travelers:      t.Travelers,
		Interests:      t.Interests,
		Pace:           t.Pace,
		Status:         t.Status,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}

	if t.Interests == nil {
		resp.Interests = []string{}
	}
	if t.StartDate != nil {
		s := t.StartDate.Format("2006-01-02")
		resp.StartDate = &s
	}
	if len(t.Itinerary) > 0 {
		resp.Itinerary = t.Itinerary
	}

	return resp
}
