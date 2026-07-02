package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// Trip is the JSON representation of a trip returned to clients.
type Trip struct {
	ID                uuid.UUID                `json:"id"`
	UserID            *uuid.UUID               `json:"userId,omitempty"`
	Destination       string                   `json:"destination"`
	StartDate         *string                  `json:"startDate,omitempty"`
	Days              int32                    `json:"days"`
	BudgetAmount      *float64                 `json:"budgetAmount,omitempty"`
	BudgetCurrency    string                   `json:"budgetCurrency"`
	Budget            *Budget                  `json:"budget"`
	Travelers         int32                    `json:"travelers"`
	Interests         []string                 `json:"interests"`
	Pace              string                   `json:"pace"`
	Status            entity.Status            `json:"status"`
	Itinerary         any                      `json:"itinerary,omitempty"`
	Accommodation     *aggregate.Accommodation `json:"accommodation"`
	ItineraryRevision int                      `json:"itineraryRevision"`
	Access            *TripAccess              `json:"access,omitempty"`
	CreatedAt         time.Time                `json:"createdAt"`
	UpdatedAt         time.Time                `json:"updatedAt"`
}

type TripAccess struct {
	Role                   string `json:"role"`
	CanEdit                bool   `json:"canEdit"`
	CanManageCollaborators bool   `json:"canManageCollaborators"`
	CanManageShare         bool   `json:"canManageShare"`
	CanRestoreVersion      bool   `json:"canRestoreVersion"`
	CanDelete              bool   `json:"canDelete"`
}

// ListTrips is the paginated envelope returned by GET /trips. Limit and Offset
// echo the values actually applied (after defaults).
type ListTrips struct {
	Items  []Trip `json:"items"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type SharedTripSummary struct {
	ID                uuid.UUID               `json:"id"`
	Destination       string                  `json:"destination"`
	StartDate         *string                 `json:"startDate,omitempty"`
	Days              int32                   `json:"days"`
	Role              entity.CollaboratorRole `json:"role"`
	OwnerUserID       *uuid.UUID              `json:"ownerUserId,omitempty"`
	Status            entity.Status           `json:"status"`
	ItineraryRevision int                     `json:"itineraryRevision"`
	UpdatedAt         time.Time               `json:"updatedAt"`
}

type TripCollaborator struct {
	ID              uuid.UUID                 `json:"id"`
	TripID          uuid.UUID                 `json:"tripId"`
	UserID          uuid.UUID                 `json:"userId"`
	Email           *string                   `json:"email,omitempty"`
	DisplayName     *string                   `json:"displayName,omitempty"`
	Role            entity.CollaboratorRole   `json:"role"`
	Status          entity.CollaboratorStatus `json:"status"`
	InvitedByUserID uuid.UUID                 `json:"invitedByUserId"`
	InvitedAt       time.Time                 `json:"invitedAt"`
	AcceptedAt      *time.Time                `json:"acceptedAt,omitempty"`
	RemovedAt       *time.Time                `json:"removedAt,omitempty"`
}

type CollaborationInvitation struct {
	CollaboratorID  uuid.UUID               `json:"collaboratorId"`
	TripID          uuid.UUID               `json:"tripId"`
	Destination     string                  `json:"destination"`
	Role            entity.CollaboratorRole `json:"role"`
	InvitedByUserID uuid.UUID               `json:"invitedByUserId"`
	InvitedAt       time.Time               `json:"invitedAt"`
}

// ItineraryVersionSummary is returned by the version-history list endpoint.
type ItineraryVersionSummary struct {
	ID              uuid.UUID                     `json:"id"`
	TripID          uuid.UUID                     `json:"tripId"`
	VersionNumber   int                           `json:"versionNumber"`
	Source          entity.ItineraryVersionSource `json:"source"`
	Metadata        map[string]any                `json:"metadata"`
	CreatedByUserID *uuid.UUID                    `json:"createdByUserId,omitempty"`
	CreatedAt       time.Time                     `json:"createdAt"`
}

// ItineraryVersionDetail includes the snapshot JSON for preview/restore flows.
type ItineraryVersionDetail struct {
	ID              uuid.UUID                     `json:"id"`
	TripID          uuid.UUID                     `json:"tripId"`
	VersionNumber   int                           `json:"versionNumber"`
	Source          entity.ItineraryVersionSource `json:"source"`
	Itinerary       any                           `json:"itinerary"`
	Metadata        map[string]any                `json:"metadata"`
	CreatedByUserID *uuid.UUID                    `json:"createdByUserId,omitempty"`
	CreatedAt       time.Time                     `json:"createdAt"`
}

// ListItineraryVersions is the paginated envelope returned by
// GET /trips/{id}/itinerary/versions.
type ListItineraryVersions struct {
	Items  []ItineraryVersionSummary `json:"items"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

// TripShareInfo is returned only to the authenticated trip owner.
type TripShareInfo struct {
	ShareToken       string     `json:"shareToken,omitempty"`
	ShareURL         string     `json:"shareUrl,omitempty"`
	Enabled          bool       `json:"enabled"`
	CreatedAt        *time.Time `json:"createdAt,omitempty"`
	UpdatedAt        *time.Time `json:"updatedAt,omitempty"`
	DisabledAt       *time.Time `json:"disabledAt,omitempty"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
	Expired          bool       `json:"expired"`
	PasswordRequired bool       `json:"passwordRequired"`
}

type PublicShareStatus struct {
	Available        bool `json:"available"`
	PasswordRequired bool `json:"passwordRequired"`
	Expired          bool `json:"expired,omitempty"`
}

type PublicShareUnlockResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type AccommodationEnvelope struct {
	Accommodation *aggregate.Accommodation `json:"accommodation"`
}

// PublicTrip is the sanitized read-only payload for public share links. The
// private trip budget (both the flat fields and the itinerary's totalBudget) is
// intentionally omitted; item-level estimated costs within the itinerary remain
// visible because they are part of the shared plan.
type PublicTrip struct {
	Destination string        `json:"destination"`
	StartDate   *string       `json:"startDate,omitempty"`
	Days        int32         `json:"days"`
	Travelers   int32         `json:"travelers,omitempty"`
	Interests   []string      `json:"interests"`
	Pace        string        `json:"pace,omitempty"`
	Status      entity.Status `json:"status"`
	Itinerary   any           `json:"itinerary,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	SharedAt    time.Time     `json:"sharedAt"`
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

func NewSharedTrips(shared []entity.SharedTrip) []SharedTripSummary {
	items := make([]SharedTripSummary, 0, len(shared))
	for i := range shared {
		items = append(items, NewSharedTripSummary(&shared[i]))
	}
	return items
}

func NewSharedTripSummary(shared *entity.SharedTrip) SharedTripSummary {
	resp := SharedTripSummary{
		ID:                shared.Trip.ID,
		Destination:       shared.Trip.Destination,
		Days:              shared.Trip.Days,
		Role:              shared.Collaborator.Role,
		OwnerUserID:       shared.Trip.UserID,
		Status:            shared.Trip.Status,
		ItineraryRevision: shared.Trip.ItineraryRevision,
		UpdatedAt:         shared.Trip.UpdatedAt,
	}
	if shared.Trip.StartDate != nil {
		s := shared.Trip.StartDate.Format("2006-01-02")
		resp.StartDate = &s
	}
	return resp
}

func NewTripCollaborator(info appdto.TripCollaboratorInfo) TripCollaborator {
	c := info.Collaborator
	return TripCollaborator{
		ID:              c.ID,
		TripID:          c.TripID,
		UserID:          c.UserID,
		Email:           info.Email,
		DisplayName:     info.DisplayName,
		Role:            c.Role,
		Status:          c.Status,
		InvitedByUserID: c.InvitedByUserID,
		InvitedAt:       c.InvitedAt,
		AcceptedAt:      c.AcceptedAt,
		RemovedAt:       c.RemovedAt,
	}
}

func NewTripCollaborators(infos []appdto.TripCollaboratorInfo) []TripCollaborator {
	items := make([]TripCollaborator, 0, len(infos))
	for _, info := range infos {
		items = append(items, NewTripCollaborator(info))
	}
	return items
}

func NewCollaborationInvitations(invitations []appdto.CollaborationInvitation) []CollaborationInvitation {
	items := make([]CollaborationInvitation, 0, len(invitations))
	for _, invitation := range invitations {
		items = append(items, CollaborationInvitation{
			CollaboratorID:  invitation.CollaboratorID,
			TripID:          invitation.TripID,
			Destination:     invitation.Destination,
			Role:            invitation.Role,
			InvitedByUserID: invitation.InvitedByUserID,
			InvitedAt:       invitation.InvitedAt,
		})
	}
	return items
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
		ID:              v.ID,
		TripID:          v.TripID,
		VersionNumber:   v.VersionNumber,
		Source:          v.Source,
		Metadata:        metadataOrEmpty(v.Metadata),
		CreatedByUserID: v.CreatedByUserID,
		CreatedAt:       v.CreatedAt,
	}
}

// NewItineraryVersionDetail maps one version to its preview representation.
func NewItineraryVersionDetail(v *entity.ItineraryVersion) ItineraryVersionDetail {
	return ItineraryVersionDetail{
		ID:              v.ID,
		TripID:          v.TripID,
		VersionNumber:   v.VersionNumber,
		Source:          v.Source,
		Itinerary:       v.Itinerary,
		Metadata:        metadataOrEmpty(v.Metadata),
		CreatedByUserID: v.CreatedByUserID,
		CreatedAt:       v.CreatedAt,
	}
}

// NewTripShareInfo maps owner-only share status to JSON.
func NewTripShareInfo(info appdto.TripShareInfo) TripShareInfo {
	return TripShareInfo{
		ShareToken:       info.ShareToken,
		ShareURL:         info.ShareURL,
		Enabled:          info.Enabled,
		CreatedAt:        info.CreatedAt,
		UpdatedAt:        info.UpdatedAt,
		DisabledAt:       info.DisabledAt,
		ExpiresAt:        info.ExpiresAt,
		Expired:          info.Expired,
		PasswordRequired: info.PasswordRequired,
	}
}

func NewPublicShareStatus(status appdto.PublicShareStatus) PublicShareStatus {
	return PublicShareStatus{
		Available:        status.Available,
		PasswordRequired: status.PasswordRequired,
		Expired:          status.Expired,
	}
}

func NewPublicShareUnlockResponse(unlock appdto.PublicShareUnlockResponse) PublicShareUnlockResponse {
	return PublicShareUnlockResponse{
		AccessToken: unlock.AccessToken,
		ExpiresAt:   unlock.ExpiresAt,
	}
}

func NewAccommodationEnvelope(accommodation *aggregate.Accommodation) AccommodationEnvelope {
	return AccommodationEnvelope{Accommodation: accommodation}
}

// NewPublicTrip maps a domain Trip to its public, read-only JSON payload. The
// private trip budget is omitted from both the trip fields and the embedded
// itinerary.
func NewPublicTrip(t *entity.Trip, sharedAt time.Time) PublicTrip {
	resp := PublicTrip{
		Destination: t.Destination,
		Days:        t.Days,
		Travelers:   t.Travelers,
		Interests:   t.Interests,
		Pace:        t.Pace,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		SharedAt:    sharedAt,
	}
	if t.Interests == nil {
		resp.Interests = []string{}
	}
	if t.StartDate != nil {
		s := t.StartDate.Format("2006-01-02")
		resp.StartDate = &s
	}
	if len(t.Itinerary) > 0 {
		resp.Itinerary = sanitizePublicItinerary(t.Itinerary)
	}
	return resp
}

// sanitizePublicItinerary strips the trip-level budget (totalBudget) from the
// shared itinerary so the owner's private budget is not exposed publicly, while
// preserving item-level estimated costs. Provider debug/review metadata such as
// priceEnrichment is removed. If the payload cannot be parsed it is dropped
// entirely to fail closed.
func sanitizePublicItinerary(raw json.RawMessage) any {
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil
	}
	delete(generic, "totalBudget")
	stripPublicItineraryMetadata(generic)
	return generic
}

func stripPublicItineraryMetadata(value any) {
	switch typed := value.(type) {
	case map[string]any:
		delete(typed, "priceEnrichment")
		for _, child := range typed {
			stripPublicItineraryMetadata(child)
		}
	case []any:
		for _, child := range typed {
			stripPublicItineraryMetadata(child)
		}
	}
}

// NewTrip maps a domain Trip to its API representation.
func NewTrip(t *entity.Trip) Trip {
	resp := Trip{
		ID:                t.ID,
		UserID:            t.UserID,
		Destination:       t.Destination,
		Days:              t.Days,
		BudgetAmount:      t.BudgetAmount,
		BudgetCurrency:    t.BudgetCurrency,
		Budget:            NewBudget(t),
		Travelers:         t.Travelers,
		Interests:         t.Interests,
		Pace:              t.Pace,
		Status:            t.Status,
		ItineraryRevision: t.ItineraryRevision,
		Accommodation:     t.Accommodation,
		CreatedAt:         t.CreatedAt,
		UpdatedAt:         t.UpdatedAt,
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

func NewTripWithAccess(t *entity.Trip, access interface {
	CanEdit() bool
	CanManageCollaborators() bool
	CanManageShare() bool
	CanRestoreVersion() bool
	CanDelete() bool
}) Trip {
	resp := NewTrip(t)
	role := "viewer"
	if access.CanManageCollaborators() {
		role = "owner"
	} else if access.CanEdit() {
		role = "editor"
	}
	resp.Access = &TripAccess{
		Role:                   role,
		CanEdit:                access.CanEdit(),
		CanManageCollaborators: access.CanManageCollaborators(),
		CanManageShare:         access.CanManageShare(),
		CanRestoreVersion:      access.CanRestoreVersion(),
		CanDelete:              access.CanDelete(),
	}
	return resp
}

func metadataOrEmpty(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}
