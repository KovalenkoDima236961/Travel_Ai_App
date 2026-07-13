package receipts

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type MockOCRProvider struct{}

func NewMockOCRProvider() *MockOCRProvider {
	return &MockOCRProvider{}
}

func (p *MockOCRProvider) Name() entity.ReceiptOCRProvider {
	return entity.ReceiptOCRProviderMock
}

func (p *MockOCRProvider) Extract(ctx context.Context, file io.Reader, metadata OCRMetadata, trip OCRTripContext) (*entity.ReceiptOCRResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if file != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(file, 4096))
	}
	currency := strings.ToUpper(strings.TrimSpace(trip.DefaultCurrency))
	if currency == "" {
		currency = "EUR"
	}
	name := strings.ToLower(metadata.OriginalFilename)
	result := &entity.ReceiptOCRResult{
		Provider:        entity.ReceiptOCRProviderMock,
		Status:          entity.ReceiptStatusExtracted,
		Confidence:      entity.ReceiptOCRConfidenceLow,
		FieldConfidence: map[string]entity.ReceiptOCRConfidence{},
		Warnings:        []string{},
		RawText:         stringPtr(fmt.Sprintf("Mock OCR fixture for %s", metadata.OriginalFilename)),
		Normalized:      map[string]any{"source": "filename"},
	}
	expenseDate := time.Now().UTC().Truncate(24 * time.Hour)
	result.ExpenseDate = &expenseDate
	set := func(merchant string, category entity.ExpenseCategory, amount float64, confidence entity.ReceiptOCRConfidence) {
		result.Merchant = stringPtr(merchant)
		result.SuggestedTitle = stringPtr(merchant)
		result.Category = &category
		result.Amount = &amount
		result.Currency = &currency
		result.Confidence = confidence
		result.FieldConfidence = map[string]entity.ReceiptOCRConfidence{
			"merchant": entity.ReceiptOCRConfidenceHigh,
			"date":     entity.ReceiptOCRConfidenceMedium,
			"amount":   entity.ReceiptOCRConfidenceHigh,
			"currency": entity.ReceiptOCRConfidenceHigh,
			"category": entity.ReceiptOCRConfidenceMedium,
		}
		result.Warnings = []string{"Verify the date before creating expense."}
	}
	switch {
	case containsAny(name, "train", "obb", "rail"):
		set("Train Tickets", entity.ExpenseCategoryTransport, 72.00, entity.ReceiptOCRConfidenceHigh)
	case containsAny(name, "restaurant", "food", "cafe"):
		set("Restaurant", entity.ExpenseCategoryFood, 38.50, entity.ReceiptOCRConfidenceHigh)
	case containsAny(name, "fuel", "gas"):
		set("Fuel Station", entity.ExpenseCategoryFuel, 55.00, entity.ReceiptOCRConfidenceHigh)
	case strings.Contains(name, "parking"):
		set("Parking", entity.ExpenseCategoryParking, 12.00, entity.ReceiptOCRConfidenceHigh)
	case containsAny(name, "hotel", "accommodation"):
		set("Accommodation", entity.ExpenseCategoryAccommodation, 180.00, entity.ReceiptOCRConfidenceHigh)
	case containsAny(name, "museum", "ticket"):
		set("Museum Tickets", entity.ExpenseCategoryTickets, 24.00, entity.ReceiptOCRConfidenceHigh)
	default:
		result.Warnings = []string{"No reliable amount was detected. Enter the expense manually."}
		result.FieldConfidence = map[string]entity.ReceiptOCRConfidence{
			"merchant": entity.ReceiptOCRConfidenceLow,
			"date":     entity.ReceiptOCRConfidenceLow,
			"amount":   entity.ReceiptOCRConfidenceLow,
			"currency": entity.ReceiptOCRConfidenceLow,
			"category": entity.ReceiptOCRConfidenceLow,
		}
	}
	return result, nil
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func stringPtr(value string) *string {
	return &value
}
