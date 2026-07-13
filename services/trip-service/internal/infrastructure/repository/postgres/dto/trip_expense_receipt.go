package dto

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const TripExpenseReceiptColumns = "id, trip_id, expense_id, status, original_filename, content_type, size_bytes, storage_key, file_sha256, created_by_user_id, updated_by_user_id, deleted_at, deleted_by_user_id, created_at, updated_at"

const ReceiptOCRResultColumns = "id, receipt_id, trip_id, provider, status, merchant, expense_date, amount, currency, tax_amount, category, suggested_title, confidence, field_confidence_json, warnings_json, raw_text, normalized_json, error_message, created_at, updated_at"

func TripExpenseReceiptInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"expense_id",
		"status",
		"original_filename",
		"content_type",
		"size_bytes",
		"storage_key",
		"file_sha256",
		"created_by_user_id",
		"updated_by_user_id",
	}
}

func TripExpenseReceiptInsertValues(receipt *entity.TripExpenseReceipt) []any {
	return []any{
		IDArg(receipt.ID),
		IDArg(receipt.TripID),
		toPgUUIDPtr(receipt.ExpenseID),
		string(receipt.Status),
		receipt.OriginalFilename,
		receipt.ContentType,
		receipt.SizeBytes,
		receipt.StorageKey,
		textPtrArg(receipt.FileSHA256),
		IDArg(receipt.CreatedByUserID),
		toPgUUIDPtr(receipt.UpdatedByUserID),
	}
}

func ReceiptOCRResultInsertColumns() []string {
	return []string{
		"id",
		"receipt_id",
		"trip_id",
		"provider",
		"status",
		"merchant",
		"expense_date",
		"amount",
		"currency",
		"tax_amount",
		"category",
		"suggested_title",
		"confidence",
		"field_confidence_json",
		"warnings_json",
		"raw_text",
		"normalized_json",
		"error_message",
	}
}

func ReceiptOCRResultInsertValues(result *entity.ReceiptOCRResult) []any {
	var category *string
	if result.Category != nil {
		value := string(*result.Category)
		category = &value
	}
	return []any{
		IDArg(result.ID),
		IDArg(result.ReceiptID),
		IDArg(result.TripID),
		string(result.Provider),
		string(result.Status),
		textPtrArg(result.Merchant),
		toPgDate(result.ExpenseDate),
		NumericArg(result.Amount),
		textPtrArg(result.Currency),
		NumericArg(result.TaxAmount),
		textPtrArg(category),
		textPtrArg(result.SuggestedTitle),
		string(result.Confidence),
		jsonConfidenceMap(result.FieldConfidence),
		jsonStringSlice(result.Warnings),
		textPtrArg(result.RawText),
		jsonArg(result.Normalized),
		textPtrArg(result.ErrorMessage),
	}
}

func ScanTripExpenseReceipt(row pgx.Row) (*entity.TripExpenseReceipt, error) {
	var (
		id, tripID, expenseID, createdByUserID, updatedByUserID, deletedByUserID pgtype.UUID
		status, originalFilename, contentType, storageKey                        string
		sizeBytes                                                                int64
		fileSHA256                                                               pgtype.Text
		deletedAt, createdAt, updatedAt                                          pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&tripID,
		&expenseID,
		&status,
		&originalFilename,
		&contentType,
		&sizeBytes,
		&storageKey,
		&fileSHA256,
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
		return nil, fmt.Errorf("scan trip expense receipt: %w", err)
	}
	return &entity.TripExpenseReceipt{
		ID:               uuid.UUID(id.Bytes),
		TripID:           uuid.UUID(tripID.Bytes),
		ExpenseID:        fromPgUUID(expenseID),
		Status:           entity.ReceiptStatus(status),
		OriginalFilename: originalFilename,
		ContentType:      contentType,
		SizeBytes:        sizeBytes,
		StorageKey:       storageKey,
		FileSHA256:       textPtr(fileSHA256),
		CreatedByUserID:  uuid.UUID(createdByUserID.Bytes),
		UpdatedByUserID:  fromPgUUID(updatedByUserID),
		DeletedAt:        timestampPtr(deletedAt),
		DeletedByUserID:  fromPgUUID(deletedByUserID),
		CreatedAt:        createdAt.Time,
		UpdatedAt:        updatedAt.Time,
	}, nil
}

func ScanTripExpenseReceiptRows(rows pgx.Rows) ([]entity.TripExpenseReceipt, error) {
	receipts := make([]entity.TripExpenseReceipt, 0)
	for rows.Next() {
		receipt, err := ScanTripExpenseReceipt(rows)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, *receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip expense receipts: %w", err)
	}
	return receipts, nil
}

func ScanReceiptOCRResult(row pgx.Row) (*entity.ReceiptOCRResult, error) {
	var (
		id, receiptID, tripID                                         pgtype.UUID
		provider, status, confidence                                  string
		merchant, currency, category, suggestedTitle, rawText, errMsg pgtype.Text
		expenseDate                                                   pgtype.Date
		amount, taxAmount                                             pgtype.Numeric
		fieldConfidenceRaw, warningsRaw, normalizedRaw                []byte
		createdAt, updatedAt                                          pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&receiptID,
		&tripID,
		&provider,
		&status,
		&merchant,
		&expenseDate,
		&amount,
		&currency,
		&taxAmount,
		&category,
		&suggestedTitle,
		&confidence,
		&fieldConfidenceRaw,
		&warningsRaw,
		&rawText,
		&normalizedRaw,
		&errMsg,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan receipt OCR result: %w", err)
	}
	fieldConfidence, err := unmarshalConfidenceMap(fieldConfidenceRaw)
	if err != nil {
		return nil, err
	}
	warnings, err := unmarshalStringSlice(warningsRaw, "receipt OCR warnings")
	if err != nil {
		return nil, err
	}
	normalized, err := unmarshalMap(normalizedRaw, "receipt OCR normalized")
	if err != nil {
		return nil, err
	}
	var categoryPtr *entity.ExpenseCategory
	if category.Valid {
		value := entity.ExpenseCategory(category.String)
		categoryPtr = &value
	}
	return &entity.ReceiptOCRResult{
		ID:              uuid.UUID(id.Bytes),
		ReceiptID:       uuid.UUID(receiptID.Bytes),
		TripID:          uuid.UUID(tripID.Bytes),
		Provider:        entity.ReceiptOCRProvider(provider),
		Status:          entity.ReceiptStatus(status),
		Merchant:        textPtr(merchant),
		ExpenseDate:     fromPgDate(expenseDate),
		Amount:          fromPgNumeric(amount),
		Currency:        textPtr(currency),
		TaxAmount:       fromPgNumeric(taxAmount),
		Category:        categoryPtr,
		SuggestedTitle:  textPtr(suggestedTitle),
		Confidence:      entity.ReceiptOCRConfidence(confidence),
		FieldConfidence: fieldConfidence,
		Warnings:        warnings,
		RawText:         textPtr(rawText),
		Normalized:      normalized,
		ErrorMessage:    textPtr(errMsg),
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
	}, nil
}

func ScanReceiptOCRResultRows(rows pgx.Rows) ([]entity.ReceiptOCRResult, error) {
	results := make([]entity.ReceiptOCRResult, 0)
	for rows.Next() {
		result, err := ScanReceiptOCRResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate receipt OCR results: %w", err)
	}
	return results, nil
}

func jsonConfidenceMap(value map[string]entity.ReceiptOCRConfidence) []byte {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

func jsonStringSlice(value []string) []byte {
	if value == nil {
		value = []string{}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("[]")
	}
	return raw
}

func unmarshalConfidenceMap(raw []byte) (map[string]entity.ReceiptOCRConfidence, error) {
	if len(raw) == 0 {
		return map[string]entity.ReceiptOCRConfidence{}, nil
	}
	var out map[string]entity.ReceiptOCRConfidence
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal receipt OCR field confidence: %w", err)
	}
	if out == nil {
		out = map[string]entity.ReceiptOCRConfidence{}
	}
	return out, nil
}

func unmarshalStringSlice(raw []byte, label string) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", label, err)
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}
