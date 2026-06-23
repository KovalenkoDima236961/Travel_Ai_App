// Package service contains the user profile/preferences use cases.
package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
)

const (
	defaultCurrency = "EUR"
	defaultLanguage = "en"
	defaultPace     = "balanced"
	maxTextLength   = 100
	maxArrayItems   = 30
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// repository is the persistence port the use case depends on.
type repository interface {
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*entity.Profile, error)
	CreateDefaultProfile(ctx context.Context, userID uuid.UUID) (*entity.Profile, error)
	UpsertProfile(ctx context.Context, profile *entity.Profile) (*entity.Profile, error)
	GetPreferencesByUserID(ctx context.Context, userID uuid.UUID) (*entity.Preferences, error)
	CreateDefaultPreferences(ctx context.Context, userID uuid.UUID) (*entity.Preferences, error)
	UpsertPreferences(ctx context.Context, preferences *entity.Preferences) (*entity.Preferences, error)
}

// Service holds profile and preferences business logic.
type Service struct {
	repo repository
	log  *zap.Logger
}

// New constructs the user service.
func New(repo repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log}
}

// GetProfile returns the current user's profile, creating defaults when needed.
func (s *Service) GetProfile(ctx context.Context) (*entity.Profile, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := s.repo.GetProfileByUserID(ctx, user.ID)
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, domainerrs.ErrNotFound) {
		return nil, err
	}

	created, err := s.repo.CreateDefaultProfile(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	s.log.Info("default profile created", zap.String("user_id", user.ID.String()))
	return created, nil
}

// UpdateProfile upserts the current user's profile.
func (s *Service) UpdateProfile(ctx context.Context, in appdto.UpdateProfileInput) (*entity.Profile, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := buildProfile(user.ID, in)
	if err != nil {
		return nil, err
	}
	return s.repo.UpsertProfile(ctx, profile)
}

// GetPreferences returns the current user's preferences, creating defaults when needed.
func (s *Service) GetPreferences(ctx context.Context) (*entity.Preferences, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	preferences, err := s.repo.GetPreferencesByUserID(ctx, user.ID)
	if err == nil {
		return normalisePreferenceSlices(preferences), nil
	}
	if !errors.Is(err, domainerrs.ErrNotFound) {
		return nil, err
	}

	created, err := s.repo.CreateDefaultPreferences(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	s.log.Info("default preferences created", zap.String("user_id", user.ID.String()))
	return normalisePreferenceSlices(created), nil
}

// PatchPreferences merges provided fields with existing preferences and upserts
// the resulting full preferences row.
func (s *Service) PatchPreferences(ctx context.Context, in appdto.PatchPreferencesInput) (*entity.Preferences, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	current, err := s.repo.GetPreferencesByUserID(ctx, user.ID)
	if err != nil {
		if !errors.Is(err, domainerrs.ErrNotFound) {
			return nil, err
		}
		current = defaultPreferences(user.ID)
	}
	current = normalisePreferenceSlices(current)

	if err := mergePreferences(current, in); err != nil {
		return nil, err
	}
	return s.repo.UpsertPreferences(ctx, current)
}

func buildProfile(userID uuid.UUID, in appdto.UpdateProfileInput) (*entity.Profile, error) {
	displayName, err := nullableText(in.DisplayName, "displayName")
	if err != nil {
		return nil, err
	}
	homeCity, err := nullableText(in.HomeCity, "homeCity")
	if err != nil {
		return nil, err
	}
	homeCountry, err := nullableText(in.HomeCountry, "homeCountry")
	if err != nil {
		return nil, err
	}

	currency := strings.TrimSpace(in.PreferredCurrency)
	if !currencyPattern.MatchString(currency) {
		return nil, apperrs.NewInvalidInput("preferredCurrency must be 3 uppercase letters")
	}

	language := strings.TrimSpace(in.PreferredLanguage)
	if utf8.RuneCountInString(language) < 2 || utf8.RuneCountInString(language) > 10 {
		return nil, apperrs.NewInvalidInput("preferredLanguage must be between 2 and 10 characters")
	}

	return &entity.Profile{
		UserID:            userID,
		DisplayName:       displayName,
		HomeCity:          homeCity,
		HomeCountry:       homeCountry,
		PreferredCurrency: currency,
		PreferredLanguage: language,
	}, nil
}

func nullableText(value, field string) (*string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > maxTextLength {
		return nil, apperrs.NewInvalidInput("%s must be at most %d characters", field, maxTextLength)
	}
	return &trimmed, nil
}

func defaultPreferences(userID uuid.UUID) *entity.Preferences {
	return &entity.Preferences{
		UserID:              userID,
		TravelStyles:        []string{},
		Pace:                defaultPace,
		FoodPreferences:     []string{},
		Avoid:               []string{},
		PreferredTransport:  []string{},
		AccommodationStyle:  []string{},
		DietaryRestrictions: []string{},
	}
}

func mergePreferences(current *entity.Preferences, in appdto.PatchPreferencesInput) error {
	if in.TravelStyles != nil {
		current.TravelStyles = cleanStringArray(*in.TravelStyles)
	}
	if in.Pace != nil {
		pace := strings.TrimSpace(*in.Pace)
		if pace != "relaxed" && pace != "balanced" && pace != "intensive" {
			return apperrs.NewInvalidInput("pace must be one of: relaxed balanced intensive")
		}
		current.Pace = pace
	}
	if in.MaxWalkingKmPerDay != nil {
		if *in.MaxWalkingKmPerDay < 0 || *in.MaxWalkingKmPerDay > 50 {
			return apperrs.NewInvalidInput("maxWalkingKmPerDay must be between 0 and 50")
		}
		current.MaxWalkingKmPerDay = in.MaxWalkingKmPerDay
	}
	if in.FoodPreferences != nil {
		current.FoodPreferences = cleanStringArray(*in.FoodPreferences)
	}
	if in.Avoid != nil {
		current.Avoid = cleanStringArray(*in.Avoid)
	}
	if in.PreferredTransport != nil {
		current.PreferredTransport = cleanStringArray(*in.PreferredTransport)
	}
	if in.AccommodationStyle != nil {
		current.AccommodationStyle = cleanStringArray(*in.AccommodationStyle)
	}
	if in.DietaryRestrictions != nil {
		current.DietaryRestrictions = cleanStringArray(*in.DietaryRestrictions)
	}
	return nil
}

func cleanStringArray(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
		if len(out) == maxArrayItems {
			break
		}
	}
	return out
}

func normalisePreferenceSlices(p *entity.Preferences) *entity.Preferences {
	if p.TravelStyles == nil {
		p.TravelStyles = []string{}
	}
	if p.Pace == "" {
		p.Pace = defaultPace
	}
	if p.FoodPreferences == nil {
		p.FoodPreferences = []string{}
	}
	if p.Avoid == nil {
		p.Avoid = []string{}
	}
	if p.PreferredTransport == nil {
		p.PreferredTransport = []string{}
	}
	if p.AccommodationStyle == nil {
		p.AccommodationStyle = []string{}
	}
	if p.DietaryRestrictions == nil {
		p.DietaryRestrictions = []string{}
	}
	return p
}
