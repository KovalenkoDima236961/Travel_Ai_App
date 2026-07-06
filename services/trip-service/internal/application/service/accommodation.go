package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

var validAccommodationTypes = map[aggregate.AccommodationType]bool{
	aggregate.AccommodationTypeHotel:      true,
	aggregate.AccommodationTypeHostel:     true,
	aggregate.AccommodationTypeApartment:  true,
	aggregate.AccommodationTypeGuesthouse: true,
	aggregate.AccommodationTypeHome:       true,
	aggregate.AccommodationTypeOther:      true,
}

// GetTripAccommodation returns the private structured accommodation for a trip.
// Any accepted owner/editor/viewer can read it; public shares never reach this
// method because there is no public route for accommodation.
func (s *Service) GetTripAccommodation(ctx context.Context, tripID uuid.UUID) (*aggregate.Accommodation, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	return trip.Accommodation, nil
}

// UpdateTripAccommodation validates and stores one structured stay location.
// Owner/editor can update it; the itinerary revision is intentionally unchanged.
func (s *Service) UpdateTripAccommodation(ctx context.Context, tripID uuid.UUID, in appdto.UpdateTripAccommodationInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}

	accommodation, err := normalizeAccommodation(in.Accommodation, current.BudgetCurrency)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateTripAccommodation(ctx, tripID, ownerID, accommodation)
	if err != nil {
		return nil, err
	}

	eventType := activity.EventAccommodationUpdated
	if current.Accommodation == nil {
		eventType = activity.EventAccommodationAdded
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   eventType,
		EntityType:  activityEntityType(activity.EntityAccommodation),
		EntityID:    activityEntityID(tripID),
		Metadata:    accommodationActivityMetadata(accommodation),
	})

	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Accommodation changed")

	return updated, nil
}

// DeleteTripAccommodation clears the structured stay location. It does not
// mutate itinerary_revision.
func (s *Service) DeleteTripAccommodation(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.ClearTripAccommodation(ctx, tripID, ownerID)
	if err != nil {
		return nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventAccommodationRemoved,
		EntityType:  activityEntityType(activity.EntityAccommodation),
		EntityID:    activityEntityID(tripID),
		Metadata:    accommodationActivityMetadata(current.Accommodation),
	})

	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Accommodation removed")

	return updated, nil
}

func normalizeAccommodation(in *aggregate.Accommodation, fallbackCurrency string) (*aggregate.Accommodation, error) {
	if in == nil {
		return nil, apperrs.NewInvalidInput("accommodation is required")
	}

	accommodation := *in
	accommodation.Name = strings.TrimSpace(accommodation.Name)
	if accommodation.Name == "" {
		return nil, apperrs.NewInvalidInput("accommodation.name is required")
	}
	if len([]rune(accommodation.Name)) > maxAccommodationNameLength {
		return nil, apperrs.NewInvalidInput("accommodation.name must be at most %d characters", maxAccommodationNameLength)
	}

	accommodation.Type = aggregate.AccommodationType(strings.ToLower(strings.TrimSpace(string(accommodation.Type))))
	if accommodation.Type == "" {
		accommodation.Type = aggregate.AccommodationTypeOther
	}
	if !validAccommodationTypes[accommodation.Type] {
		return nil, apperrs.NewInvalidInput("accommodation.type must be one of hotel, hostel, apartment, guesthouse, home, other")
	}

	accommodation.Address = strings.TrimSpace(accommodation.Address)
	if len([]rune(accommodation.Address)) > maxAccommodationAddress {
		return nil, apperrs.NewInvalidInput("accommodation.address must be at most %d characters", maxAccommodationAddress)
	}

	accommodation.Notes = strings.TrimSpace(accommodation.Notes)
	if len([]rune(accommodation.Notes)) > maxAccommodationNotes {
		return nil, apperrs.NewInvalidInput("accommodation.notes must be at most %d characters", maxAccommodationNotes)
	}

	accommodation.CheckInDate = strings.TrimSpace(accommodation.CheckInDate)
	accommodation.CheckOutDate = strings.TrimSpace(accommodation.CheckOutDate)
	checkIn, err := parseAccommodationDate(accommodation.CheckInDate, "accommodation.checkInDate")
	if err != nil {
		return nil, err
	}
	checkOut, err := parseAccommodationDate(accommodation.CheckOutDate, "accommodation.checkOutDate")
	if err != nil {
		return nil, err
	}
	if checkIn != nil && checkOut != nil && !checkOut.After(*checkIn) {
		return nil, apperrs.NewInvalidInput("accommodation.checkOutDate must be after checkInDate")
	}

	if err := validateAndNormalizePlaceRef(accommodation.Place, "accommodation.place"); err != nil {
		return nil, err
	}

	if accommodation.EstimatedCost != nil {
		if accommodation.EstimatedCost.Amount != nil && accommodation.EstimatedCost.Currency == "" {
			accommodation.EstimatedCost.Currency = strings.ToUpper(strings.TrimSpace(fallbackCurrency))
		}
		accommodation.EstimatedCost.Category = budget.CategoryAccommodation
		accommodation.EstimatedCost.Source = budget.SourceManual
		if err := budget.NormalizeEstimatedCost(accommodation.EstimatedCost, budget.SourceManual); err != nil {
			return nil, apperrs.NewInvalidInput("accommodation.estimatedCost: %s", err.Error())
		}
		accommodation.EstimatedCost.Category = budget.CategoryAccommodation
		accommodation.EstimatedCost.Source = budget.SourceManual
	}

	return &accommodation, nil
}

func parseAccommodationDate(value, field string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, apperrs.NewInvalidInput("%s must be in YYYY-MM-DD format", field)
	}
	return &parsed, nil
}

func accommodationActivityMetadata(accommodation *aggregate.Accommodation) map[string]any {
	if accommodation == nil {
		return map[string]any{}
	}
	return map[string]any{
		"name": accommodation.Name,
		"type": string(accommodation.Type),
	}
}
