package postgres

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

type queryer interface {
	rowQuerier
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *Repository) CreateTripExpenseWithParticipants(
	ctx context.Context,
	expense *entity.TripExpense,
	participants []entity.TripExpenseParticipant,
) (*entity.TripExpense, []entity.TripExpenseParticipant, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin create trip expense: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	created, err := r.createTripExpense(ctx, tx, expense)
	if err != nil {
		return nil, nil, err
	}
	createdParticipants := make([]entity.TripExpenseParticipant, 0, len(participants))
	for i := range participants {
		participants[i].ExpenseID = created.ID
		participants[i].TripID = created.TripID
		createdParticipant, err := r.createTripExpenseParticipant(ctx, tx, &participants[i])
		if err != nil {
			return nil, nil, err
		}
		createdParticipants = append(createdParticipants, *createdParticipant)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit create trip expense: %w", err)
	}
	committed = true
	return created, createdParticipants, nil
}

func (r *Repository) UpdateTripExpenseWithParticipants(
	ctx context.Context,
	expense *entity.TripExpense,
	participants []entity.TripExpenseParticipant,
	replaceParticipants bool,
) (*entity.TripExpense, []entity.TripExpenseParticipant, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin update trip expense: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	updated, err := r.updateTripExpense(ctx, tx, expense)
	if err != nil {
		return nil, nil, err
	}
	var updatedParticipants []entity.TripExpenseParticipant
	if replaceParticipants {
		query, args, err := r.db.Builder.
			Delete("trip_expense_participants").
			Where(sq.Eq{"expense_id": dto.IDArg(expense.ID), "trip_id": dto.IDArg(expense.TripID)}).
			ToSql()
		if err != nil {
			return nil, nil, fmt.Errorf("build delete expense participants: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, nil, fmt.Errorf("delete expense participants: %w", err)
		}
		updatedParticipants = make([]entity.TripExpenseParticipant, 0, len(participants))
		for i := range participants {
			participants[i].ExpenseID = updated.ID
			participants[i].TripID = updated.TripID
			createdParticipant, err := r.createTripExpenseParticipant(ctx, tx, &participants[i])
			if err != nil {
				return nil, nil, err
			}
			updatedParticipants = append(updatedParticipants, *createdParticipant)
		}
	} else {
		updatedParticipants, err = r.listTripExpenseParticipantsByExpense(ctx, tx, expense.TripID, expense.ID)
		if err != nil {
			return nil, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit update trip expense: %w", err)
	}
	committed = true
	return updated, updatedParticipants, nil
}

func (r *Repository) GetTripExpenseByID(ctx context.Context, tripID, expenseID uuid.UUID) (*entity.TripExpense, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripExpenseColumns).
		From("trip_expenses").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(expenseID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip expense: %w", err)
	}
	return dto.ScanTripExpense(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripExpensesByTrip(ctx context.Context, tripID uuid.UUID, filters appdto.ListExpensesInput) ([]entity.TripExpense, error) {
	builder := r.db.Builder.
		Select(dto.TripExpenseColumns).
		From("trip_expenses").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "status": string(entity.ExpenseStatusActive)})
	if filters.Category != nil {
		builder = builder.Where(sq.Eq{"category": string(*filters.Category)})
	}
	if filters.PaidByUserID != nil {
		builder = builder.Where(sq.Eq{"paid_by_user_id": dto.IDArg(*filters.PaidByUserID)})
	}
	if filters.FromDate != nil {
		builder = builder.Where(sq.GtOrEq{"expense_date": dto.DateArg(filters.FromDate)})
	}
	if filters.ToDate != nil {
		builder = builder.Where(sq.LtOrEq{"expense_date": dto.DateArg(filters.ToDate)})
	}
	if filters.LinkedOnly {
		builder = builder.Where(sq.Or{
			sq.Expr("linked_day_number IS NOT NULL"),
			sq.Expr("linked_route_leg_id IS NOT NULL"),
			sq.Eq{"linked_accommodation": true},
		})
	}
	builder = builder.OrderBy("expense_date DESC", "created_at DESC", "id DESC")
	if filters.Limit > 0 {
		builder = builder.Limit(uint64(filters.Limit))
	}
	if filters.Offset > 0 {
		builder = builder.Offset(uint64(filters.Offset))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip expenses: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip expenses: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripExpenseRows(rows)
}

func (r *Repository) SoftDeleteTripExpense(ctx context.Context, tripID, expenseID, actorUserID uuid.UUID) (*entity.TripExpense, error) {
	query, args, err := r.db.Builder.
		Update("trip_expenses").
		Set("status", string(entity.ExpenseStatusDeleted)).
		Set("deleted_at", sq.Expr("NOW()")).
		Set("deleted_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(expenseID)}).
		Where(sq.NotEq{"status": string(entity.ExpenseStatusDeleted)}).
		Suffix("RETURNING " + dto.TripExpenseColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build soft delete trip expense: %w", err)
	}
	return dto.ScanTripExpense(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListExpenseParticipantsByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripExpenseParticipant, error) {
	columns := prefixColumns("p", dto.TripExpenseParticipantColumns)
	query, args, err := r.db.Builder.
		Select(columns).
		From("trip_expense_participants p").
		Join("trip_expenses e ON e.id = p.expense_id").
		Where(sq.Eq{"p.trip_id": dto.IDArg(tripID), "e.status": string(entity.ExpenseStatusActive)}).
		OrderBy("p.user_id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list expense participants by trip: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query expense participants by trip: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripExpenseParticipantRows(rows)
}

func prefixColumns(prefix, columns string) string {
	parts := strings.Split(columns, ",")
	for i := range parts {
		parts[i] = prefix + "." + strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, ", ")
}

func (r *Repository) ListExpenseParticipantsByExpense(ctx context.Context, tripID, expenseID uuid.UUID) ([]entity.TripExpenseParticipant, error) {
	return r.listTripExpenseParticipantsByExpense(ctx, r.db, tripID, expenseID)
}

func (r *Repository) CreateTripSettlement(ctx context.Context, settlement *entity.TripSettlement) (*entity.TripSettlement, error) {
	query, args, err := r.db.Builder.
		Insert("trip_settlements").
		Columns(dto.TripSettlementInsertColumns()...).
		Values(dto.TripSettlementInsertValues(settlement)...).
		Suffix("RETURNING " + dto.TripSettlementColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip settlement: %w", err)
	}
	return dto.ScanTripSettlement(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripSettlementByID(ctx context.Context, tripID, settlementID uuid.UUID) (*entity.TripSettlement, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripSettlementColumns).
		From("trip_settlements").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(settlementID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip settlement: %w", err)
	}
	return dto.ScanTripSettlement(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripSettlementsByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripSettlement, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripSettlementColumns).
		From("trip_settlements").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("created_at DESC", "id DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip settlements: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip settlements: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripSettlementRows(rows)
}

func (r *Repository) MarkTripSettlementPaid(ctx context.Context, tripID, settlementID, actorUserID uuid.UUID, notes *string) (*entity.TripSettlement, error) {
	query, args, err := r.db.Builder.
		Update("trip_settlements").
		Set("status", string(entity.SettlementStatusPaid)).
		Set("paid_at", sq.Expr("NOW()")).
		Set("paid_by_user_id", dto.IDArg(actorUserID)).
		Set("notes", dto.TextPtrArg(notes)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(settlementID)}).
		Where(sq.NotEq{"status": string(entity.SettlementStatusCancelled)}).
		Suffix("RETURNING " + dto.TripSettlementColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark trip settlement paid: %w", err)
	}
	return dto.ScanTripSettlement(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CancelTripSettlement(ctx context.Context, tripID, settlementID, actorUserID uuid.UUID) (*entity.TripSettlement, error) {
	query, args, err := r.db.Builder.
		Update("trip_settlements").
		Set("status", string(entity.SettlementStatusCancelled)).
		Set("cancelled_at", sq.Expr("NOW()")).
		Set("cancelled_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(settlementID)}).
		Where(sq.NotEq{"status": string(entity.SettlementStatusCancelled)}).
		Suffix("RETURNING " + dto.TripSettlementColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build cancel trip settlement: %w", err)
	}
	return dto.ScanTripSettlement(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) createTripExpense(ctx context.Context, q rowQuerier, expense *entity.TripExpense) (*entity.TripExpense, error) {
	query, args, err := r.db.Builder.
		Insert("trip_expenses").
		Columns(dto.TripExpenseInsertColumns()...).
		Values(dto.TripExpenseInsertValues(expense)...).
		Suffix("RETURNING " + dto.TripExpenseColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip expense: %w", err)
	}
	return dto.ScanTripExpense(q.QueryRow(ctx, query, args...))
}

func (r *Repository) updateTripExpense(ctx context.Context, q rowQuerier, expense *entity.TripExpense) (*entity.TripExpense, error) {
	query, args, err := r.db.Builder.
		Update("trip_expenses").
		Set("title", expense.Title).
		Set("description", dto.TextPtrArg(expense.Description)).
		Set("amount", dto.NumericArg(&expense.Amount)).
		Set("currency", expense.Currency).
		Set("category", string(expense.Category)).
		Set("expense_date", dto.DateArg(&expense.ExpenseDate)).
		Set("paid_by_user_id", dto.IDArg(expense.PaidByUserID)).
		Set("split_type", string(expense.SplitType)).
		Set("linked_day_number", dto.IntPtrArg(expense.LinkedDayNumber)).
		Set("linked_item_index", dto.IntPtrArg(expense.LinkedItemIndex)).
		Set("linked_item_id", dto.TextPtrArg(expense.LinkedItemID)).
		Set("linked_route_leg_id", dto.TextPtrArg(expense.LinkedRouteLegID)).
		Set("linked_accommodation", expense.LinkedAccommodation).
		Set("notes", dto.TextPtrArg(expense.Notes)).
		Set("metadata", dto.JSONArg(expense.Metadata)).
		Set("updated_by_user_id", dto.IDArg(*expense.UpdatedByUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(expense.TripID), "id": dto.IDArg(expense.ID)}).
		Where(sq.Eq{"status": string(entity.ExpenseStatusActive)}).
		Suffix("RETURNING " + dto.TripExpenseColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip expense: %w", err)
	}
	return dto.ScanTripExpense(q.QueryRow(ctx, query, args...))
}

func (r *Repository) createTripExpenseParticipant(ctx context.Context, q rowQuerier, participant *entity.TripExpenseParticipant) (*entity.TripExpenseParticipant, error) {
	query, args, err := r.db.Builder.
		Insert("trip_expense_participants").
		Columns(dto.TripExpenseParticipantInsertColumns()...).
		Values(dto.TripExpenseParticipantInsertValues(participant)...).
		Suffix("RETURNING " + dto.TripExpenseParticipantColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create expense participant: %w", err)
	}
	return dto.ScanTripExpenseParticipant(q.QueryRow(ctx, query, args...))
}

func (r *Repository) listTripExpenseParticipantsByExpense(ctx context.Context, q queryer, tripID, expenseID uuid.UUID) ([]entity.TripExpenseParticipant, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripExpenseParticipantColumns).
		From("trip_expense_participants").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "expense_id": dto.IDArg(expenseID)}).
		OrderBy("user_id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list expense participants: %w", err)
	}
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query expense participants: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripExpenseParticipantRows(rows)
}
