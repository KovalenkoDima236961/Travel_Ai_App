package entity

import (
	"time"

	"github.com/google/uuid"
)

// CommentStatus is the lifecycle state of an itinerary comment. Comments are
// soft-deleted: a deleted comment keeps its row (and body, for audit) but is
// excluded from normal list/count responses.
type CommentStatus string

const (
	CommentStatusActive  CommentStatus = "active"
	CommentStatusDeleted CommentStatus = "deleted"
)

type CommentTargetType string

const (
	CommentTargetTrip          CommentTargetType = "trip"
	CommentTargetDay           CommentTargetType = "day"
	CommentTargetItineraryItem CommentTargetType = "itinerary_item"
	CommentTargetBudgetItem    CommentTargetType = "budget_item"
	CommentTargetRoute         CommentTargetType = "route"
	CommentTargetAttachment    CommentTargetType = "attachment"
)

// ItineraryComment is a private collaboration comment. Existing itinerary-item
// comments are linked by trip_id + day_number + item_index; newer targets use
// target_type/target_id so the same table can support trip, route, budget and
// attachment discussions without embedding comments in itinerary JSON.
type ItineraryComment struct {
	ID           uuid.UUID
	TripID       uuid.UUID
	DayNumber    int
	ItemIndex    int
	TargetType   CommentTargetType
	TargetID     string
	ParentID     *uuid.UUID
	AuthorUserID uuid.UUID
	Body         string
	Status       CommentStatus
	Mentions     []uuid.UUID
	Attachments  []string
	ResolvedAt   *time.Time
	ResolvedBy   *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	EditedAt     *time.Time
	DeletedAt    *time.Time
}

// ItineraryCommentCount is the number of active comments attached to one
// itinerary item, used to render per-item badges.
type ItineraryCommentCount struct {
	DayNumber int
	ItemIndex int
	Count     int
}
