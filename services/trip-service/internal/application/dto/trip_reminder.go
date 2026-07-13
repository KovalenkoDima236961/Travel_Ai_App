package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type GenerateRemindersMode string

const (
	GenerateRemindersModeFull       GenerateRemindersMode = "full"
	GenerateRemindersModeAddMissing GenerateRemindersMode = "add_missing"
	GenerateRemindersModeCategory   GenerateRemindersMode = "category"
)

type GenerateRemindersInput struct {
	Mode                             GenerateRemindersMode
	Categories                       []entity.ReminderCategory
	PreserveManualReminders          bool
	PreserveCompletedReminders       bool
	ReplaceGeneratedPendingReminders bool
	Instructions                     string
}

type CreateReminderInput struct {
	Title              string
	Description        *string
	Category           entity.ReminderCategory
	Priority           entity.ReminderPriority
	TriggerDate        time.Time
	TriggerTime        *string
	Timezone           *string
	RelativeOffsetDays *int
	AssignedToUserID   *uuid.UUID
	ChecklistItemID    *uuid.UUID
	RelatedDayNumber   *int
	RelatedItemIndex   *int
	RelatedItemID      *string
	Metadata           map[string]any
}

type UpdateReminderInput struct {
	Title               *string
	Description         *string
	ClearDescription    bool
	Category            *entity.ReminderCategory
	Priority            *entity.ReminderPriority
	TriggerDate         *time.Time
	TriggerTime         *string
	ClearTriggerTime    bool
	Timezone            *string
	ClearTimezone       bool
	RelativeOffsetDays  *int
	ClearRelativeOffset bool
	AssignedToUserID    *uuid.UUID
	ClearAssignee       bool
	Metadata            map[string]any
}

type ReminderListFilters struct {
	Status           *entity.ReminderStatus
	Category         *entity.ReminderCategory
	AssignedToMe     bool
	UpcomingOnly     bool
	HighPriorityOnly bool
	FromDate         *time.Time
	ToDate           *time.Time
}

type ReminderViewResponse struct {
	Reminders []TripReminderDTO `json:"reminders"`
	Summary   ReminderSummary   `json:"summary"`
}

type TripReminderDTO struct {
	ID                    uuid.UUID               `json:"id"`
	TripID                uuid.UUID               `json:"tripId"`
	Title                 string                  `json:"title"`
	Description           *string                 `json:"description"`
	Category              entity.ReminderCategory `json:"category"`
	Priority              entity.ReminderPriority `json:"priority"`
	Source                entity.ReminderSource   `json:"source"`
	Status                entity.ReminderStatus   `json:"status"`
	TriggerDate           string                  `json:"triggerDate"`
	TriggerTime           *string                 `json:"triggerTime"`
	Timezone              *string                 `json:"timezone"`
	RelativeOffsetDays    *int                    `json:"relativeOffsetDays"`
	AssignedToUserID      *uuid.UUID              `json:"assignedToUserId"`
	AssignedToDisplayName *string                 `json:"assignedToDisplayName"`
	ChecklistItemID       *uuid.UUID              `json:"checklistItemId"`
	RelatedDayNumber      *int                    `json:"relatedDayNumber"`
	RelatedItemIndex      *int                    `json:"relatedItemIndex"`
	RelatedItemID         *string                 `json:"relatedItemId,omitempty"`
	SentAt                *time.Time              `json:"sentAt"`
	CompletedAt           *time.Time              `json:"completedAt"`
	CompletedByUserID     *uuid.UUID              `json:"completedByUserId"`
	DisabledAt            *time.Time              `json:"disabledAt"`
	DisabledByUserID      *uuid.UUID              `json:"disabledByUserId"`
	FailureReason         *string                 `json:"failureReason"`
	Metadata              map[string]any          `json:"metadata"`
	CreatedByUserID       *uuid.UUID              `json:"createdByUserId"`
	UpdatedByUserID       *uuid.UUID              `json:"updatedByUserId"`
	CreatedAt             time.Time               `json:"createdAt"`
	UpdatedAt             time.Time               `json:"updatedAt"`
}

type ReminderSummary struct {
	Total               int  `json:"total"`
	Pending             int  `json:"pending"`
	Completed           int  `json:"completed"`
	Overdue             int  `json:"overdue"`
	DueToday            int  `json:"dueToday"`
	HighPriorityPending int  `json:"highPriorityPending"`
	AssignedToMe        int  `json:"assignedToMe"`
	Stale               bool `json:"stale"`
}

type ProcessDueRemindersInput struct {
	Now   time.Time
	Limit int
}

type ProcessDueRemindersResult struct {
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
}

func NewTripReminderDTO(reminder *entity.TripReminder) TripReminderDTO {
	metadata := reminder.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	return TripReminderDTO{
		ID:                    reminder.ID,
		TripID:                reminder.TripID,
		Title:                 reminder.Title,
		Description:           reminder.Description,
		Category:              reminder.Category,
		Priority:              reminder.Priority,
		Source:                reminder.Source,
		Status:                reminder.Status,
		TriggerDate:           reminder.TriggerDate.Format("2006-01-02"),
		TriggerTime:           reminder.TriggerTime,
		Timezone:              reminder.Timezone,
		RelativeOffsetDays:    reminder.RelativeOffsetDays,
		AssignedToUserID:      reminder.AssignedToUserID,
		AssignedToDisplayName: reminder.AssignedToDisplayName,
		ChecklistItemID:       reminder.ChecklistItemID,
		RelatedDayNumber:      reminder.RelatedDayNumber,
		RelatedItemIndex:      reminder.RelatedItemIndex,
		RelatedItemID:         reminder.RelatedItemID,
		SentAt:                reminder.SentAt,
		CompletedAt:           reminder.CompletedAt,
		CompletedByUserID:     reminder.CompletedByUserID,
		DisabledAt:            reminder.DisabledAt,
		DisabledByUserID:      reminder.DisabledByUserID,
		FailureReason:         reminder.FailureReason,
		Metadata:              metadata,
		CreatedByUserID:       reminder.CreatedByUserID,
		UpdatedByUserID:       reminder.UpdatedByUserID,
		CreatedAt:             reminder.CreatedAt,
		UpdatedAt:             reminder.UpdatedAt,
	}
}

func NewTripReminderDTOs(reminders []entity.TripReminder) []TripReminderDTO {
	out := make([]TripReminderDTO, 0, len(reminders))
	for i := range reminders {
		out = append(out, NewTripReminderDTO(&reminders[i]))
	}
	return out
}
