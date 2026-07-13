package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateTripExpenseReceipt(ctx context.Context, receipt *entity.TripExpenseReceipt) (*entity.TripExpenseReceipt, error) {
	query, args, err := r.db.Builder.
		Insert("trip_expense_receipts").
		Columns(dto.TripExpenseReceiptInsertColumns()...).
		Values(dto.TripExpenseReceiptInsertValues(receipt)...).
		Suffix("RETURNING " + dto.TripExpenseReceiptColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip expense receipt: %w", err)
	}
	return dto.ScanTripExpenseReceipt(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripExpenseReceiptByID(ctx context.Context, tripID, receiptID uuid.UUID, includeDeleted bool) (*entity.TripExpenseReceipt, error) {
	builder := r.db.Builder.
		Select(dto.TripExpenseReceiptColumns).
		From("trip_expense_receipts").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(receiptID)})
	if !includeDeleted {
		builder = builder.Where(sq.Eq{"deleted_at": nil}).Where(sq.NotEq{"status": string(entity.ReceiptStatusDeleted)})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip expense receipt: %w", err)
	}
	return dto.ScanTripExpenseReceipt(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripExpenseReceipts(ctx context.Context, tripID uuid.UUID, filters appdto.ListReceiptsInput) ([]entity.TripExpenseReceipt, error) {
	builder := r.db.Builder.
		Select(dto.TripExpenseReceiptColumns).
		From("trip_expense_receipts").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "deleted_at": nil}).
		Where(sq.NotEq{"status": string(entity.ReceiptStatusDeleted)})
	if filters.ExpenseID != nil {
		builder = builder.Where(sq.Eq{"expense_id": dto.IDArg(*filters.ExpenseID)})
	}
	if filters.Status != nil {
		builder = builder.Where(sq.Eq{"status": string(*filters.Status)})
	}
	if filters.UnlinkedOnly {
		builder = builder.Where(sq.Eq{"expense_id": nil})
	}
	query, args, err := builder.
		OrderBy("created_at DESC", "id DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip expense receipts: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip expense receipts: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripExpenseReceiptRows(rows)
}

func (r *Repository) ListTripExpenseReceiptsByExpense(ctx context.Context, tripID, expenseID uuid.UUID) ([]entity.TripExpenseReceipt, error) {
	return r.ListTripExpenseReceipts(ctx, tripID, appdto.ListReceiptsInput{ExpenseID: &expenseID})
}

func (r *Repository) UpdateTripExpenseReceiptStatus(ctx context.Context, tripID, receiptID uuid.UUID, status entity.ReceiptStatus, actorUserID *uuid.UUID) (*entity.TripExpenseReceipt, error) {
	query, args, err := r.db.Builder.
		Update("trip_expense_receipts").
		Set("status", string(status)).
		Set("updated_by_user_id", dto.IDArgPtr(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(receiptID), "deleted_at": nil}).
		Where(sq.NotEq{"status": string(entity.ReceiptStatusDeleted)}).
		Suffix("RETURNING " + dto.TripExpenseReceiptColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update receipt status: %w", err)
	}
	return dto.ScanTripExpenseReceipt(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) AttachTripExpenseReceipt(ctx context.Context, tripID, receiptID, expenseID, actorUserID uuid.UUID) (*entity.TripExpenseReceipt, error) {
	query, args, err := r.db.Builder.
		Update("trip_expense_receipts").
		Set("expense_id", dto.IDArg(expenseID)).
		Set("status", string(entity.ReceiptStatusAttached)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(receiptID), "deleted_at": nil}).
		Where(sq.NotEq{"status": string(entity.ReceiptStatusDeleted)}).
		Suffix("RETURNING " + dto.TripExpenseReceiptColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build attach receipt: %w", err)
	}
	return dto.ScanTripExpenseReceipt(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) SoftDeleteTripExpenseReceipt(ctx context.Context, tripID, receiptID, actorUserID uuid.UUID) (*entity.TripExpenseReceipt, error) {
	query, args, err := r.db.Builder.
		Update("trip_expense_receipts").
		Set("status", string(entity.ReceiptStatusDeleted)).
		Set("deleted_at", sq.Expr("NOW()")).
		Set("deleted_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(receiptID), "deleted_at": nil}).
		Where(sq.NotEq{"status": string(entity.ReceiptStatusDeleted)}).
		Suffix("RETURNING " + dto.TripExpenseReceiptColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build soft delete receipt: %w", err)
	}
	return dto.ScanTripExpenseReceipt(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CreateReceiptOCRResult(ctx context.Context, result *entity.ReceiptOCRResult) (*entity.ReceiptOCRResult, error) {
	query, args, err := r.db.Builder.
		Insert("receipt_ocr_results").
		Columns(dto.ReceiptOCRResultInsertColumns()...).
		Values(dto.ReceiptOCRResultInsertValues(result)...).
		Suffix("RETURNING " + dto.ReceiptOCRResultColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create receipt OCR result: %w", err)
	}
	return dto.ScanReceiptOCRResult(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetLatestReceiptOCRResult(ctx context.Context, tripID, receiptID uuid.UUID) (*entity.ReceiptOCRResult, error) {
	query, args, err := r.db.Builder.
		Select(dto.ReceiptOCRResultColumns).
		From("receipt_ocr_results").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "receipt_id": dto.IDArg(receiptID)}).
		OrderBy("created_at DESC", "id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get latest receipt OCR result: %w", err)
	}
	return dto.ScanReceiptOCRResult(r.db.QueryRow(ctx, query, args...))
}
