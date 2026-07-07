package dto

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/storage/postgres"
)

// PreferencesColumns is the canonical column order used by SELECT/RETURNING.
const PreferencesColumns = "user_id, travel_styles, pace, max_walking_km_per_day, food_preferences, avoid, preferred_transport, accommodation_style, dietary_restrictions, created_at, updated_at"

// PreferencesInsertColumns returns the columns set on preferences upsert.
func PreferencesInsertColumns() []string {
	return []string{
		"user_id", "travel_styles", "pace", "max_walking_km_per_day",
		"food_preferences", "avoid", "preferred_transport",
		"accommodation_style", "dietary_restrictions",
	}
}

// PreferencesInsertValues returns values for PreferencesInsertColumns.
func PreferencesInsertValues(p *entity.Preferences) ([]any, error) {
	travelStyles, err := marshalStringArray(p.TravelStyles)
	if err != nil {
		return nil, err
	}
	foodPreferences, err := marshalStringArray(p.FoodPreferences)
	if err != nil {
		return nil, err
	}
	avoid, err := marshalStringArray(p.Avoid)
	if err != nil {
		return nil, err
	}
	preferredTransport, err := marshalStringArray(p.PreferredTransport)
	if err != nil {
		return nil, err
	}
	accommodationStyle, err := marshalStringArray(p.AccommodationStyle)
	if err != nil {
		return nil, err
	}
	dietaryRestrictions, err := marshalStringArray(p.DietaryRestrictions)
	if err != nil {
		return nil, err
	}

	return []any{
		UUIDArg(p.UserID),
		travelStyles,
		p.Pace,
		toPgNumericPtr(p.MaxWalkingKmPerDay),
		foodPreferences,
		avoid,
		preferredTransport,
		accommodationStyle,
		dietaryRestrictions,
	}, nil
}

// ScanPreferences reads a row into domain Preferences.
func ScanPreferences(row pgx.Row) (*entity.Preferences, error) {
	var (
		userID                pgtype.UUID
		travelStylesRaw       []byte
		pace                  string
		maxWalkingKmPerDay    pgtype.Numeric
		foodPreferencesRaw    []byte
		avoidRaw              []byte
		preferredTransportRaw []byte
		accommodationStyleRaw []byte
		dietaryRaw            []byte
		createdAt, updatedAt  pgtype.Timestamp
	)

	err := row.Scan(
		&userID, &travelStylesRaw, &pace, &maxWalkingKmPerDay,
		&foodPreferencesRaw, &avoidRaw, &preferredTransportRaw,
		&accommodationStyleRaw, &dietaryRaw, &createdAt, &updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan preferences: %w", err)
	}

	travelStyles, err := unmarshalStringArray(travelStylesRaw)
	if err != nil {
		return nil, err
	}
	foodPreferences, err := unmarshalStringArray(foodPreferencesRaw)
	if err != nil {
		return nil, err
	}
	avoid, err := unmarshalStringArray(avoidRaw)
	if err != nil {
		return nil, err
	}
	preferredTransport, err := unmarshalStringArray(preferredTransportRaw)
	if err != nil {
		return nil, err
	}
	accommodationStyle, err := unmarshalStringArray(accommodationStyleRaw)
	if err != nil {
		return nil, err
	}
	dietaryRestrictions, err := unmarshalStringArray(dietaryRaw)
	if err != nil {
		return nil, err
	}

	return &entity.Preferences{
		UserID:              fromPgUUID(userID),
		TravelStyles:        travelStyles,
		Pace:                pace,
		MaxWalkingKmPerDay:  fromPgNumericPtr(maxWalkingKmPerDay),
		FoodPreferences:     foodPreferences,
		Avoid:               avoid,
		PreferredTransport:  preferredTransport,
		AccommodationStyle:  accommodationStyle,
		DietaryRestrictions: dietaryRestrictions,
		CreatedAt:           createdAt.Time,
		UpdatedAt:           updatedAt.Time,
	}, nil
}
