package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// activityService is the activity feed port. The concrete activity.Service
// satisfies it; tests substitute a fake to assert which events are recorded.
type activityService interface {
	Record(ctx context.Context, in activity.RecordActivityInput) error
	List(ctx context.Context, in activity.ListActivityInput) (*activity.ListActivityResult, error)
}

// WithActivity enables persistent activity-feed recording and reading. When not
// configured, recording is a no-op and ListActivity returns an empty page so
// older trips and tests keep working.
func WithActivity(svc activityService) Option {
	return func(s *Service) {
		s.activity = svc
	}
}

// ListActivity returns one newest-first page of a trip's activity feed. Access
// is restricted to the owner and accepted editor/viewer collaborators; pending,
// removed, and non-collaborators receive the same not-found shape as other trip
// reads. Public share viewers never reach this method (no public route).
func (s *Service) ListActivity(ctx context.Context, tripID uuid.UUID, limit int, cursor string) (*activity.ListActivityResult, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}

	if limit != 0 && (limit < 1 || limit > activity.MaxLimit) {
		return nil, apperrs.NewInvalidInput("limit must be between 1 and %d", activity.MaxLimit)
	}

	cursorCreatedAt, cursorID, err := activity.DecodeCursor(cursor)
	if err != nil {
		return nil, apperrs.NewInvalidInput("invalid cursor")
	}

	if s.activity == nil {
		return &activity.ListActivityResult{Events: []entity.TripActivityEvent{}}, nil
	}

	return s.activity.List(ctx, activity.ListActivityInput{
		TripID:          tripID,
		Limit:           limit,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
	})
}

// recordActivity persists one activity event best-effort. Recording never fails
// the originating user action: a failure is logged and swallowed. Callers must
// only invoke it after the action has already succeeded.
func (s *Service) recordActivity(ctx context.Context, in activity.RecordActivityInput) {
	if s.activity == nil {
		return
	}
	if err := s.activity.Record(ctx, in); err != nil {
		s.log.Warn("failed to record activity event",
			zap.String("event_type", in.EventType),
			zap.String("trip_id", in.TripID.String()),
			zap.Error(err),
		)
	}
}

// activityEntityType is a small helper for the optional *string entity_type
// field so call sites stay readable.
func activityEntityType(entityType string) *string {
	return &entityType
}

// activityEntityID is a small helper for the optional *uuid.UUID entity_id field.
func activityEntityID(id uuid.UUID) *uuid.UUID {
	return &id
}
