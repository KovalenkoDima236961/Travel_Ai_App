package notifications

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

// Repository is the persistence port the notification service depends on. The
// concrete postgres repository satisfies it; tests substitute a fake.
type Repository interface {
	CreateNotifications(ctx context.Context, notifications []entity.Notification) (int, error)
	ListNotificationsByUser(ctx context.Context, in ListInput) ([]entity.Notification, error)
	GetNotificationByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error)
	CountUnreadNotifications(ctx context.Context, userID uuid.UUID) (int, error)
	MarkNotificationRead(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error)
	MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) (int, error)
}

// Service holds the notification business logic. It depends on the repository
// port and a logger. It performs no JWT/permission checks itself — the HTTP
// layer authenticates the caller and passes the trusted user/recipient ids.
type Service struct {
	repo Repository
	log  *zap.Logger
}

// New constructs the notification service.
func New(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log}
}

// CreateBatch validates and persists a batch of notifications, returning the
// notifications that were created (so the caller can fan out email for selected
// types). Notifications addressed to the actor themselves are skipped (userId ==
// actorUserId) so users never get notified about their own actions. An empty
// batch (or a batch that is entirely self-notifications) creates nothing and
// returns an empty slice without error.
//
// The whole batch is inserted in one transaction, so on success the returned
// slice is exactly what was persisted.
func (s *Service) CreateBatch(ctx context.Context, inputs []CreateInput) ([]entity.Notification, error) {
	if len(inputs) == 0 {
		return nil, apperrs.NewInvalidInput("notifications array is required")
	}
	if len(inputs) > MaxBatchSize {
		return nil, apperrs.NewInvalidInput("batch size must be at most %d", MaxBatchSize)
	}

	toCreate := make([]entity.Notification, 0, len(inputs))
	for i := range inputs {
		in := inputs[i]
		if err := validateCreateInput(in); err != nil {
			return nil, err
		}
		// Skip self-notifications defensively even though Trip Service is also
		// expected to omit them.
		if in.ActorUserID != nil && *in.ActorUserID == in.UserID {
			continue
		}
		toCreate = append(toCreate, entity.Notification{
			ID:          uuid.New(),
			UserID:      in.UserID,
			TripID:      in.TripID,
			ActorUserID: in.ActorUserID,
			Type:        in.Type,
			Title:       in.Title,
			Message:     in.Message,
			EntityType:  in.EntityType,
			EntityID:    in.EntityID,
			Metadata:    sanitizeMetadata(in.Metadata),
		})
	}

	if len(toCreate) == 0 {
		return nil, nil
	}

	if _, err := s.repo.CreateNotifications(ctx, toCreate); err != nil {
		return nil, err
	}
	return toCreate, nil
}

// List returns one newest-first page of a user's notifications plus an opaque
// cursor for the next page. It fetches limit+1 rows to detect whether more
// exist without a separate count query.
func (s *Service) List(ctx context.Context, in ListInput) (*ListResult, error) {
	limit := NormalizeLimit(in.Limit)

	rows, err := s.repo.ListNotificationsByUser(ctx, ListInput{
		UserID:          in.UserID,
		Limit:           limit + 1,
		CursorCreatedAt: in.CursorCreatedAt,
		CursorID:        in.CursorID,
	})
	if err != nil {
		return nil, err
	}

	result := &ListResult{}
	if len(rows) > limit {
		last := rows[limit-1]
		result.NextCursor = EncodeCursor(last.CreatedAt, last.ID)
		rows = rows[:limit]
	}
	result.Notifications = rows
	return result, nil
}

// CountUnread returns the number of unread notifications for a user.
func (s *Service) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnreadNotifications(ctx, userID)
}

// MarkRead marks one notification (scoped to its owner) as read. It is
// idempotent: marking an already-read notification simply returns it unchanged.
func (s *Service) MarkRead(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error) {
	return s.repo.MarkNotificationRead(ctx, id, userID)
}

// MarkAllRead marks all of a user's unread notifications as read and returns how
// many rows changed.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.MarkAllNotificationsRead(ctx, userID)
}

func validateCreateInput(in CreateInput) error {
	if in.UserID == uuid.Nil {
		return apperrs.NewInvalidInput("userId is required")
	}
	notificationType := strings.TrimSpace(in.Type)
	if notificationType == "" {
		return apperrs.NewInvalidInput("type is required")
	}
	if !IsKnownType(notificationType) {
		return apperrs.NewInvalidInput("type %q is not a known notification type", notificationType)
	}
	if strings.TrimSpace(in.Title) == "" {
		return apperrs.NewInvalidInput("title is required")
	}
	if len(in.Title) > MaxTitleLength {
		return apperrs.NewInvalidInput("title must be at most %d characters", MaxTitleLength)
	}
	if strings.TrimSpace(in.Message) == "" {
		return apperrs.NewInvalidInput("message is required")
	}
	if len(in.Message) > MaxMessageLength {
		return apperrs.NewInvalidInput("message must be at most %d characters", MaxMessageLength)
	}
	return nil
}
