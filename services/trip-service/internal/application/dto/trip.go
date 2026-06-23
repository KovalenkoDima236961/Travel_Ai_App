// Package dto holds application-level (use-case) input/output types. The HTTP
// layer builds these from its transport payloads.
package dto

import "encoding/json"

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
