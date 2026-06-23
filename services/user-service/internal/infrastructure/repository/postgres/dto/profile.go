package dto

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/storage/postgres"
)

// ProfileColumns is the canonical column order used by SELECT/RETURNING.
const ProfileColumns = "user_id, display_name, home_city, home_country, preferred_currency, preferred_language, created_at, updated_at"

// ProfileInsertColumns returns the columns set on profile upsert.
func ProfileInsertColumns() []string {
	return []string{
		"user_id", "display_name", "home_city", "home_country",
		"preferred_currency", "preferred_language",
	}
}

// ProfileInsertValues returns values for ProfileInsertColumns.
func ProfileInsertValues(p *entity.Profile) []any {
	return []any{
		UUIDArg(p.UserID),
		toPgTextPtr(p.DisplayName),
		toPgTextPtr(p.HomeCity),
		toPgTextPtr(p.HomeCountry),
		p.PreferredCurrency,
		p.PreferredLanguage,
	}
}

// ScanProfile reads a row into a domain Profile.
func ScanProfile(row pgx.Row) (*entity.Profile, error) {
	var (
		userID                pgtype.UUID
		displayName, homeCity pgtype.Text
		homeCountry           pgtype.Text
		preferredCurrency     string
		preferredLanguage     string
		createdAt, updatedAt  pgtype.Timestamp
	)

	err := row.Scan(
		&userID, &displayName, &homeCity, &homeCountry,
		&preferredCurrency, &preferredLanguage, &createdAt, &updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan profile: %w", err)
	}

	return &entity.Profile{
		UserID:            fromPgUUID(userID),
		DisplayName:       fromPgTextPtr(displayName),
		HomeCity:          fromPgTextPtr(homeCity),
		HomeCountry:       fromPgTextPtr(homeCountry),
		PreferredCurrency: preferredCurrency,
		PreferredLanguage: preferredLanguage,
		CreatedAt:         createdAt.Time,
		UpdatedAt:         updatedAt.Time,
	}, nil
}
