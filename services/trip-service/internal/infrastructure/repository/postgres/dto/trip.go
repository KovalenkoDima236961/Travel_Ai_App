// Package dto maps between the trip domain entity and its PostgreSQL row
// representation (pgtype). It keeps all persistence-shape concerns out of the
// repository's query-building code.
package dto

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

// Columns is the canonical column order used by all SELECT/RETURNING statements
// and the Scan helper.
const Columns = "id, user_id, destination, start_date, days, budget_amount, " +
	"budget_currency, travelers, interests, pace, status, itinerary, itinerary_revision, created_at, updated_at"

// InsertColumns returns the columns set on INSERT (DB-defaulted columns omitted),
// in the same order as InsertValues.
func InsertColumns() []string {
	return []string{
		"user_id", "destination", "start_date", "days", "budget_amount",
		"budget_currency", "travelers", "interests", "pace", "status",
	}
}

// InsertValues returns the values for InsertColumns, in matching order.
func InsertValues(t *entity.Trip) ([]any, error) {
	interests, err := marshalInterests(t.Interests)
	if err != nil {
		return nil, err
	}
	return []any{
		toPgUUIDPtr(t.UserID), t.Destination, toPgDate(t.StartDate), t.Days,
		toPgNumeric(t.BudgetAmount), toPgText(t.BudgetCurrency), t.Travelers,
		interests, t.Pace, string(t.Status),
	}, nil
}

// IDArg encodes a trip id for use in a WHERE clause.
func IDArg(id uuid.UUID) pgtype.UUID {
	return toPgUUID(id)
}

// NumericArg encodes a nullable decimal (e.g. budget_amount) for a query value.
func NumericArg(f *float64) pgtype.Numeric {
	return toPgNumeric(f)
}

// TextArg encodes a nullable text column (e.g. budget_currency); an empty string
// is stored as NULL.
func TextArg(s string) pgtype.Text {
	return toPgText(s)
}

// Scan reads a single row (in Columns order) into a domain Trip. It returns
// domain errs.ErrNotFound when the row is absent.
func Scan(row pgx.Row) (*entity.Trip, error) {
	var (
		id, userID           pgtype.UUID
		destination          string
		startDate            pgtype.Date
		days                 int32
		budgetAmount         pgtype.Numeric
		budgetCurrency       pgtype.Text
		travelers            pgtype.Int4
		interestsRaw         []byte
		pace, status         string
		itineraryRaw         []byte
		itineraryRevision    int32
		createdAt, updatedAt pgtype.Timestamp
	)

	err := row.Scan(
		&id, &userID, &destination, &startDate, &days, &budgetAmount,
		&budgetCurrency, &travelers, &interestsRaw, &pace, &status,
		&itineraryRaw, &itineraryRevision, &createdAt, &updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}

	interests, err := unmarshalInterests(interestsRaw)
	if err != nil {
		return nil, err
	}

	t := &entity.Trip{
		ID:                uuid.UUID(id.Bytes),
		UserID:            fromPgUUID(userID),
		Destination:       destination,
		StartDate:         fromPgDate(startDate),
		Days:              days,
		BudgetAmount:      fromPgNumeric(budgetAmount),
		BudgetCurrency:    budgetCurrency.String,
		Travelers:         travelers.Int32,
		Interests:         interests,
		Pace:              pace,
		Status:            entity.Status(status),
		ItineraryRevision: int(itineraryRevision),
		CreatedAt:         createdAt.Time,
		UpdatedAt:         updatedAt.Time,
	}
	if len(itineraryRaw) > 0 {
		t.Itinerary = json.RawMessage(itineraryRaw)
	}

	return t, nil
}

// --- mapping helpers: domain (plain Go) <-> pgtype ---

func marshalInterests(interests []string) ([]byte, error) {
	if interests == nil {
		interests = []string{}
	}
	b, err := json.Marshal(interests)
	if err != nil {
		return nil, fmt.Errorf("marshal interests: %w", err)
	}
	return b, nil
}

func unmarshalInterests(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var interests []string
	if err := json.Unmarshal(raw, &interests); err != nil {
		return nil, fmt.Errorf("unmarshal interests: %w", err)
	}
	return interests, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func fromPgUUID(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	id := uuid.UUID(p.Bytes)
	return &id
}

func toPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func fromPgDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func toPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// toPgNumeric converts a float64 to NUMERIC(10,2) via integer cents, avoiding
// binary-float artefacts.
func toPgNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	cents := big.NewInt(int64(math.Round(*f * 100)))
	return pgtype.Numeric{Int: cents, Exp: -2, Valid: true}
}

// fromPgNumeric converts NUMERIC back to float64 using big.Rat for an exact
// decimal-to-float conversion.
func fromPgNumeric(n pgtype.Numeric) *float64 {
	if !n.Valid || n.Int == nil {
		return nil
	}
	rat := new(big.Rat).SetInt(n.Int)
	if n.Exp != 0 {
		scale := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs32(n.Exp))), nil))
		if n.Exp > 0 {
			rat.Mul(rat, scale)
		} else {
			rat.Quo(rat, scale)
		}
	}
	v, _ := rat.Float64()
	return &v
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
