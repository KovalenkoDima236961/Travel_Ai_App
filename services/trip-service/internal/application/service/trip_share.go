package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

const shareTokenCreateAttempts = 5

// GetTripShare returns the owner's current share status for a trip. Missing
// share rows are represented as enabled=false so the web app can render a
// simple "create link" state.
func (s *Service) GetTripShare(ctx context.Context, tripID uuid.UUID) (appdto.TripShareInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	if !s.publicSharingEnabled {
		return appdto.TripShareInfo{}, apperrs.NewInvalidInput("public sharing is disabled")
	}
	if _, err := s.repo.GetByIDAndUserID(ctx, tripID, user.ID); err != nil {
		return appdto.TripShareInfo{}, err
	}

	share, err := s.repo.GetTripShareByTripAndUser(ctx, tripID, user.ID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return appdto.TripShareInfo{Enabled: false}, nil
		}
		return appdto.TripShareInfo{}, err
	}

	return s.tripShareInfo(share), nil
}

// CreateOrEnableTripShare returns an active share link for an owned trip.
func (s *Service) CreateOrEnableTripShare(ctx context.Context, tripID uuid.UUID) (appdto.TripShareInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	if !s.publicSharingEnabled {
		return appdto.TripShareInfo{}, apperrs.NewInvalidInput("public sharing is disabled")
	}
	if _, err := s.repo.GetByIDAndUserID(ctx, tripID, user.ID); err != nil {
		return appdto.TripShareInfo{}, err
	}

	share, err := s.repo.GetTripShareByTripAndUser(ctx, tripID, user.ID)
	switch {
	case err == nil && share.Enabled:
		return s.tripShareInfo(share), nil
	case err == nil:
		enabled, err := s.repo.EnableTripShare(ctx, tripID, user.ID)
		if err != nil {
			return appdto.TripShareInfo{}, err
		}
		return s.tripShareInfo(enabled), nil
	case !errors.Is(err, domainerrs.ErrNotFound):
		return appdto.TripShareInfo{}, err
	}

	for attempt := 0; attempt < shareTokenCreateAttempts; attempt++ {
		token, err := generateShareToken(s.shareTokenBytes)
		if err != nil {
			return appdto.TripShareInfo{}, err
		}

		created, err := s.repo.CreateTripShare(ctx, &entity.TripShare{
			TripID:     tripID,
			UserID:     user.ID,
			ShareToken: token,
			Enabled:    true,
		})
		if err == nil {
			return s.tripShareInfo(created), nil
		}
		if !errors.Is(err, domainerrs.ErrConflict) {
			return appdto.TripShareInfo{}, err
		}

		existing, lookupErr := s.repo.GetTripShareByTripAndUser(ctx, tripID, user.ID)
		if lookupErr == nil {
			if existing.Enabled {
				return s.tripShareInfo(existing), nil
			}
			enabled, enableErr := s.repo.EnableTripShare(ctx, tripID, user.ID)
			if enableErr != nil {
				return appdto.TripShareInfo{}, enableErr
			}
			return s.tripShareInfo(enabled), nil
		}
	}

	return appdto.TripShareInfo{}, fmt.Errorf("create unique trip share token: %w", domainerrs.ErrConflict)
}

// DisableTripShare disables the owner's share link. It is idempotent once the
// trip ownership check succeeds.
func (s *Service) DisableTripShare(ctx context.Context, tripID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	if !s.publicSharingEnabled {
		return apperrs.NewInvalidInput("public sharing is disabled")
	}
	if _, err := s.repo.GetByIDAndUserID(ctx, tripID, user.ID); err != nil {
		return err
	}
	if _, err := s.repo.DisableTripShare(ctx, tripID, user.ID); err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
		return err
	}
	return nil
}

// GetPublicTripByShareToken returns a sanitized-source trip and share metadata
// for an enabled public token. It intentionally does not read auth context.
func (s *Service) GetPublicTripByShareToken(ctx context.Context, shareToken string) (*entity.Trip, *entity.TripShare, error) {
	if !s.publicSharingEnabled {
		return nil, nil, domainerrs.ErrNotFound
	}
	token := strings.TrimSpace(shareToken)
	if token == "" {
		return nil, nil, domainerrs.ErrNotFound
	}

	share, err := s.repo.GetTripShareByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	if !share.Enabled {
		return nil, nil, domainerrs.ErrNotFound
	}

	trip, err := s.repo.GetByID(ctx, share.TripID)
	if err != nil {
		return nil, nil, err
	}
	return trip, share, nil
}

func (s *Service) tripShareInfo(share *entity.TripShare) appdto.TripShareInfo {
	if share == nil {
		return appdto.TripShareInfo{Enabled: false}
	}

	info := appdto.TripShareInfo{
		Enabled:    share.Enabled,
		CreatedAt:  &share.CreatedAt,
		DisabledAt: share.DisabledAt,
	}
	if share.Enabled {
		info.ShareToken = share.ShareToken
		if s.publicWebBaseURL != "" {
			info.ShareURL = s.publicWebBaseURL + "/share/" + share.ShareToken
		} else {
			info.ShareURL = "/share/" + share.ShareToken
		}
	}
	return info
}

func generateShareToken(byteCount int) (string, error) {
	if byteCount < 32 {
		byteCount = 32
	}
	randomBytes := make([]byte, byteCount)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate share token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}
