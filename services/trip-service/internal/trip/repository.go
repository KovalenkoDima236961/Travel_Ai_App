package trip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

// ErrNotFound is returned when a trip does not exist.
var ErrNotFound = errors.New("trip not found")

// tripColumns is the canonical column order used by all SELECT/RETURNING
// statements and the scanTrip helper.
const tripColumns = "id, user_id, destination, start_date, days, budget_amount, " +
	"budget_currency, travelers, interests, pace, status, itinerary, created_at, updated_at"

// Repository persists trips using squirrel query building over the shared
// postgres pool.
type Repository struct {
	db *postgres.DB
}

// NewRepository constructs the trip repository.
func NewRepository(db *postgres.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new trip and returns the stored row.
func (r *Repository) Create(ctx context.Context, t *Trip) (*Trip, error) {
	interests, err := marshalInterests(t.Interests)
	if err != nil {
		return nil, err
	}

	query, args, err := r.db.Builder.
		Insert("trips").
		Columns(
			"user_id", "destination", "start_date", "days", "budget_amount",
			"budget_currency", "travelers", "interests", "pace", "status",
		).
		Values(
			toPgUUIDPtr(t.UserID), t.Destination, toPgDate(t.StartDate), t.Days,
			toPgNumeric(t.BudgetAmount), toPgText(t.BudgetCurrency), t.Travelers,
			interests, t.Pace, string(t.Status),
		).
		Suffix("RETURNING " + tripColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert: %w", err)
	}

	return scanTrip(r.db.QueryRow(ctx, query, args...))
}

// GetByID loads a trip by UUID, returning ErrNotFound when absent.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	query, args, err := r.db.Builder.
		Select(tripColumns).
		From("trips").
		Where(sq.Eq{"id": toPgUUID(id)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select: %w", err)
	}

	return scanTrip(r.db.QueryRow(ctx, query, args...))
}

// UpdateStatus transitions a trip to the given status.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (*Trip, error) {
	query, args, err := r.db.Builder.
		Update("trips").
		Set("status", string(status)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": toPgUUID(id)}).
		Suffix("RETURNING " + tripColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update status: %w", err)
	}

	return scanTrip(r.db.QueryRow(ctx, query, args...))
}

// UpdateItinerary stores the generated itinerary and resulting status.
func (r *Repository) UpdateItinerary(ctx context.Context, id uuid.UUID, itinerary json.RawMessage, status Status) (*Trip, error) {
	query, args, err := r.db.Builder.
		Update("trips").
		Set("itinerary", []byte(itinerary)).
		Set("status", string(status)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": toPgUUID(id)}).
		Suffix("RETURNING " + tripColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update itinerary: %w", err)
	}

	return scanTrip(r.db.QueryRow(ctx, query, args...))
}

// scanTrip scans a single row (in tripColumns order) into a domain Trip.
func scanTrip(row pgx.Row) (*Trip, error) {
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
		createdAt, updatedAt pgtype.Timestamp
	)

	err := row.Scan(
		&id, &userID, &destination, &startDate, &days, &budgetAmount,
		&budgetCurrency, &travelers, &interestsRaw, &pace, &status,
		&itineraryRaw, &createdAt, &updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}

	interests, err := unmarshalInterests(interestsRaw)
	if err != nil {
		return nil, err
	}

	t := &Trip{
		ID:             uuid.UUID(id.Bytes),
		UserID:         fromPgUUID(userID),
		Destination:    destination,
		StartDate:      fromPgDate(startDate),
		Days:           days,
		BudgetAmount:   fromPgNumeric(budgetAmount),
		BudgetCurrency: budgetCurrency.String,
		Travelers:      travelers.Int32,
		Interests:      interests,
		Pace:           pace,
		Status:         Status(status),
		CreatedAt:      createdAt.Time,
		UpdatedAt:      updatedAt.Time,
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
