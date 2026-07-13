package entity

import (
	"time"

	"github.com/google/uuid"
)

type ExpenseCategory string

const (
	ExpenseCategoryTransport     ExpenseCategory = "transport"
	ExpenseCategoryAccommodation ExpenseCategory = "accommodation"
	ExpenseCategoryFood          ExpenseCategory = "food"
	ExpenseCategoryTickets       ExpenseCategory = "tickets"
	ExpenseCategoryActivities    ExpenseCategory = "activities"
	ExpenseCategoryShopping      ExpenseCategory = "shopping"
	ExpenseCategoryFuel          ExpenseCategory = "fuel"
	ExpenseCategoryParking       ExpenseCategory = "parking"
	ExpenseCategoryTolls         ExpenseCategory = "tolls"
	ExpenseCategoryCamping       ExpenseCategory = "camping"
	ExpenseCategoryGroceries     ExpenseCategory = "groceries"
	ExpenseCategoryHealthSafety  ExpenseCategory = "health_safety"
	ExpenseCategoryOther         ExpenseCategory = "other"
)

type ExpenseSplitType string

const (
	ExpenseSplitEqual             ExpenseSplitType = "equal"
	ExpenseSplitSelectedEqual     ExpenseSplitType = "selected_equal"
	ExpenseSplitCustomAmounts     ExpenseSplitType = "custom_amounts"
	ExpenseSplitCustomPercentages ExpenseSplitType = "custom_percentages"
	ExpenseSplitPayerOnly         ExpenseSplitType = "payer_only"
)

type ExpenseStatus string

const (
	ExpenseStatusActive  ExpenseStatus = "active"
	ExpenseStatusDeleted ExpenseStatus = "deleted"
)

type SettlementStatus string

const (
	SettlementStatusPending   SettlementStatus = "pending"
	SettlementStatusPaid      SettlementStatus = "paid"
	SettlementStatusCancelled SettlementStatus = "cancelled"
)

type SettlementSource string

const (
	SettlementSourceCalculated SettlementSource = "calculated"
	SettlementSourceManual     SettlementSource = "manual"
)

type TripExpense struct {
	ID                  uuid.UUID
	TripID              uuid.UUID
	Title               string
	Description         *string
	Amount              float64
	Currency            string
	Category            ExpenseCategory
	ExpenseDate         time.Time
	PaidByUserID        uuid.UUID
	SplitType           ExpenseSplitType
	LinkedDayNumber     *int
	LinkedItemIndex     *int
	LinkedItemID        *string
	LinkedRouteLegID    *string
	LinkedAccommodation bool
	Notes               *string
	Status              ExpenseStatus
	Metadata            map[string]any
	CreatedByUserID     uuid.UUID
	UpdatedByUserID     *uuid.UUID
	DeletedAt           *time.Time
	DeletedByUserID     *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type TripExpenseParticipant struct {
	ID              uuid.UUID
	ExpenseID       uuid.UUID
	TripID          uuid.UUID
	UserID          uuid.UUID
	ShareAmount     *float64
	ShareCurrency   *string
	SharePercentage *float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TripSettlement struct {
	ID                uuid.UUID
	TripID            uuid.UUID
	FromUserID        uuid.UUID
	ToUserID          uuid.UUID
	Amount            float64
	Currency          string
	Status            SettlementStatus
	Source            SettlementSource
	CalculationHash   *string
	PaidAt            *time.Time
	PaidByUserID      *uuid.UUID
	CancelledAt       *time.Time
	CancelledByUserID *uuid.UUID
	Notes             *string
	Metadata          map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
