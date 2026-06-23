// Package response holds the outbound HTTP payloads for the trip endpoints and
// their mapping from domain entities.
package response

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// Trip is the JSON representation of a trip returned to clients.
type Trip struct {
	ID             uuid.UUID     `json:"id"`
	UserID         *uuid.UUID    `json:"userId,omitempty"`
	Destination    string        `json:"destination"`
	StartDate      *string       `json:"startDate,omitempty"`
	Days           int32         `json:"days"`
	BudgetAmount   *float64      `json:"budgetAmount,omitempty"`
	BudgetCurrency string        `json:"budgetCurrency"`
	Travelers      int32         `json:"travelers"`
	Interests      []string      `json:"interests"`
	Pace           string        `json:"pace"`
	Status         entity.Status `json:"status"`
	Itinerary      any           `json:"itinerary,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

// ListTrips is the paginated envelope returned by GET /trips. Limit and Offset
// echo the values actually applied (after defaults).
type ListTrips struct {
	Items  []Trip `json:"items"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// ItineraryVersionSummary is returned by the version-history list endpoint.
type ItineraryVersionSummary struct {
	ID            uuid.UUID                     `json:"id"`
	TripID        uuid.UUID                     `json:"tripId"`
	VersionNumber int                           `json:"versionNumber"`
	Source        entity.ItineraryVersionSource `json:"source"`
	Metadata      map[string]any                `json:"metadata"`
	CreatedAt     time.Time                     `json:"createdAt"`
}

// ItineraryVersionDetail includes the snapshot JSON for preview/restore flows.
type ItineraryVersionDetail struct {
	ID            uuid.UUID                     `json:"id"`
	TripID        uuid.UUID                     `json:"tripId"`
	VersionNumber int                           `json:"versionNumber"`
	Source        entity.ItineraryVersionSource `json:"source"`
	Itinerary     any                           `json:"itinerary"`
	Metadata      map[string]any                `json:"metadata"`
	CreatedAt     time.Time                     `json:"createdAt"`
}

// ListItineraryVersions is the paginated envelope returned by
// GET /trips/{id}/itinerary/versions.
type ListItineraryVersions struct {
	Items  []ItineraryVersionSummary `json:"items"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

// NewListTrips maps a page of domain trips to the API envelope. Items is always
// a (possibly empty) slice so it serialises as [] rather than null.
func NewListTrips(trips []entity.Trip, limit, offset int) ListTrips {
	items := make([]Trip, 0, len(trips))
	for i := range trips {
		items = append(items, NewTrip(&trips[i]))
	}
	return ListTrips{Items: items, Limit: limit, Offset: offset}
}

// NewListItineraryVersions maps version entities to the list response without
// including the full itinerary payload.
func NewListItineraryVersions(versions []entity.ItineraryVersion, limit, offset int) ListItineraryVersions {
	items := make([]ItineraryVersionSummary, 0, len(versions))
	for i := range versions {
		items = append(items, NewItineraryVersionSummary(&versions[i]))
	}
	return ListItineraryVersions{Items: items, Limit: limit, Offset: offset}
}

// NewItineraryVersionSummary maps one version to its list representation.
func NewItineraryVersionSummary(v *entity.ItineraryVersion) ItineraryVersionSummary {
	return ItineraryVersionSummary{
		ID:            v.ID,
		TripID:        v.TripID,
		VersionNumber: v.VersionNumber,
		Source:        v.Source,
		Metadata:      metadataOrEmpty(v.Metadata),
		CreatedAt:     v.CreatedAt,
	}
}

// NewItineraryVersionDetail maps one version to its preview representation.
func NewItineraryVersionDetail(v *entity.ItineraryVersion) ItineraryVersionDetail {
	return ItineraryVersionDetail{
		ID:            v.ID,
		TripID:        v.TripID,
		VersionNumber: v.VersionNumber,
		Source:        v.Source,
		Itinerary:     v.Itinerary,
		Metadata:      metadataOrEmpty(v.Metadata),
		CreatedAt:     v.CreatedAt,
	}
}

// NewTrip maps a domain Trip to its API representation.
func NewTrip(t *entity.Trip) Trip {
	resp := Trip{
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

func metadataOrEmpty(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}
