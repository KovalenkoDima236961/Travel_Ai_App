package response

import (
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// ItineraryComment is the JSON representation of a comment returned to clients.
// authorDisplayName/authorEmail are intentionally omitted in v1 (no batch user
// lookup); clients render "You" for the author and "Collaborator" otherwise.
type ItineraryComment struct {
	ID           uuid.UUID `json:"id"`
	TripID       uuid.UUID `json:"tripId"`
	DayNumber    int       `json:"dayNumber"`
	ItemIndex    int       `json:"itemIndex"`
	AuthorUserID uuid.UUID `json:"authorUserId"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	IsAuthor     bool      `json:"isAuthor"`
	CanEdit      bool      `json:"canEdit"`
	CanDelete    bool      `json:"canDelete"`
}

// ListComments is the envelope returned by the comment list endpoints.
type ListComments struct {
	Items []ItineraryComment `json:"items"`
}

// CommentCount is the active comment count for one itinerary item.
type CommentCount struct {
	DayNumber int `json:"dayNumber"`
	ItemIndex int `json:"itemIndex"`
	Count     int `json:"count"`
}

// CommentCounts is the envelope returned by GET /trips/{id}/comments/counts.
type CommentCounts struct {
	Items []CommentCount `json:"items"`
}

// NewItineraryComment maps a comment + permissions to its API representation.
func NewItineraryComment(info appdto.ItineraryCommentInfo) ItineraryComment {
	c := info.Comment
	return ItineraryComment{
		ID:           c.ID,
		TripID:       c.TripID,
		DayNumber:    c.DayNumber,
		ItemIndex:    c.ItemIndex,
		AuthorUserID: c.AuthorUserID,
		Body:         c.Body,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		IsAuthor:     info.IsAuthor,
		CanEdit:      info.CanEdit,
		CanDelete:    info.CanDelete,
	}
}

// NewListComments maps a page of comments to the list envelope. Items is always
// a (possibly empty) slice so it serialises as [] rather than null.
func NewListComments(infos []appdto.ItineraryCommentInfo) ListComments {
	items := make([]ItineraryComment, 0, len(infos))
	for _, info := range infos {
		items = append(items, NewItineraryComment(info))
	}
	return ListComments{Items: items}
}

// NewCommentCounts maps grouped counts to the counts envelope.
func NewCommentCounts(counts []entity.ItineraryCommentCount) CommentCounts {
	items := make([]CommentCount, 0, len(counts))
	for _, count := range counts {
		items = append(items, CommentCount{
			DayNumber: count.DayNumber,
			ItemIndex: count.ItemIndex,
			Count:     count.Count,
		})
	}
	return CommentCounts{Items: items}
}
