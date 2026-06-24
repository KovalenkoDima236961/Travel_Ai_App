package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

const (
	maxCommentBodyLength = 2000
)

// ListComments returns all active comments on a trip the caller can view.
func (s *Service) ListComments(ctx context.Context, tripID uuid.UUID) ([]appdto.ItineraryCommentInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}

	comments, err := s.repo.ListItineraryCommentsByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	return s.toCommentInfos(comments, user.ID, access), nil
}

// ListItemComments returns active comments for a single itinerary item.
func (s *Service) ListItemComments(ctx context.Context, tripID uuid.UUID, dayNumber, itemIndex int) ([]appdto.ItineraryCommentInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}

	comments, err := s.repo.ListItineraryCommentsByItem(ctx, tripID, dayNumber, itemIndex)
	if err != nil {
		return nil, err
	}
	return s.toCommentInfos(comments, user.ID, access), nil
}

// ListCommentCounts returns active comment counts grouped per itinerary item.
func (s *Service) ListCommentCounts(ctx context.Context, tripID uuid.UUID) ([]entity.ItineraryCommentCount, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	return s.repo.CountItineraryCommentsByTripGrouped(ctx, tripID)
}

// CreateComment validates input, confirms the target item exists, and stores a
// new active comment. Owners and accepted editor/viewer collaborators may
// comment because comments never mutate itinerary content.
func (s *Service) CreateComment(ctx context.Context, tripID uuid.UUID, in appdto.CreateCommentInput) (appdto.ItineraryCommentInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}

	if in.DayNumber < 1 {
		return appdto.ItineraryCommentInfo{}, apperrs.NewInvalidInput("dayNumber must be >= 1")
	}
	if in.ItemIndex < 0 {
		return appdto.ItineraryCommentInfo{}, apperrs.NewInvalidInput("itemIndex must be >= 0")
	}
	body, err := normalizeCommentBody(in.Body)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}
	if err := assertItineraryItemExists(trip, in.DayNumber, in.ItemIndex); err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}

	created, err := s.repo.CreateItineraryComment(ctx, &entity.ItineraryComment{
		ID:           uuid.New(),
		TripID:       tripID,
		DayNumber:    in.DayNumber,
		ItemIndex:    in.ItemIndex,
		AuthorUserID: user.ID,
		Body:         body,
		Status:       entity.CommentStatusActive,
	})
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCommentCreated,
		EntityType:  activityEntityType(activity.EntityComment),
		EntityID:    activityEntityID(created.ID),
		Metadata:    commentActivityMetadata(trip, created.DayNumber, created.ItemIndex),
	})

	return s.toCommentInfo(*created, user.ID, access), nil
}

// UpdateComment edits the body of the caller's own active comment.
func (s *Service) UpdateComment(ctx context.Context, tripID, commentID uuid.UUID, in appdto.UpdateCommentInput) (appdto.ItineraryCommentInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}

	body, err := normalizeCommentBody(in.Body)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}

	existing, err := s.repo.GetItineraryCommentByID(ctx, tripID, commentID)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}
	if existing.Status != entity.CommentStatusActive {
		return appdto.ItineraryCommentInfo{}, domainerrs.ErrNotFound
	}
	if existing.AuthorUserID != user.ID {
		return appdto.ItineraryCommentInfo{}, apperrs.ErrForbidden
	}

	updated, err := s.repo.UpdateItineraryCommentBody(ctx, tripID, commentID, body)
	if err != nil {
		return appdto.ItineraryCommentInfo{}, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCommentUpdated,
		EntityType:  activityEntityType(activity.EntityComment),
		EntityID:    activityEntityID(updated.ID),
		Metadata:    commentActivityMetadata(trip, updated.DayNumber, updated.ItemIndex),
	})

	return s.toCommentInfo(*updated, user.ID, access), nil
}

// DeleteComment soft-deletes a comment. The author may delete their own comment;
// the trip owner may delete any comment.
func (s *Service) DeleteComment(ctx context.Context, tripID, commentID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetItineraryCommentByID(ctx, tripID, commentID)
	if err != nil {
		return err
	}
	if existing.Status != entity.CommentStatusActive {
		return domainerrs.ErrNotFound
	}
	isAuthor := existing.AuthorUserID == user.ID
	isOwner := access.Level == AccessLevelOwner
	if !isAuthor && !isOwner {
		return apperrs.ErrForbidden
	}

	if _, err := s.repo.SoftDeleteItineraryComment(ctx, tripID, commentID); err != nil {
		return err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCommentDeleted,
		EntityType:  activityEntityType(activity.EntityComment),
		EntityID:    activityEntityID(existing.ID),
		Metadata:    commentActivityMetadata(trip, existing.DayNumber, existing.ItemIndex),
	})

	return nil
}

func (s *Service) toCommentInfos(comments []entity.ItineraryComment, userID uuid.UUID, access TripAccess) []appdto.ItineraryCommentInfo {
	out := make([]appdto.ItineraryCommentInfo, 0, len(comments))
	for i := range comments {
		out = append(out, s.toCommentInfo(comments[i], userID, access))
	}
	return out
}

func (s *Service) toCommentInfo(comment entity.ItineraryComment, userID uuid.UUID, access TripAccess) appdto.ItineraryCommentInfo {
	isAuthor := comment.AuthorUserID == userID
	isOwner := access.Level == AccessLevelOwner
	return appdto.ItineraryCommentInfo{
		Comment:   comment,
		IsAuthor:  isAuthor,
		CanEdit:   isAuthor,
		CanDelete: isAuthor || isOwner,
	}
}

func normalizeCommentBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", apperrs.NewInvalidInput("body is required")
	}
	if len(body) > maxCommentBodyLength {
		return "", apperrs.NewInvalidInput("body must be at most %d characters", maxCommentBodyLength)
	}
	return body, nil
}

// itineraryItemName returns the name of the itinerary item at the given
// day/item position, or "" when the trip has no itinerary or the position does
// not resolve. It is best-effort: it never errors so activity metadata can omit
// the name rather than fail the surrounding action.
func itineraryItemName(t *entity.Trip, dayNumber, itemIndex int) string {
	if t == nil || len(t.Itinerary) == 0 || strings.EqualFold(strings.TrimSpace(string(t.Itinerary)), "null") {
		return ""
	}
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(t.Itinerary, &itinerary); err != nil {
		return ""
	}
	for _, day := range itinerary.Days {
		if day.Day == dayNumber {
			if itemIndex < 0 || itemIndex >= len(day.Items) {
				return ""
			}
			return strings.TrimSpace(day.Items[itemIndex].Name)
		}
	}
	return ""
}

// commentActivityMetadata builds the small, body-free metadata payload shared by
// the comment_created/updated/deleted events.
func commentActivityMetadata(trip *entity.Trip, dayNumber, itemIndex int) map[string]any {
	metadata := map[string]any{
		"dayNumber": dayNumber,
		"itemIndex": itemIndex,
	}
	if name := itineraryItemName(trip, dayNumber, itemIndex); name != "" {
		metadata["itemName"] = name
	}
	return metadata
}

// assertItineraryItemExists confirms the target day/item exists in the trip's
// stored itinerary so comments cannot be attached to non-existent items. It is
// intentionally targeted (not a full itinerary validation) so a problem on one
// unrelated item never blocks commenting elsewhere.
func assertItineraryItemExists(t *entity.Trip, dayNumber, itemIndex int) error {
	notFound := apperrs.NewInvalidInput("itinerary item (dayNumber=%d, itemIndex=%d) does not exist", dayNumber, itemIndex)
	if t == nil || len(t.Itinerary) == 0 || strings.EqualFold(strings.TrimSpace(string(t.Itinerary)), "null") {
		return notFound
	}

	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(t.Itinerary, &itinerary); err != nil {
		return notFound
	}
	for _, day := range itinerary.Days {
		if day.Day == dayNumber {
			if itemIndex < 0 || itemIndex >= len(day.Items) {
				return notFound
			}
			return nil
		}
	}
	return notFound
}
