package entity

import (
	"time"

	"github.com/google/uuid"
)

type ChecklistStatus string

const (
	ChecklistStatusActive   ChecklistStatus = "active"
	ChecklistStatusArchived ChecklistStatus = "archived"
)

type ChecklistCategory string

const (
	ChecklistCategoryDocuments       ChecklistCategory = "documents"
	ChecklistCategoryClothing        ChecklistCategory = "clothing"
	ChecklistCategoryElectronics     ChecklistCategory = "electronics"
	ChecklistCategoryHealthSafety    ChecklistCategory = "health_safety"
	ChecklistCategoryTransport       ChecklistCategory = "transport"
	ChecklistCategoryAccommodation   ChecklistCategory = "accommodation"
	ChecklistCategoryActivities      ChecklistCategory = "activities"
	ChecklistCategoryFoodWater       ChecklistCategory = "food_water"
	ChecklistCategoryMoney           ChecklistCategory = "money"
	ChecklistCategoryBeforeDeparture ChecklistCategory = "before_departure"
	ChecklistCategoryGroupItems      ChecklistCategory = "group_items"
	ChecklistCategoryCampingHiking   ChecklistCategory = "camping_hiking"
	ChecklistCategoryWeather         ChecklistCategory = "weather"
	ChecklistCategoryOther           ChecklistCategory = "other"
)

type ChecklistPriority string

const (
	ChecklistPriorityLow      ChecklistPriority = "low"
	ChecklistPriorityMedium   ChecklistPriority = "medium"
	ChecklistPriorityHigh     ChecklistPriority = "high"
	ChecklistPriorityCritical ChecklistPriority = "critical"
)

type ChecklistSource string

const (
	ChecklistSourceAI          ChecklistSource = "ai"
	ChecklistSourceManual      ChecklistSource = "manual"
	ChecklistSourceTemplate    ChecklistSource = "template"
	ChecklistSourceRegenerated ChecklistSource = "regenerated"
	ChecklistSourceSystem      ChecklistSource = "system"
)

type ChecklistItemType string

const (
	ChecklistItemTypePacking         ChecklistItemType = "packing"
	ChecklistItemTypePreparation     ChecklistItemType = "preparation"
	ChecklistItemTypeBookingCheck    ChecklistItemType = "booking_check"
	ChecklistItemTypeDocument        ChecklistItemType = "document"
	ChecklistItemTypeSharedGroupItem ChecklistItemType = "shared_group_item"
	ChecklistItemTypeReminder        ChecklistItemType = "reminder"
	ChecklistItemTypeSafetyCheck     ChecklistItemType = "safety_check"
	ChecklistItemTypeOther           ChecklistItemType = "other"
)

type TripChecklist struct {
	ID                             uuid.UUID
	TripID                         uuid.UUID
	Status                         ChecklistStatus
	Title                          string
	Summary                        *string
	GeneratedFromItineraryRevision *int
	GeneratedFromRouteRevision     *int
	GeneratedByUserID              *uuid.UUID
	CreatedByUserID                uuid.UUID
	UpdatedByUserID                *uuid.UUID
	Metadata                       map[string]any
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
	ArchivedAt                     *time.Time
	ArchivedByUserID               *uuid.UUID
	Items                          []TripChecklistItem
}

type TripChecklistItem struct {
	ID                    uuid.UUID
	ChecklistID           uuid.UUID
	TripID                uuid.UUID
	Title                 string
	Description           *string
	Category              ChecklistCategory
	ItemType              ChecklistItemType
	Priority              ChecklistPriority
	Quantity              *int
	AssignedToUserID      *uuid.UUID
	AssignedToDisplayName *string
	DueDate               *time.Time
	Checked               bool
	CheckedAt             *time.Time
	CheckedByUserID       *uuid.UUID
	Source                ChecklistSource
	Reason                *string
	RelatedDayNumber      *int
	RelatedItemIndex      *int
	RelatedItemID         *string
	SortOrder             int
	Metadata              map[string]any
	CreatedByUserID       *uuid.UUID
	UpdatedByUserID       *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
	DeletedByUserID       *uuid.UUID
}
