package request

import (
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type GenerateReminders struct {
	Mode                             string   `json:"mode"`
	Categories                       []string `json:"categories"`
	PreserveManualReminders          *bool    `json:"preserveManualReminders"`
	PreserveCompletedReminders       *bool    `json:"preserveCompletedReminders"`
	ReplaceGeneratedPendingReminders bool     `json:"replaceGeneratedPendingReminders"`
	Instructions                     string   `json:"instructions"`
}

func (r GenerateReminders) ToInput() appdto.GenerateRemindersInput {
	preserveManual := true
	if r.PreserveManualReminders != nil {
		preserveManual = *r.PreserveManualReminders
	}
	preserveCompleted := true
	if r.PreserveCompletedReminders != nil {
		preserveCompleted = *r.PreserveCompletedReminders
	}
	categories := make([]entity.ReminderCategory, 0, len(r.Categories))
	for _, category := range r.Categories {
		categories = append(categories, entity.ReminderCategory(strings.TrimSpace(category)))
	}
	mode := appdto.GenerateRemindersMode(strings.TrimSpace(r.Mode))
	if mode == "" {
		mode = appdto.GenerateRemindersModeFull
	}
	return appdto.GenerateRemindersInput{
		Mode:                             mode,
		Categories:                       categories,
		PreserveManualReminders:          preserveManual,
		PreserveCompletedReminders:       preserveCompleted,
		ReplaceGeneratedPendingReminders: r.ReplaceGeneratedPendingReminders,
		Instructions:                     strings.TrimSpace(r.Instructions),
	}
}

type CreateReminder struct {
	Title              string         `json:"title"`
	Description        *string        `json:"description"`
	Category           string         `json:"category"`
	Priority           string         `json:"priority"`
	TriggerDate        string         `json:"triggerDate"`
	TriggerTime        *string        `json:"triggerTime"`
	Timezone           *string        `json:"timezone"`
	RelativeOffsetDays *int           `json:"relativeOffsetDays"`
	AssignedToUserID   *uuid.UUID     `json:"assignedToUserId"`
	ChecklistItemID    *uuid.UUID     `json:"checklistItemId"`
	RelatedDayNumber   *int           `json:"relatedDayNumber"`
	RelatedItemIndex   *int           `json:"relatedItemIndex"`
	RelatedItemID      *string        `json:"relatedItemId"`
	Metadata           map[string]any `json:"metadata"`
}

func (r CreateReminder) ToInput() (appdto.CreateReminderInput, error) {
	triggerDate, err := parseRequiredReminderDate(r.TriggerDate)
	if err != nil {
		return appdto.CreateReminderInput{}, err
	}
	return appdto.CreateReminderInput{
		Title:              strings.TrimSpace(r.Title),
		Description:        trimStringPtr(r.Description),
		Category:           entity.ReminderCategory(strings.TrimSpace(r.Category)),
		Priority:           entity.ReminderPriority(strings.TrimSpace(r.Priority)),
		TriggerDate:        triggerDate,
		TriggerTime:        trimStringPtr(r.TriggerTime),
		Timezone:           trimStringPtr(r.Timezone),
		RelativeOffsetDays: r.RelativeOffsetDays,
		AssignedToUserID:   r.AssignedToUserID,
		ChecklistItemID:    r.ChecklistItemID,
		RelatedDayNumber:   r.RelatedDayNumber,
		RelatedItemIndex:   r.RelatedItemIndex,
		RelatedItemID:      trimStringPtr(r.RelatedItemID),
		Metadata:           r.Metadata,
	}, nil
}

type UpdateReminder struct {
	Title               *string        `json:"title"`
	Description         *string        `json:"description"`
	ClearDescription    bool           `json:"clearDescription"`
	Category            *string        `json:"category"`
	Priority            *string        `json:"priority"`
	TriggerDate         *string        `json:"triggerDate"`
	TriggerTime         *string        `json:"triggerTime"`
	ClearTriggerTime    bool           `json:"clearTriggerTime"`
	Timezone            *string        `json:"timezone"`
	ClearTimezone       bool           `json:"clearTimezone"`
	RelativeOffsetDays  *int           `json:"relativeOffsetDays"`
	ClearRelativeOffset bool           `json:"clearRelativeOffset"`
	AssignedToUserID    *uuid.UUID     `json:"assignedToUserId"`
	ClearAssignee       bool           `json:"clearAssignee"`
	Metadata            map[string]any `json:"metadata"`
}

func (r UpdateReminder) ToInput() (appdto.UpdateReminderInput, error) {
	triggerDate, err := parseOptionalReminderDate(r.TriggerDate)
	if err != nil {
		return appdto.UpdateReminderInput{}, err
	}
	var category *entity.ReminderCategory
	if r.Category != nil {
		value := entity.ReminderCategory(strings.TrimSpace(*r.Category))
		category = &value
	}
	var priority *entity.ReminderPriority
	if r.Priority != nil {
		value := entity.ReminderPriority(strings.TrimSpace(*r.Priority))
		priority = &value
	}
	return appdto.UpdateReminderInput{
		Title:               trimStringPtr(r.Title),
		Description:         trimStringPtr(r.Description),
		ClearDescription:    r.ClearDescription,
		Category:            category,
		Priority:            priority,
		TriggerDate:         triggerDate,
		TriggerTime:         trimStringPtr(r.TriggerTime),
		ClearTriggerTime:    r.ClearTriggerTime,
		Timezone:            trimStringPtr(r.Timezone),
		ClearTimezone:       r.ClearTimezone,
		RelativeOffsetDays:  r.RelativeOffsetDays,
		ClearRelativeOffset: r.ClearRelativeOffset,
		AssignedToUserID:    r.AssignedToUserID,
		ClearAssignee:       r.ClearAssignee,
		Metadata:            r.Metadata,
	}, nil
}

type ProcessDueReminders struct {
	Now   *time.Time `json:"now"`
	Limit int        `json:"limit"`
}

func (r ProcessDueReminders) ToInput() appdto.ProcessDueRemindersInput {
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now.UTC()
	}
	return appdto.ProcessDueRemindersInput{Now: now, Limit: r.Limit}
}

func parseRequiredReminderDate(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, apperrs.NewInvalidInput("triggerDate is required")
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return time.Time{}, apperrs.NewInvalidInput("triggerDate must be in YYYY-MM-DD format")
	}
	return parsed, nil
}

func parseOptionalReminderDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	parsed, err := parseRequiredReminderDate(*raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
