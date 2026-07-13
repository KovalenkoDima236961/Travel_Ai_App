package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const TripExpenseColumns = "id, trip_id, title, description, amount, currency, category, expense_date, paid_by_user_id, split_type, linked_day_number, linked_item_index, linked_item_id, linked_route_leg_id, linked_accommodation, notes, status, metadata, created_by_user_id, updated_by_user_id, deleted_at, deleted_by_user_id, created_at, updated_at"

const TripExpenseParticipantColumns = "id, expense_id, trip_id, user_id, share_amount, share_currency, share_percentage, created_at, updated_at"

const TripSettlementColumns = "id, trip_id, from_user_id, to_user_id, amount, currency, status, source, calculation_hash, paid_at, paid_by_user_id, cancelled_at, cancelled_by_user_id, notes, metadata, created_at, updated_at"

func TripExpenseInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"title",
		"description",
		"amount",
		"currency",
		"category",
		"expense_date",
		"paid_by_user_id",
		"split_type",
		"linked_day_number",
		"linked_item_index",
		"linked_item_id",
		"linked_route_leg_id",
		"linked_accommodation",
		"notes",
		"status",
		"metadata",
		"created_by_user_id",
		"updated_by_user_id",
	}
}

func TripExpenseInsertValues(expense *entity.TripExpense) []any {
	amount := expense.Amount
	return []any{
		IDArg(expense.ID),
		IDArg(expense.TripID),
		expense.Title,
		textPtrArg(expense.Description),
		NumericArg(&amount),
		expense.Currency,
		string(expense.Category),
		toPgDate(&expense.ExpenseDate),
		IDArg(expense.PaidByUserID),
		string(expense.SplitType),
		intPtrArg(expense.LinkedDayNumber),
		intPtrArg(expense.LinkedItemIndex),
		textPtrArg(expense.LinkedItemID),
		textPtrArg(expense.LinkedRouteLegID),
		expense.LinkedAccommodation,
		textPtrArg(expense.Notes),
		string(expense.Status),
		jsonArg(expense.Metadata),
		IDArg(expense.CreatedByUserID),
		toPgUUIDPtr(expense.UpdatedByUserID),
	}
}

func TripExpenseParticipantInsertColumns() []string {
	return []string{
		"id",
		"expense_id",
		"trip_id",
		"user_id",
		"share_amount",
		"share_currency",
		"share_percentage",
	}
}

func TripExpenseParticipantInsertValues(participant *entity.TripExpenseParticipant) []any {
	return []any{
		IDArg(participant.ID),
		IDArg(participant.ExpenseID),
		IDArg(participant.TripID),
		IDArg(participant.UserID),
		NumericArg(participant.ShareAmount),
		textPtrArg(participant.ShareCurrency),
		NumericArg(participant.SharePercentage),
	}
}

func TripSettlementInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"from_user_id",
		"to_user_id",
		"amount",
		"currency",
		"status",
		"source",
		"calculation_hash",
		"paid_at",
		"paid_by_user_id",
		"notes",
		"metadata",
	}
}

func TripSettlementInsertValues(settlement *entity.TripSettlement) []any {
	amount := settlement.Amount
	return []any{
		IDArg(settlement.ID),
		IDArg(settlement.TripID),
		IDArg(settlement.FromUserID),
		IDArg(settlement.ToUserID),
		NumericArg(&amount),
		settlement.Currency,
		string(settlement.Status),
		string(settlement.Source),
		textPtrArg(settlement.CalculationHash),
		timestampArg(settlement.PaidAt),
		toPgUUIDPtr(settlement.PaidByUserID),
		textPtrArg(settlement.Notes),
		jsonArg(settlement.Metadata),
	}
}

func ScanTripExpense(row pgx.Row) (*entity.TripExpense, error) {
	var (
		id, tripID, paidByUserID, createdByUserID, updatedByUserID, deletedByUserID pgtype.UUID
		title, currency, category, splitType, status                                string
		description, linkedItemID, linkedRouteLegID, notes                          pgtype.Text
		amount                                                                      pgtype.Numeric
		expenseDate                                                                 pgtype.Date
		linkedDayNumber, linkedItemIndex                                            pgtype.Int4
		linkedAccommodation                                                         bool
		metadataRaw                                                                 []byte
		deletedAt, createdAt, updatedAt                                             pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&tripID,
		&title,
		&description,
		&amount,
		&currency,
		&category,
		&expenseDate,
		&paidByUserID,
		&splitType,
		&linkedDayNumber,
		&linkedItemIndex,
		&linkedItemID,
		&linkedRouteLegID,
		&linkedAccommodation,
		&notes,
		&status,
		&metadataRaw,
		&createdByUserID,
		&updatedByUserID,
		&deletedAt,
		&deletedByUserID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip expense: %w", err)
	}
	metadata, err := unmarshalMap(metadataRaw, "trip expense metadata")
	if err != nil {
		return nil, err
	}
	amountValue := fromPgNumeric(amount)
	if amountValue == nil {
		zero := 0.0
		amountValue = &zero
	}
	return &entity.TripExpense{
		ID:                  uuid.UUID(id.Bytes),
		TripID:              uuid.UUID(tripID.Bytes),
		Title:               title,
		Description:         textPtr(description),
		Amount:              *amountValue,
		Currency:            currency,
		Category:            entity.ExpenseCategory(category),
		ExpenseDate:         expenseDate.Time,
		PaidByUserID:        uuid.UUID(paidByUserID.Bytes),
		SplitType:           entity.ExpenseSplitType(splitType),
		LinkedDayNumber:     int4Ptr(linkedDayNumber),
		LinkedItemIndex:     int4Ptr(linkedItemIndex),
		LinkedItemID:        textPtr(linkedItemID),
		LinkedRouteLegID:    textPtr(linkedRouteLegID),
		LinkedAccommodation: linkedAccommodation,
		Notes:               textPtr(notes),
		Status:              entity.ExpenseStatus(status),
		Metadata:            metadata,
		CreatedByUserID:     uuid.UUID(createdByUserID.Bytes),
		UpdatedByUserID:     fromPgUUID(updatedByUserID),
		DeletedAt:           timestampPtr(deletedAt),
		DeletedByUserID:     fromPgUUID(deletedByUserID),
		CreatedAt:           createdAt.Time,
		UpdatedAt:           updatedAt.Time,
	}, nil
}

func ScanTripExpenseRows(rows pgx.Rows) ([]entity.TripExpense, error) {
	expenses := make([]entity.TripExpense, 0)
	for rows.Next() {
		expense, err := ScanTripExpense(rows)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, *expense)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip expenses: %w", err)
	}
	return expenses, nil
}

func ScanTripExpenseParticipant(row pgx.Row) (*entity.TripExpenseParticipant, error) {
	var (
		id, expenseID, tripID, userID pgtype.UUID
		shareAmount                   pgtype.Numeric
		shareCurrency                 pgtype.Text
		sharePercentage               pgtype.Numeric
		createdAt, updatedAt          pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&expenseID,
		&tripID,
		&userID,
		&shareAmount,
		&shareCurrency,
		&sharePercentage,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip expense participant: %w", err)
	}
	return &entity.TripExpenseParticipant{
		ID:              uuid.UUID(id.Bytes),
		ExpenseID:       uuid.UUID(expenseID.Bytes),
		TripID:          uuid.UUID(tripID.Bytes),
		UserID:          uuid.UUID(userID.Bytes),
		ShareAmount:     fromPgNumeric(shareAmount),
		ShareCurrency:   textPtr(shareCurrency),
		SharePercentage: fromPgNumeric(sharePercentage),
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
	}, nil
}

func ScanTripExpenseParticipantRows(rows pgx.Rows) ([]entity.TripExpenseParticipant, error) {
	participants := make([]entity.TripExpenseParticipant, 0)
	for rows.Next() {
		participant, err := ScanTripExpenseParticipant(rows)
		if err != nil {
			return nil, err
		}
		participants = append(participants, *participant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip expense participants: %w", err)
	}
	return participants, nil
}

func ScanTripSettlement(row pgx.Row) (*entity.TripSettlement, error) {
	var (
		id, tripID, fromUserID, toUserID, paidByUserID, cancelledByUserID pgtype.UUID
		amount                                                            pgtype.Numeric
		currency, status, source                                          string
		calculationHash, notes                                            pgtype.Text
		metadataRaw                                                       []byte
		paidAt, cancelledAt, createdAt, updatedAt                         pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&tripID,
		&fromUserID,
		&toUserID,
		&amount,
		&currency,
		&status,
		&source,
		&calculationHash,
		&paidAt,
		&paidByUserID,
		&cancelledAt,
		&cancelledByUserID,
		&notes,
		&metadataRaw,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip settlement: %w", err)
	}
	metadata, err := unmarshalMap(metadataRaw, "trip settlement metadata")
	if err != nil {
		return nil, err
	}
	amountValue := fromPgNumeric(amount)
	if amountValue == nil {
		zero := 0.0
		amountValue = &zero
	}
	return &entity.TripSettlement{
		ID:                uuid.UUID(id.Bytes),
		TripID:            uuid.UUID(tripID.Bytes),
		FromUserID:        uuid.UUID(fromUserID.Bytes),
		ToUserID:          uuid.UUID(toUserID.Bytes),
		Amount:            *amountValue,
		Currency:          currency,
		Status:            entity.SettlementStatus(status),
		Source:            entity.SettlementSource(source),
		CalculationHash:   textPtr(calculationHash),
		PaidAt:            timestampPtr(paidAt),
		PaidByUserID:      fromPgUUID(paidByUserID),
		CancelledAt:       timestampPtr(cancelledAt),
		CancelledByUserID: fromPgUUID(cancelledByUserID),
		Notes:             textPtr(notes),
		Metadata:          metadata,
		CreatedAt:         createdAt.Time,
		UpdatedAt:         updatedAt.Time,
	}, nil
}

func ScanTripSettlementRows(rows pgx.Rows) ([]entity.TripSettlement, error) {
	settlements := make([]entity.TripSettlement, 0)
	for rows.Next() {
		settlement, err := ScanTripSettlement(rows)
		if err != nil {
			return nil, err
		}
		settlements = append(settlements, *settlement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip settlements: %w", err)
	}
	return settlements, nil
}
