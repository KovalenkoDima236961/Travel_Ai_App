package dto

import (
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type MoneyAmount struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type ExpenseParticipant struct {
	UserID          uuid.UUID   `json:"userId"`
	DisplayName     string      `json:"displayName"`
	ShareAmount     MoneyAmount `json:"shareAmount"`
	SharePercentage *float64    `json:"sharePercentage,omitempty"`
}

type LinkedItineraryRef struct {
	DayNumber int     `json:"dayNumber"`
	ItemIndex int     `json:"itemIndex"`
	ItemID    *string `json:"itemId,omitempty"`
}

type TripExpense struct {
	ID                  uuid.UUID               `json:"id"`
	TripID              uuid.UUID               `json:"tripId"`
	Title               string                  `json:"title"`
	Description         *string                 `json:"description,omitempty"`
	Amount              MoneyAmount             `json:"amount"`
	Category            entity.ExpenseCategory  `json:"category"`
	ExpenseDate         string                  `json:"expenseDate"`
	PaidByUserID        uuid.UUID               `json:"paidByUserId"`
	PaidByDisplayName   string                  `json:"paidByDisplayName"`
	SplitType           entity.ExpenseSplitType `json:"splitType"`
	Participants        []ExpenseParticipant    `json:"participants"`
	LinkedItinerary     *LinkedItineraryRef     `json:"linkedItinerary,omitempty"`
	LinkedRouteLegID    *string                 `json:"linkedRouteLegId,omitempty"`
	LinkedAccommodation bool                    `json:"linkedAccommodation"`
	Notes               *string                 `json:"notes,omitempty"`
	Metadata            map[string]any          `json:"metadata"`
	ReceiptCount        int                     `json:"receiptCount"`
	HasReceipt          bool                    `json:"hasReceipt"`
	LatestReceiptStatus *entity.ReceiptStatus   `json:"latestReceiptStatus,omitempty"`
	Receipts            []ExpenseReceiptSummary `json:"receipts"`
	CreatedByUserID     uuid.UUID               `json:"createdByUserId"`
	CreatedAt           time.Time               `json:"createdAt"`
	UpdatedAt           time.Time               `json:"updatedAt"`
}

type TripExpensesResponse struct {
	Items      []TripExpense `json:"items"`
	NextOffset *int          `json:"nextOffset,omitempty"`
}

type ExpenseCustomAmount struct {
	UserID   uuid.UUID
	Amount   float64
	Currency string
}

type ExpenseCustomPercentage struct {
	UserID     uuid.UUID
	Percentage float64
}

type CreateExpenseInput struct {
	Title               string
	Description         *string
	Amount              MoneyAmount
	Category            entity.ExpenseCategory
	ExpenseDate         time.Time
	PaidByUserID        uuid.UUID
	SplitType           entity.ExpenseSplitType
	ParticipantUserIDs  []uuid.UUID
	CustomShares        []ExpenseCustomAmount
	CustomPercentages   []ExpenseCustomPercentage
	LinkedItinerary     *LinkedItineraryRef
	LinkedRouteLegID    *string
	LinkedAccommodation bool
	Notes               *string
	Metadata            map[string]any
}

type UpdateExpenseInput struct {
	Title                 *string
	Description           *string
	ClearDescription      bool
	Amount                *MoneyAmount
	Category              *entity.ExpenseCategory
	ExpenseDate           *time.Time
	PaidByUserID          *uuid.UUID
	SplitType             *entity.ExpenseSplitType
	ParticipantUserIDs    []uuid.UUID
	ParticipantUserIDsSet bool
	CustomShares          []ExpenseCustomAmount
	CustomSharesSet       bool
	CustomPercentages     []ExpenseCustomPercentage
	CustomPercentagesSet  bool
	LinkedItinerary       *LinkedItineraryRef
	LinkedItinerarySet    bool
	LinkedRouteLegID      *string
	LinkedRouteLegIDSet   bool
	LinkedAccommodation   *bool
	Notes                 *string
	ClearNotes            bool
	Metadata              map[string]any
}

type ListExpensesInput struct {
	Category     *entity.ExpenseCategory
	PaidByUserID *uuid.UUID
	FromDate     *time.Time
	ToDate       *time.Time
	LinkedOnly   bool
	Limit        int
	Offset       int
}

type ExpenseCategoryTotal struct {
	Category entity.ExpenseCategory `json:"category"`
	Amount   MoneyAmount            `json:"amount"`
}

type ExpensePayerTotal struct {
	UserID      uuid.UUID   `json:"userId"`
	DisplayName string      `json:"displayName"`
	Paid        MoneyAmount `json:"paid"`
}

type ExpenseBalance struct {
	UserID               uuid.UUID   `json:"userId"`
	DisplayName          string      `json:"displayName"`
	Paid                 MoneyAmount `json:"paid"`
	Share                MoneyAmount `json:"share"`
	Net                  MoneyAmount `json:"net"`
	NetBeforeSettlements MoneyAmount `json:"netBeforeSettlements"`
	SettledAmount        MoneyAmount `json:"settledAmount"`
	NetOutstanding       MoneyAmount `json:"netOutstanding"`
	Status               string      `json:"status"`
}

type PlannedVsActual struct {
	Difference  MoneyAmount `json:"difference"`
	PercentUsed float64     `json:"percentUsed"`
}

type SettlementSummary struct {
	PendingCount int         `json:"pendingCount"`
	PaidCount    int         `json:"paidCount"`
	TotalPending MoneyAmount `json:"totalPending"`
}

type ExpenseSummary struct {
	TripID                 uuid.UUID              `json:"tripId"`
	ExpenseCount           int                    `json:"expenseCount"`
	Currency               string                 `json:"currency"`
	ActualTotal            MoneyAmount            `json:"actualTotal"`
	EstimatedTotal         *MoneyAmount           `json:"estimatedTotal,omitempty"`
	PlannedVsActual        *PlannedVsActual       `json:"plannedVsActual,omitempty"`
	OriginalCurrencyTotals []MoneyAmount          `json:"originalCurrencyTotals"`
	ByCategory             []ExpenseCategoryTotal `json:"byCategory"`
	ByPayer                []ExpensePayerTotal    `json:"byPayer"`
	Balances               []ExpenseBalance       `json:"balances"`
	ConversionWarnings     []string               `json:"conversionWarnings"`
	SettlementSummary      SettlementSummary      `json:"settlementSummary"`
}

type SettlementSuggestion struct {
	ID              string                  `json:"id"`
	FromUserID      uuid.UUID               `json:"fromUserId"`
	FromDisplayName string                  `json:"fromDisplayName"`
	ToUserID        uuid.UUID               `json:"toUserId"`
	ToDisplayName   string                  `json:"toDisplayName"`
	Amount          MoneyAmount             `json:"amount"`
	Status          entity.SettlementStatus `json:"status"`
	Source          entity.SettlementSource `json:"source"`
	CalculationHash string                  `json:"calculationHash,omitempty"`
}

type TripSettlement struct {
	ID                uuid.UUID               `json:"id"`
	TripID            uuid.UUID               `json:"tripId"`
	FromUserID        uuid.UUID               `json:"fromUserId"`
	FromDisplayName   string                  `json:"fromDisplayName"`
	ToUserID          uuid.UUID               `json:"toUserId"`
	ToDisplayName     string                  `json:"toDisplayName"`
	Amount            MoneyAmount             `json:"amount"`
	Status            entity.SettlementStatus `json:"status"`
	Source            entity.SettlementSource `json:"source"`
	PaidAt            *time.Time              `json:"paidAt,omitempty"`
	PaidByUserID      *uuid.UUID              `json:"paidByUserId,omitempty"`
	CancelledAt       *time.Time              `json:"cancelledAt,omitempty"`
	CancelledByUserID *uuid.UUID              `json:"cancelledByUserId,omitempty"`
	Notes             *string                 `json:"notes,omitempty"`
	CreatedAt         time.Time               `json:"createdAt"`
	UpdatedAt         time.Time               `json:"updatedAt"`
}

type SettlementsResponse struct {
	Currency        string                 `json:"currency"`
	Suggestions     []SettlementSuggestion `json:"suggestions"`
	PaidSettlements []TripSettlement       `json:"paidSettlements"`
	Warnings        []string               `json:"warnings"`
}

type MarkSettlementPaidInput struct {
	Notes *string
}

type ExpenseReceiptSummary struct {
	ID               uuid.UUID                    `json:"id"`
	OriginalFilename string                       `json:"originalFilename"`
	ContentType      string                       `json:"contentType"`
	Status           entity.ReceiptStatus         `json:"status"`
	OCRConfidence    *entity.ReceiptOCRConfidence `json:"ocrConfidence,omitempty"`
	CreatedAt        time.Time                    `json:"createdAt"`
}

type ReceiptOCRResult struct {
	Merchant        *string                                `json:"merchant"`
	ExpenseDate     *string                                `json:"expenseDate"`
	Amount          *MoneyAmount                           `json:"amount"`
	TaxAmount       *MoneyAmount                           `json:"taxAmount"`
	Category        *entity.ExpenseCategory                `json:"category"`
	SuggestedTitle  *string                                `json:"suggestedTitle"`
	Confidence      entity.ReceiptOCRConfidence            `json:"confidence"`
	FieldConfidence map[string]entity.ReceiptOCRConfidence `json:"fieldConfidence"`
	Warnings        []string                               `json:"warnings"`
	RawText         *string                                `json:"rawText,omitempty"`
}

type ExpenseReceipt struct {
	ID               uuid.UUID            `json:"id"`
	TripID           uuid.UUID            `json:"tripId"`
	ExpenseID        *uuid.UUID           `json:"expenseId"`
	Status           entity.ReceiptStatus `json:"status"`
	OriginalFilename string               `json:"originalFilename"`
	ContentType      string               `json:"contentType"`
	SizeBytes        int64                `json:"sizeBytes"`
	PreviewURL       string               `json:"previewUrl"`
	OCRResult        *ReceiptOCRResult    `json:"ocrResult"`
	CreatedByUserID  uuid.UUID            `json:"createdByUserId"`
	CreatedAt        time.Time            `json:"createdAt"`
	UpdatedAt        time.Time            `json:"updatedAt"`
}

type TripReceiptsResponse struct {
	Receipts   []ExpenseReceipt `json:"receipts"`
	NextOffset *int             `json:"nextOffset,omitempty"`
}

type ListReceiptsInput struct {
	ExpenseID      *uuid.UUID
	ExpenseIDs     []uuid.UUID
	Status         *entity.ReceiptStatus
	UnlinkedOnly   bool
	IncludeRawText bool
	Limit          int
	Offset         int
}

type UploadReceiptInput struct {
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
	ExpenseID        *uuid.UUID
	RunOCR           bool
	File             io.Reader
}

type ExtractReceiptInput struct {
	Provider *entity.ReceiptOCRProvider
}

type AttachReceiptInput struct {
	ReceiptID uuid.UUID
}
