package service

import (
	"context"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// ListItineraryVersions returns summary snapshots for one owned trip, newest
// first. The ownership check is explicit so a non-owner receives the same 404
// shape as other trip endpoints instead of an empty list.
func (s *Service) ListItineraryVersions(ctx context.Context, tripID uuid.UUID, limit, offset int) ([]entity.ItineraryVersion, int, int, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, 0, 0, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLimit)
	}
	if offset < 0 {
		return nil, 0, 0, apperrs.NewInvalidInput("offset must be >= 0")
	}

	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, 0, 0, err
	}

	versions, err := s.repo.ListItineraryVersionsByTrip(ctx, tripID, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	return versions, limit, offset, nil
}

// GetItineraryVersion returns one full snapshot for an owned trip.
func (s *Service) GetItineraryVersion(ctx context.Context, tripID, versionID uuid.UUID) (*entity.ItineraryVersion, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}

	return s.repo.GetItineraryVersionByIDTrip(ctx, versionID, tripID)
}

// RestoreItineraryVersion replaces the current trip itinerary with an existing
// snapshot and records that restore as a new version. Old versions remain
// untouched.
func (s *Service) RestoreItineraryVersion(ctx context.Context, tripID, versionID uuid.UUID) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return nil, err
	}

	version, err := s.repo.GetItineraryVersionByIDTrip(ctx, versionID, tripID)
	if err != nil {
		return nil, err
	}

	normalized, err := validateAndNormalizeItinerary(version.Itinerary)
	if err != nil {
		return nil, err
	}

	return s.saveItineraryWithVersion(ctx, tripID, ownerID, user.ID, normalized, entity.ItineraryVersionSourceRestored, map[string]any{
		"restoredFromVersionId":     version.ID.String(),
		"restoredFromVersionNumber": version.VersionNumber,
	})
}
