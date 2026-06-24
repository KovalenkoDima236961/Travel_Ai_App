package dto

import (
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// CreateCommentInput is the validated, application-level payload for creating a
// comment on an itinerary item.
type CreateCommentInput struct {
	DayNumber int
	ItemIndex int
	Body      string
}

// UpdateCommentInput is the application-level payload for editing a comment body.
type UpdateCommentInput struct {
	Body string
}

// ItineraryCommentInfo wraps a comment with the requesting user's permissions.
// Permission flags are computed by the service from trip access + authorship so
// the transport layer (and clients) never re-derive them.
type ItineraryCommentInfo struct {
	Comment   entity.ItineraryComment
	IsAuthor  bool
	CanEdit   bool
	CanDelete bool
}
