package activity

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// Repository is the persistence port the activity service depends on. The
// existing postgres repository satisfies it; tests substitute a fake.
type Repository interface {
	CreateTripActivityEvent(ctx context.Context, event *entity.TripActivityEvent) (*entity.TripActivityEvent, error)
	ListTripActivityEvents(
		ctx context.Context,
		tripID uuid.UUID,
		limit int,
		cursorCreatedAt *time.Time,
		cursorID *uuid.UUID,
	) ([]entity.TripActivityEvent, error)
}

// Service records and lists trip activity events. It performs no permission
// checks itself — the trip use case enforces who may read activity before
// calling List, and only records events after an action has already succeeded.
type Service struct {
	repo Repository
	log  *zap.Logger
}

// Record persists one activity event. It validates the event type, sanitizes
// metadata, and never assumes a transaction. Recording failures are the
// caller's to tolerate: the main action has already succeeded.
func (s *Service) Record(ctx context.Context, in RecordActivityInput) error {
	if in.TripID == uuid.Nil {
		return fmt.Errorf("activity: trip id is required")
	}
	if in.EventType == "" {
		return fmt.Errorf("activity: event type is required")
	}
	if !IsKnownEventType(in.EventType) {
		// Forward-compatible: still record, but surface likely typos.
		s.log.Warn("recording unknown activity event type",
			zap.String("event_type", in.EventType),
			zap.String("trip_id", in.TripID.String()),
		)
	}

	event := &entity.TripActivityEvent{
		ID:          uuid.New(),
		TripID:      in.TripID,
		ActorUserID: in.ActorUserID,
		EventType:   in.EventType,
		EntityType:  in.EntityType,
		EntityID:    in.EntityID,
		Metadata:    sanitizeMetadata(in.Metadata),
	}

	if _, err := s.repo.CreateTripActivityEvent(ctx, event); err != nil {
		return fmt.Errorf("create trip activity event: %w", err)
	}
	return nil
}

// List returns one newest-first page of a trip's activity plus an opaque cursor
// for the next page. It fetches limit+1 rows to detect whether more exist
// without a separate count query.
func (s *Service) List(ctx context.Context, in ListActivityInput) (*ListActivityResult, error) {
	limit := NormalizeLimit(in.Limit)

	events, err := s.repo.ListTripActivityEvents(ctx, in.TripID, limit+1, in.CursorCreatedAt, in.CursorID)
	if err != nil {
		return nil, err
	}

	result := &ListActivityResult{}
	if len(events) > limit {
		last := events[limit-1]
		result.NextCursor = EncodeCursor(last.CreatedAt, last.ID)
		events = events[:limit]
	}
	result.Events = events
	return result, nil
}
