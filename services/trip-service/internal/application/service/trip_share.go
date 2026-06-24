package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/sharing"
)

const shareTokenCreateAttempts = 5

var (
	ErrSharePasswordRequired = errors.New("share password required")
	ErrInvalidSharePassword  = errors.New("invalid share password")
)

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
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}

	share, err := s.repo.GetTripShareByTripAndUser(ctx, tripID, ownerID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return appdto.TripShareInfo{Enabled: false, PasswordRequired: false}, nil
		}
		return appdto.TripShareInfo{}, err
	}

	return s.tripShareInfo(share), nil
}

// CreateOrEnableTripShare returns an active share link for an owned trip.
func (s *Service) CreateOrEnableTripShare(ctx context.Context, tripID uuid.UUID, in appdto.CreateTripShareInput) (appdto.TripShareInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	if !s.publicSharingEnabled {
		return appdto.TripShareInfo{}, apperrs.NewInvalidInput("public sharing is disabled")
	}
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}

	settings, hasSettings, err := s.shareSettingsForCreate(in)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}

	share, err := s.repo.GetTripShareByTripAndUser(ctx, tripID, ownerID)
	switch {
	case err == nil && share.Enabled:
		if hasSettings {
			updated, err := s.repo.UpdateTripShareSettings(ctx, tripID, ownerID, settings.expiresAt, settings.passwordRequired, settings.passwordHash)
			if err != nil {
				return appdto.TripShareInfo{}, err
			}
			return s.tripShareInfo(updated), nil
		}
		return s.tripShareInfo(share), nil
	case err == nil:
		enabled, err := s.repo.EnableTripShare(ctx, tripID, ownerID)
		if err != nil {
			return appdto.TripShareInfo{}, err
		}
		if hasSettings {
			enabled, err = s.repo.UpdateTripShareSettings(ctx, tripID, ownerID, settings.expiresAt, settings.passwordRequired, settings.passwordHash)
			if err != nil {
				return appdto.TripShareInfo{}, err
			}
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
			TripID:           tripID,
			UserID:           ownerID,
			ShareToken:       token,
			Enabled:          true,
			ExpiresAt:        settings.expiresAt,
			PasswordHash:     settings.passwordHash,
			PasswordRequired: settings.passwordRequired,
		})
		if err == nil {
			return s.tripShareInfo(created), nil
		}
		if !errors.Is(err, domainerrs.ErrConflict) {
			return appdto.TripShareInfo{}, err
		}

		existing, lookupErr := s.repo.GetTripShareByTripAndUser(ctx, tripID, ownerID)
		if lookupErr == nil {
			if existing.Enabled {
				if hasSettings {
					updated, updateErr := s.repo.UpdateTripShareSettings(ctx, tripID, ownerID, settings.expiresAt, settings.passwordRequired, settings.passwordHash)
					if updateErr != nil {
						return appdto.TripShareInfo{}, updateErr
					}
					return s.tripShareInfo(updated), nil
				}
				return s.tripShareInfo(existing), nil
			}
			enabled, enableErr := s.repo.EnableTripShare(ctx, tripID, ownerID)
			if enableErr != nil {
				return appdto.TripShareInfo{}, enableErr
			}
			if hasSettings {
				enabled, enableErr = s.repo.UpdateTripShareSettings(ctx, tripID, ownerID, settings.expiresAt, settings.passwordRequired, settings.passwordHash)
				if enableErr != nil {
					return appdto.TripShareInfo{}, enableErr
				}
			}
			return s.tripShareInfo(enabled), nil
		}
	}

	return appdto.TripShareInfo{}, fmt.Errorf("create unique trip share token: %w", domainerrs.ErrConflict)
}

// UpdateTripShareSettings changes owner-controlled share expiration/password
// settings without rotating the share token.
func (s *Service) UpdateTripShareSettings(ctx context.Context, tripID uuid.UUID, in appdto.UpdateTripShareInput) (appdto.TripShareInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	if !s.publicSharingEnabled {
		return appdto.TripShareInfo{}, apperrs.NewInvalidInput("public sharing is disabled")
	}
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}

	share, err := s.repo.GetTripShareByTripAndUser(ctx, tripID, ownerID)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	settings, changed, err := s.shareSettingsForUpdate(share, in)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	if !changed {
		return s.tripShareInfo(share), nil
	}

	updated, err := s.repo.UpdateTripShareSettings(ctx, tripID, ownerID, settings.expiresAt, settings.passwordRequired, settings.passwordHash)
	if err != nil {
		return appdto.TripShareInfo{}, err
	}
	return s.tripShareInfo(updated), nil
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
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return err
	}
	if _, err := s.repo.DisableTripShare(ctx, tripID, ownerID); err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
		return err
	}
	return nil
}

// GetPublicTripByShareToken returns a sanitized-source trip and share metadata
// for an enabled public token. It intentionally does not read auth context.
func (s *Service) GetPublicTripByShareToken(ctx context.Context, shareToken string, shareAccessToken string) (*entity.Trip, *entity.TripShare, error) {
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
	if !sharing.IsShareActive(share, time.Now().UTC()) {
		return nil, nil, domainerrs.ErrNotFound
	}
	if sharing.RequiresPassword(share) {
		if share.PasswordHash == nil {
			return nil, nil, ErrSharePasswordRequired
		}
		if err := s.publicShareTokens.ValidatePublicShareAccessToken(shareAccessToken, token); err != nil {
			return nil, nil, ErrSharePasswordRequired
		}
		if !sharing.IsShareActive(share, time.Now().UTC()) {
			return nil, nil, domainerrs.ErrNotFound
		}
	}

	trip, err := s.repo.GetByID(ctx, share.TripID)
	if err != nil {
		return nil, nil, err
	}
	return trip, share, nil
}

// GetPublicTripShareStatus returns public-safe metadata for an active share.
func (s *Service) GetPublicTripShareStatus(ctx context.Context, shareToken string) (appdto.PublicShareStatus, error) {
	share, err := s.activeShareByToken(ctx, shareToken)
	if err != nil {
		return appdto.PublicShareStatus{}, err
	}
	return appdto.PublicShareStatus{
		Available:        true,
		PasswordRequired: sharing.RequiresPassword(share),
		Expired:          false,
	}, nil
}

// UnlockPublicTripShare verifies a public share password and returns a
// short-lived token scoped to exactly one share token.
func (s *Service) UnlockPublicTripShare(ctx context.Context, shareToken string, password string) (appdto.PublicShareUnlockResponse, error) {
	share, err := s.activeShareByToken(ctx, shareToken)
	if err != nil {
		return appdto.PublicShareUnlockResponse{}, err
	}

	if sharing.RequiresPassword(share) {
		if share.PasswordHash == nil || !sharing.VerifySharePassword(*share.PasswordHash, password) {
			return appdto.PublicShareUnlockResponse{}, ErrInvalidSharePassword
		}
	}

	accessToken, expiresAt, err := s.publicShareTokens.CreatePublicShareAccessToken(share.ShareToken)
	if err != nil {
		return appdto.PublicShareUnlockResponse{}, err
	}
	return appdto.PublicShareUnlockResponse{AccessToken: accessToken, ExpiresAt: expiresAt}, nil
}

func (s *Service) tripShareInfo(share *entity.TripShare) appdto.TripShareInfo {
	if share == nil {
		return appdto.TripShareInfo{Enabled: false, PasswordRequired: false}
	}

	info := appdto.TripShareInfo{
		Enabled:          share.Enabled,
		CreatedAt:        &share.CreatedAt,
		UpdatedAt:        &share.UpdatedAt,
		DisabledAt:       share.DisabledAt,
		ExpiresAt:        share.ExpiresAt,
		Expired:          sharing.IsShareExpired(share, time.Now().UTC()),
		PasswordRequired: share.PasswordRequired,
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

type tripShareSettings struct {
	expiresAt        *time.Time
	passwordRequired bool
	passwordHash     *string
}

func (s *Service) shareSettingsForCreate(in appdto.CreateTripShareInput) (tripShareSettings, bool, error) {
	settings := tripShareSettings{}
	hasSettings := false
	if in.ExpiresAt != nil {
		expiresAt := in.ExpiresAt.UTC()
		if !expiresAt.After(time.Now().UTC()) {
			return tripShareSettings{}, false, apperrs.NewInvalidInput("expiresAt must be in the future")
		}
		settings.expiresAt = &expiresAt
		hasSettings = true
	}
	if in.Password != "" {
		hash, err := sharing.HashSharePassword(in.Password)
		if err != nil {
			return tripShareSettings{}, false, apperrs.NewInvalidInput("%s", err.Error())
		}
		settings.passwordHash = &hash
		settings.passwordRequired = true
		hasSettings = true
	}
	return settings, hasSettings, nil
}

func (s *Service) shareSettingsForUpdate(share *entity.TripShare, in appdto.UpdateTripShareInput) (tripShareSettings, bool, error) {
	if in.Password != "" && in.ClearPassword {
		return tripShareSettings{}, false, apperrs.NewInvalidInput("password and clearPassword cannot be used together")
	}
	if in.ExpiresAt != nil && in.ClearExpiration {
		return tripShareSettings{}, false, apperrs.NewInvalidInput("expiresAt and clearExpiration cannot be used together")
	}

	settings := tripShareSettings{
		expiresAt:        share.ExpiresAt,
		passwordRequired: share.PasswordRequired,
		passwordHash:     share.PasswordHash,
	}
	changed := false

	if in.ClearExpiration {
		settings.expiresAt = nil
		changed = true
	} else if in.ExpiresAt != nil {
		expiresAt := in.ExpiresAt.UTC()
		if !expiresAt.After(time.Now().UTC()) {
			return tripShareSettings{}, false, apperrs.NewInvalidInput("expiresAt must be in the future")
		}
		settings.expiresAt = &expiresAt
		changed = true
	}

	if in.ClearPassword {
		settings.passwordRequired = false
		settings.passwordHash = nil
		changed = true
	} else if in.Password != "" {
		hash, err := sharing.HashSharePassword(in.Password)
		if err != nil {
			return tripShareSettings{}, false, apperrs.NewInvalidInput("%s", err.Error())
		}
		settings.passwordRequired = true
		settings.passwordHash = &hash
		changed = true
	}

	return settings, changed, nil
}

func (s *Service) activeShareByToken(ctx context.Context, shareToken string) (*entity.TripShare, error) {
	if !s.publicSharingEnabled {
		return nil, domainerrs.ErrNotFound
	}
	token := strings.TrimSpace(shareToken)
	if token == "" {
		return nil, domainerrs.ErrNotFound
	}
	share, err := s.repo.GetTripShareByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !sharing.IsShareActive(share, time.Now().UTC()) {
		return nil, domainerrs.ErrNotFound
	}
	return share, nil
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
