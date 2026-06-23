// Package postgres is the PostgreSQL adapter for user profile/preferences.
package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/infrastructure/repository/postgres/dto"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/storage/postgres"
)

// Repository persists profiles and preferences using squirrel over pgx.
type Repository struct {
	db *storage.DB
}

// New constructs the user repository.
func New(db *storage.DB) *Repository {
	return &Repository{db: db}
}

// GetProfileByUserID loads one user's profile.
func (r *Repository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*entity.Profile, error) {
	query, args, err := r.db.Builder.
		Select(dto.ProfileColumns).
		From("user_profiles").
		Where(sq.Eq{"user_id": dto.UUIDArg(userID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build profile select: %w", err)
	}

	return dto.ScanProfile(r.db.QueryRow(ctx, query, args...))
}

// CreateDefaultProfile creates a default profile or returns the existing row.
func (r *Repository) CreateDefaultProfile(ctx context.Context, userID uuid.UUID) (*entity.Profile, error) {
	query, args, err := r.db.Builder.
		Insert("user_profiles").
		Columns("user_id").
		Values(dto.UUIDArg(userID)).
		Suffix("ON CONFLICT (user_id) DO UPDATE SET user_id = user_profiles.user_id RETURNING " + dto.ProfileColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build default profile insert: %w", err)
	}

	return dto.ScanProfile(r.db.QueryRow(ctx, query, args...))
}

// UpsertProfile stores a complete profile row.
func (r *Repository) UpsertProfile(ctx context.Context, profile *entity.Profile) (*entity.Profile, error) {
	values := dto.ProfileInsertValues(profile)

	query, args, err := r.db.Builder.
		Insert("user_profiles").
		Columns(dto.ProfileInsertColumns()...).
		Values(values...).
		Suffix(`ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			home_city = EXCLUDED.home_city,
			home_country = EXCLUDED.home_country,
			preferred_currency = EXCLUDED.preferred_currency,
			preferred_language = EXCLUDED.preferred_language,
			updated_at = NOW()
			RETURNING ` + dto.ProfileColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build profile upsert: %w", err)
	}

	return dto.ScanProfile(r.db.QueryRow(ctx, query, args...))
}

// GetPreferencesByUserID loads one user's preferences.
func (r *Repository) GetPreferencesByUserID(ctx context.Context, userID uuid.UUID) (*entity.Preferences, error) {
	query, args, err := r.db.Builder.
		Select(dto.PreferencesColumns).
		From("user_preferences").
		Where(sq.Eq{"user_id": dto.UUIDArg(userID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build preferences select: %w", err)
	}

	return dto.ScanPreferences(r.db.QueryRow(ctx, query, args...))
}

// CreateDefaultPreferences creates default preferences or returns the existing row.
func (r *Repository) CreateDefaultPreferences(ctx context.Context, userID uuid.UUID) (*entity.Preferences, error) {
	query, args, err := r.db.Builder.
		Insert("user_preferences").
		Columns("user_id").
		Values(dto.UUIDArg(userID)).
		Suffix("ON CONFLICT (user_id) DO UPDATE SET user_id = user_preferences.user_id RETURNING " + dto.PreferencesColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build default preferences insert: %w", err)
	}

	return dto.ScanPreferences(r.db.QueryRow(ctx, query, args...))
}

// UpsertPreferences stores a complete preferences row.
func (r *Repository) UpsertPreferences(ctx context.Context, preferences *entity.Preferences) (*entity.Preferences, error) {
	values, err := dto.PreferencesInsertValues(preferences)
	if err != nil {
		return nil, err
	}

	query, args, err := r.db.Builder.
		Insert("user_preferences").
		Columns(dto.PreferencesInsertColumns()...).
		Values(values...).
		Suffix(`ON CONFLICT (user_id) DO UPDATE SET
			travel_styles = EXCLUDED.travel_styles,
			pace = EXCLUDED.pace,
			max_walking_km_per_day = EXCLUDED.max_walking_km_per_day,
			food_preferences = EXCLUDED.food_preferences,
			avoid = EXCLUDED.avoid,
			preferred_transport = EXCLUDED.preferred_transport,
			accommodation_style = EXCLUDED.accommodation_style,
			dietary_restrictions = EXCLUDED.dietary_restrictions,
			updated_at = NOW()
			RETURNING ` + dto.PreferencesColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build preferences upsert: %w", err)
	}

	return dto.ScanPreferences(r.db.QueryRow(ctx, query, args...))
}
