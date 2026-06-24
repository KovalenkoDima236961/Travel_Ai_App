package dto

import (
	"encoding/json"
	"time"
)

// CreateTripInput is the validated, application-level representation of a create
// request.
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

// UpdateItineraryInput is the application-level payload for replacing a trip's
// itinerary JSON.
type UpdateItineraryInput struct {
	Itinerary json.RawMessage
}

// RegenerateItineraryPartInput is the application-level payload for partial AI
// regeneration. Instruction is optional and normalized by the service.
type RegenerateItineraryPartInput struct {
	Instruction string
}

// TripShareInfo is the application-level share-link status returned to the
// owning user. Disabled shares intentionally omit the token/URL.
type TripShareInfo struct {
	ShareToken string
	ShareURL   string
	Enabled    bool
	CreatedAt  *time.Time
	DisabledAt *time.Time
}
