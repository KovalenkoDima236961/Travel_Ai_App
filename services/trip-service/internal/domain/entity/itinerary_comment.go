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

// ItineraryComment is a comment attached to a specific itinerary item, linked by
// trip_id + day_number + item_index. Comments are stored in their own table and
// are never embedded in the itinerary JSON.
type ItineraryComment struct {
	ID           uuid.UUID
	TripID       uuid.UUID
	DayNumber    int
	ItemIndex    int
	AuthorUserID uuid.UUID
	Body         string
	Status       CommentStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// ItineraryCommentCount is the number of active comments attached to one
// itinerary item, used to render per-item badges.
type ItineraryCommentCount struct {
	DayNumber int
	ItemIndex int
	Count     int
}
