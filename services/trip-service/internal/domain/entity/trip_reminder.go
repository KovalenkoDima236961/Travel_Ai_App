package entity

import (
	"time"

	"github.com/google/uuid"
)

type ReminderCategory string

const (
	ReminderCategoryDocuments       ReminderCategory = "documents"
	ReminderCategoryPacking         ReminderCategory = "packing"
	ReminderCategoryTransport       ReminderCategory = "transport"
	ReminderCategoryAccommodation   ReminderCategory = "accommodation"
	ReminderCategoryWeather         ReminderCategory = "weather"
	ReminderCategoryActivities      ReminderCategory = "activities"
	ReminderCategoryGroup           ReminderCategory = "group"
	ReminderCategoryChecklist       ReminderCategory = "checklist"
	ReminderCategoryBeforeDeparture ReminderCategory = "before_departure"
	ReminderCategoryRoute           ReminderCategory = "route"
	ReminderCategorySafety          ReminderCategory = "safety"
	ReminderCategoryOther           ReminderCategory = "other"
)

type ReminderPriority string

const (
	ReminderPriorityLow      ReminderPriority = "low"
	ReminderPriorityMedium   ReminderPriority = "medium"
	ReminderPriorityHigh     ReminderPriority = "high"
	ReminderPriorityCritical ReminderPriority = "critical"
)

type ReminderSource string

const (
	ReminderSourceChecklist     ReminderSource = "checklist"
	ReminderSourceRoute         ReminderSource = "route"
	ReminderSourceTransport     ReminderSource = "transport"
	ReminderSourceAccommodation ReminderSource = "accommodation"
	ReminderSourceWeather       ReminderSource = "weather"
	ReminderSourceManual        ReminderSource = "manual"
	ReminderSourceSystem        ReminderSource = "system"
	ReminderSourceRegenerated   ReminderSource = "regenerated"
)

type ReminderStatus string

const (
	ReminderStatusPending   ReminderStatus = "pending"
	ReminderStatusSent      ReminderStatus = "sent"
	ReminderStatusCompleted ReminderStatus = "completed"
	ReminderStatusDisabled  ReminderStatus = "disabled"
	ReminderStatusCancelled ReminderStatus = "cancelled"
	ReminderStatusFailed    ReminderStatus = "failed"
)

type TripReminder struct {
	ID                    uuid.UUID
	TripID                uuid.UUID
	Title                 string
	Description           *string
	Category              ReminderCategory
	Priority              ReminderPriority
	Source                ReminderSource
	Status                ReminderStatus
	TriggerDate           time.Time
	TriggerTime           *string
	Timezone              *string
	RelativeOffsetDays    *int
	AssignedToUserID      *uuid.UUID
	AssignedToDisplayName *string
	ChecklistItemID       *uuid.UUID
	RelatedDayNumber      *int
	RelatedItemIndex      *int
	RelatedItemID         *string
	SentAt                *time.Time
	CompletedAt           *time.Time
	CompletedByUserID     *uuid.UUID
	DisabledAt            *time.Time
	DisabledByUserID      *uuid.UUID
	CancelledAt           *time.Time
	CancelledByUserID     *uuid.UUID
	FailedAt              *time.Time
	FailureReason         *string
	Metadata              map[string]any
	CreatedByUserID       *uuid.UUID
	UpdatedByUserID       *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
	DeletedByUserID       *uuid.UUID
}

type TripReminderFilters struct {
	Status           *ReminderStatus
	Category         *ReminderCategory
	AssignedToUserID *uuid.UUID
	UpcomingOnly     bool
	FromDate         *time.Time
	ToDate           *time.Time
	HighPriorityOnly bool
}
