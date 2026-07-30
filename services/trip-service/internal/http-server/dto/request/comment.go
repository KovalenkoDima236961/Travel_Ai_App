package request

import (
	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// CreateComment is the JSON body accepted by POST /trips/{id}/comments. Body
// length/content rules are enforced by the service so they stay unit-testable;
// the handler only applies the structural validation tags.
type CreateComment struct {
	DayNumber       int                      `json:"dayNumber" validate:"gte=0"`
	ItemIndex       int                      `json:"itemIndex" validate:"gte=0"`
	TargetType      entity.CommentTargetType `json:"targetType,omitempty"`
	TargetID        string                   `json:"targetId,omitempty"`
	ParentCommentID *uuid.UUID               `json:"parentCommentId,omitempty"`
	MentionUserIDs  []uuid.UUID              `json:"mentionUserIds,omitempty"`
	Body            string                   `json:"body" validate:"required"`
}

func (r CreateComment) ToInput() appdto.CreateCommentInput {
	return appdto.CreateCommentInput{
		DayNumber:       r.DayNumber,
		ItemIndex:       r.ItemIndex,
		TargetType:      r.TargetType,
		TargetID:        r.TargetID,
		ParentCommentID: r.ParentCommentID,
		MentionUserIDs:  append([]uuid.UUID(nil), r.MentionUserIDs...),
		Body:            r.Body,
	}
}

// UpdateComment is the JSON body accepted by PATCH /trips/{id}/comments/{commentId}.
type UpdateComment struct {
	Body string `json:"body" validate:"required"`
}

func (r UpdateComment) ToInput() appdto.UpdateCommentInput {
	return appdto.UpdateCommentInput{Body: r.Body}
}
