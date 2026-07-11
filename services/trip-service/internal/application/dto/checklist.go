package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type GenerateChecklistMode string

const (
	GenerateChecklistModeFull       GenerateChecklistMode = "full"
	GenerateChecklistModeAddMissing GenerateChecklistMode = "add_missing"
	GenerateChecklistModeCategory   GenerateChecklistMode = "category"
)

type GenerateChecklistInput struct {
	Mode                 GenerateChecklistMode
	Categories           []entity.ChecklistCategory
	Instructions         string
	PreserveCheckedItems bool
	PreserveManualItems  bool
	ReplaceAIItems       bool
	OutputLanguage       string
}

type CreateChecklistItemInput struct {
	Title            string
	Description      *string
	Category         entity.ChecklistCategory
	ItemType         entity.ChecklistItemType
	Priority         entity.ChecklistPriority
	Quantity         *int
	AssignedToUserID *uuid.UUID
	DueDate          *time.Time
	Reason           *string
	RelatedDayNumber *int
	RelatedItemIndex *int
	RelatedItemID    *string
	Metadata         map[string]any
}

type UpdateChecklistItemInput struct {
	Title             *string
	Description       *string
	ClearDescription  bool
	Category          *entity.ChecklistCategory
	ItemType          *entity.ChecklistItemType
	Priority          *entity.ChecklistPriority
	Quantity          *int
	ClearQuantity     bool
	AssignedToUserID  *uuid.UUID
	ClearAssignee     bool
	DueDate           *time.Time
	ClearDueDate      bool
	Reason            *string
	ClearReason       bool
	RelatedDayNumber  *int
	ClearRelatedDay   bool
	RelatedItemIndex  *int
	ClearRelatedIndex bool
	RelatedItemID     *string
	ClearRelatedItem  bool
	SortOrder         *int
	Metadata          map[string]any
}

type ChecklistReorderInput struct {
	ItemIDs []uuid.UUID
}

type ChecklistViewResponse struct {
	Checklist   *TripChecklistDTO `json:"checklist"`
	Summary     *ChecklistSummary `json:"summary,omitempty"`
	CanGenerate bool              `json:"canGenerate"`
}

type TripChecklistDTO struct {
	ID                             uuid.UUID              `json:"id"`
	TripID                         uuid.UUID              `json:"tripId"`
	Status                         string                 `json:"status"`
	GeneratedFromRevision          *int                   `json:"generatedFromRevision,omitempty"`
	GeneratedFromItineraryRevision *int                   `json:"generatedFromItineraryRevision,omitempty"`
	GeneratedFromRouteRevision     *int                   `json:"generatedFromRouteRevision,omitempty"`
	Title                          string                 `json:"title"`
	Summary                        *string                `json:"summary,omitempty"`
	CreatedByUserID                uuid.UUID              `json:"createdByUserId"`
	UpdatedAt                      time.Time              `json:"updatedAt"`
	Items                          []TripChecklistItemDTO `json:"items"`
	Metadata                       map[string]any         `json:"metadata,omitempty"`
	CreatedAt                      time.Time              `json:"createdAt"`
}

type TripChecklistItemDTO struct {
	ID                    uuid.UUID                `json:"id"`
	ChecklistID           uuid.UUID                `json:"checklistId"`
	Title                 string                   `json:"title"`
	Description           *string                  `json:"description"`
	Category              entity.ChecklistCategory `json:"category"`
	ItemType              entity.ChecklistItemType `json:"itemType"`
	Priority              entity.ChecklistPriority `json:"priority"`
	Quantity              *int                     `json:"quantity"`
	AssignedToUserID      *uuid.UUID               `json:"assignedToUserId"`
	AssignedToDisplayName *string                  `json:"assignedToDisplayName"`
	DueDate               *string                  `json:"dueDate"`
	Checked               bool                     `json:"checked"`
	CheckedAt             *time.Time               `json:"checkedAt"`
	CheckedByUserID       *uuid.UUID               `json:"checkedByUserId"`
	Source                entity.ChecklistSource   `json:"source"`
	Reason                *string                  `json:"reason"`
	RelatedDayNumber      *int                     `json:"relatedDayNumber"`
	RelatedItemIndex      *int                     `json:"relatedItemIndex"`
	RelatedItemID         *string                  `json:"relatedItemId,omitempty"`
	SortOrder             int                      `json:"sortOrder"`
	Metadata              map[string]any           `json:"metadata"`
	CreatedAt             time.Time                `json:"createdAt"`
	UpdatedAt             time.Time                `json:"updatedAt"`
}

type ChecklistSummary struct {
	TotalItems            int                        `json:"totalItems"`
	CheckedItems          int                        `json:"checkedItems"`
	UncheckedItems        int                        `json:"uncheckedItems"`
	HighPriorityUnchecked int                        `json:"highPriorityUnchecked"`
	AssignedToMe          int                        `json:"assignedToMe"`
	Categories            []ChecklistCategorySummary `json:"categories"`
}

type ChecklistCategorySummary struct {
	Category entity.ChecklistCategory `json:"category"`
	Total    int                      `json:"total"`
	Checked  int                      `json:"checked"`
}

type GeneratedChecklist struct {
	Title    string
	Summary  string
	Items    []GeneratedChecklistItem
	Warnings []string
}

type GeneratedChecklistItem struct {
	Title            string
	Description      string
	Category         entity.ChecklistCategory
	ItemType         entity.ChecklistItemType
	Priority         entity.ChecklistPriority
	Quantity         *int
	Reason           string
	RelatedDayNumber *int
	RelatedItemIndex *int
	RelatedItemID    *string
	Metadata         map[string]any
}

func NewTripChecklistDTO(checklist *entity.TripChecklist) *TripChecklistDTO {
	if checklist == nil {
		return nil
	}
	items := make([]TripChecklistItemDTO, 0, len(checklist.Items))
	for i := range checklist.Items {
		items = append(items, NewTripChecklistItemDTO(&checklist.Items[i]))
	}
	return &TripChecklistDTO{
		ID:                             checklist.ID,
		TripID:                         checklist.TripID,
		Status:                         string(checklist.Status),
		GeneratedFromRevision:          checklist.GeneratedFromItineraryRevision,
		GeneratedFromItineraryRevision: checklist.GeneratedFromItineraryRevision,
		GeneratedFromRouteRevision:     checklist.GeneratedFromRouteRevision,
		Title:                          checklist.Title,
		Summary:                        checklist.Summary,
		CreatedByUserID:                checklist.CreatedByUserID,
		UpdatedAt:                      checklist.UpdatedAt,
		Items:                          items,
		Metadata:                       checklist.Metadata,
		CreatedAt:                      checklist.CreatedAt,
	}
}

func NewTripChecklistItemDTO(item *entity.TripChecklistItem) TripChecklistItemDTO {
	var dueDate *string
	if item.DueDate != nil {
		formatted := item.DueDate.Format("2006-01-02")
		dueDate = &formatted
	}
	metadata := item.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	return TripChecklistItemDTO{
		ID:                    item.ID,
		ChecklistID:           item.ChecklistID,
		Title:                 item.Title,
		Description:           item.Description,
		Category:              item.Category,
		ItemType:              item.ItemType,
		Priority:              item.Priority,
		Quantity:              item.Quantity,
		AssignedToUserID:      item.AssignedToUserID,
		AssignedToDisplayName: item.AssignedToDisplayName,
		DueDate:               dueDate,
		Checked:               item.Checked,
		CheckedAt:             item.CheckedAt,
		CheckedByUserID:       item.CheckedByUserID,
		Source:                item.Source,
		Reason:                item.Reason,
		RelatedDayNumber:      item.RelatedDayNumber,
		RelatedItemIndex:      item.RelatedItemIndex,
		RelatedItemID:         item.RelatedItemID,
		SortOrder:             item.SortOrder,
		Metadata:              metadata,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}
