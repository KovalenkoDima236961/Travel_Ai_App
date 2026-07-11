package request

import (
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type GenerateChecklist struct {
	Mode                 string   `json:"mode"`
	Categories           []string `json:"categories"`
	Instructions         string   `json:"instructions"`
	PreserveCheckedItems *bool    `json:"preserveCheckedItems"`
	PreserveManualItems  *bool    `json:"preserveManualItems"`
	ReplaceAIItems       bool     `json:"replaceAiItems"`
	OutputLanguage       string   `json:"outputLanguage"`
}

func (r GenerateChecklist) ToInput() appdto.GenerateChecklistInput {
	preserveChecked := true
	if r.PreserveCheckedItems != nil {
		preserveChecked = *r.PreserveCheckedItems
	}
	preserveManual := true
	if r.PreserveManualItems != nil {
		preserveManual = *r.PreserveManualItems
	}
	categories := make([]entity.ChecklistCategory, 0, len(r.Categories))
	for _, category := range r.Categories {
		categories = append(categories, entity.ChecklistCategory(strings.TrimSpace(category)))
	}
	mode := appdto.GenerateChecklistMode(strings.TrimSpace(r.Mode))
	if mode == "" {
		mode = appdto.GenerateChecklistModeFull
	}
	outputLanguage := strings.TrimSpace(r.OutputLanguage)
	if outputLanguage == "" {
		outputLanguage = "en"
	}
	return appdto.GenerateChecklistInput{
		Mode:                 mode,
		Categories:           categories,
		Instructions:         strings.TrimSpace(r.Instructions),
		PreserveCheckedItems: preserveChecked,
		PreserveManualItems:  preserveManual,
		ReplaceAIItems:       r.ReplaceAIItems,
		OutputLanguage:       outputLanguage,
	}
}

type CreateChecklistItem struct {
	Title            string         `json:"title"`
	Description      *string        `json:"description"`
	Category         string         `json:"category"`
	ItemType         string         `json:"itemType"`
	Priority         string         `json:"priority"`
	Quantity         *int           `json:"quantity"`
	AssignedToUserID *uuid.UUID     `json:"assignedToUserId"`
	DueDate          *string        `json:"dueDate"`
	Reason           *string        `json:"reason"`
	RelatedDayNumber *int           `json:"relatedDayNumber"`
	RelatedItemIndex *int           `json:"relatedItemIndex"`
	RelatedItemID    *string        `json:"relatedItemId"`
	Metadata         map[string]any `json:"metadata"`
}

func (r CreateChecklistItem) ToInput() (appdto.CreateChecklistItemInput, error) {
	dueDate, err := parseOptionalChecklistDate(r.DueDate)
	if err != nil {
		return appdto.CreateChecklistItemInput{}, err
	}
	return appdto.CreateChecklistItemInput{
		Title:            strings.TrimSpace(r.Title),
		Description:      trimStringPtr(r.Description),
		Category:         entity.ChecklistCategory(strings.TrimSpace(r.Category)),
		ItemType:         entity.ChecklistItemType(strings.TrimSpace(r.ItemType)),
		Priority:         entity.ChecklistPriority(strings.TrimSpace(r.Priority)),
		Quantity:         r.Quantity,
		AssignedToUserID: r.AssignedToUserID,
		DueDate:          dueDate,
		Reason:           trimStringPtr(r.Reason),
		RelatedDayNumber: r.RelatedDayNumber,
		RelatedItemIndex: r.RelatedItemIndex,
		RelatedItemID:    trimStringPtr(r.RelatedItemID),
		Metadata:         r.Metadata,
	}, nil
}

type UpdateChecklistItem struct {
	Title             *string        `json:"title"`
	Description       *string        `json:"description"`
	ClearDescription  bool           `json:"clearDescription"`
	Category          *string        `json:"category"`
	ItemType          *string        `json:"itemType"`
	Priority          *string        `json:"priority"`
	Quantity          *int           `json:"quantity"`
	ClearQuantity     bool           `json:"clearQuantity"`
	AssignedToUserID  *uuid.UUID     `json:"assignedToUserId"`
	ClearAssignee     bool           `json:"clearAssignee"`
	DueDate           *string        `json:"dueDate"`
	ClearDueDate      bool           `json:"clearDueDate"`
	Reason            *string        `json:"reason"`
	ClearReason       bool           `json:"clearReason"`
	RelatedDayNumber  *int           `json:"relatedDayNumber"`
	ClearRelatedDay   bool           `json:"clearRelatedDay"`
	RelatedItemIndex  *int           `json:"relatedItemIndex"`
	ClearRelatedIndex bool           `json:"clearRelatedIndex"`
	RelatedItemID     *string        `json:"relatedItemId"`
	ClearRelatedItem  bool           `json:"clearRelatedItem"`
	SortOrder         *int           `json:"sortOrder"`
	Metadata          map[string]any `json:"metadata"`
}

func (r UpdateChecklistItem) ToInput() (appdto.UpdateChecklistItemInput, error) {
	dueDate, err := parseOptionalChecklistDate(r.DueDate)
	if err != nil {
		return appdto.UpdateChecklistItemInput{}, err
	}
	var category *entity.ChecklistCategory
	if r.Category != nil {
		value := entity.ChecklistCategory(strings.TrimSpace(*r.Category))
		category = &value
	}
	var itemType *entity.ChecklistItemType
	if r.ItemType != nil {
		value := entity.ChecklistItemType(strings.TrimSpace(*r.ItemType))
		itemType = &value
	}
	var priority *entity.ChecklistPriority
	if r.Priority != nil {
		value := entity.ChecklistPriority(strings.TrimSpace(*r.Priority))
		priority = &value
	}
	return appdto.UpdateChecklistItemInput{
		Title:             trimStringPtr(r.Title),
		Description:       trimStringPtr(r.Description),
		ClearDescription:  r.ClearDescription,
		Category:          category,
		ItemType:          itemType,
		Priority:          priority,
		Quantity:          r.Quantity,
		ClearQuantity:     r.ClearQuantity,
		AssignedToUserID:  r.AssignedToUserID,
		ClearAssignee:     r.ClearAssignee,
		DueDate:           dueDate,
		ClearDueDate:      r.ClearDueDate,
		Reason:            trimStringPtr(r.Reason),
		ClearReason:       r.ClearReason,
		RelatedDayNumber:  r.RelatedDayNumber,
		ClearRelatedDay:   r.ClearRelatedDay,
		RelatedItemIndex:  r.RelatedItemIndex,
		ClearRelatedIndex: r.ClearRelatedIndex,
		RelatedItemID:     trimStringPtr(r.RelatedItemID),
		ClearRelatedItem:  r.ClearRelatedItem,
		SortOrder:         r.SortOrder,
		Metadata:          r.Metadata,
	}, nil
}

type ReorderChecklistItems struct {
	ItemIDs []uuid.UUID `json:"itemIds"`
}

func (r ReorderChecklistItems) ToInput() appdto.ChecklistReorderInput {
	return appdto.ChecklistReorderInput{ItemIDs: r.ItemIDs}
}

func parseOptionalChecklistDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, apperrs.NewInvalidInput("dueDate must be in YYYY-MM-DD format")
	}
	return &parsed, nil
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
