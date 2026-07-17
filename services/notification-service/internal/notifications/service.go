package notifications

import (
	"context"
	"strings"
	"time"

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
	repo           Repository
	log            *zap.Logger
	dedupeWindow   time.Duration
	groupingWindow time.Duration
}

// New constructs the notification service.
func New(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo: repo, log: log,
		dedupeWindow: 30 * time.Minute, groupingWindow: 60 * time.Minute,
	}
}

func (s *Service) WithGroupingWindow(window time.Duration) *Service {
	if window > 0 {
		s.groupingWindow = window
	}
	return s
}

func (s *Service) WithDedupeWindow(window time.Duration) *Service {
	if window > 0 {
		s.dedupeWindow = window
	}
	return s
}

type dedupeRepository interface {
	FindRecentNotificationByDedupeKey(ctx context.Context, userID uuid.UUID, dedupeKey string, since time.Time) (*entity.Notification, error)
	GroupNotification(ctx context.Context, id, userID uuid.UUID, latest entity.Notification) (*entity.Notification, error)
}

type atomicDedupeRepository interface {
	ClaimNotificationDedupe(ctx context.Context, userID uuid.UUID, dedupeKey string, now, since time.Time) (bool, *uuid.UUID, error)
	BindNotificationDedupe(ctx context.Context, userID uuid.UUID, dedupeKey string, notificationID uuid.UUID, since time.Time) error
	GroupNotification(ctx context.Context, id, userID uuid.UUID, latest entity.Notification) (*entity.Notification, error)
}

type groupingRepository interface {
	FindRecentNotificationByDigestKey(ctx context.Context, userID uuid.UUID, digestKey string, since time.Time) (*entity.Notification, error)
	GroupRelatedNotification(ctx context.Context, id, userID uuid.UUID, latest entity.Notification) (*entity.Notification, error)
}

type dedupeBinding struct {
	userID         uuid.UUID
	dedupeKey      string
	notificationID uuid.UUID
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
	result, err := s.CreateBatchWithPreferences(ctx, inputs, nil)
	if err != nil {
		return nil, err
	}
	return result.Created, nil
}

// CreateBatchWithPreferences validates an internal batch, creates in-app rows
// allowed by the supplied preference gate, and returns email candidates
// independently of the in-app outcome. A nil gate means "allow all in-app",
// preserving the legacy behavior used by existing tests and callers.
func (s *Service) CreateBatchWithPreferences(ctx context.Context, inputs []CreateInput, gate InAppPreferenceGate) (*BatchCreateResult, error) {
	if len(inputs) == 0 {
		return nil, apperrs.NewInvalidInput("notifications array is required")
	}
	if len(inputs) > MaxBatchSize {
		return nil, apperrs.NewInvalidInput("batch size must be at most %d", MaxBatchSize)
	}
	// Validate the entire request before claiming any dedupe keys. A malformed
	// later item must not leave an earlier event claimed but undelivered.
	for i := range inputs {
		if err := validateCreateInput(inputs[i]); err != nil {
			return nil, err
		}
	}

	result := &BatchCreateResult{Requested: len(inputs)}
	toCreate := make([]entity.Notification, 0, len(inputs))
	pendingCandidates := make(map[string]int, len(inputs))
	pendingCreates := make(map[string]int, len(inputs))
	pendingGroups := make(map[string]int, len(inputs))
	pendingBindings := make(map[string]dedupeBinding, len(inputs))
	legacyDedupeRepo, hasLegacyDedupe := s.repo.(dedupeRepository)
	atomicDedupeRepo, hasAtomicDedupe := s.repo.(atomicDedupeRepository)
	groupingRepo, hasGrouping := s.repo.(groupingRepository)
	noiseControlEnabled := hasAtomicDedupe || hasLegacyDedupe
	now := time.Now().UTC()
	for i := range inputs {
		in := inputs[i]
		// Skip self-notifications defensively even though Trip Service is also
		// expected to omit them.
		if in.ActorUserID != nil && *in.ActorUserID == in.UserID {
			result.Skipped++
			continue
		}
		priority := strings.ToLower(strings.TrimSpace(in.Priority))
		if priority == "" {
			priority = DefaultPriority(in.Type)
		}
		category := strings.ToLower(strings.TrimSpace(in.Category))
		if category == "" {
			category = DefaultCategory(in.Type)
		}
		digestKeyValue := ""
		if in.DigestKey != nil {
			digestKeyValue = strings.TrimSpace(*in.DigestKey)
		}
		if digestKeyValue == "" {
			tripID := ""
			if in.TripID != nil {
				tripID = in.TripID.String()
			}
			digestKeyValue = DefaultDigestKey(in.Type, category, tripID)
		}
		digestKey := &digestKeyValue
		var dedupeKey *string
		if in.DedupeKey != nil && strings.TrimSpace(*in.DedupeKey) != "" {
			value := strings.TrimSpace(*in.DedupeKey)
			dedupeKey = &value
		}
		notification := entity.Notification{
			ID:            uuid.New(),
			UserID:        in.UserID,
			TripID:        in.TripID,
			ActorUserID:   in.ActorUserID,
			Type:          in.Type,
			Title:         in.Title,
			Message:       in.Message,
			EntityType:    in.EntityType,
			EntityID:      in.EntityID,
			Metadata:      sanitizeMetadata(in.Metadata),
			Priority:      priority,
			Category:      category,
			DigestKey:     digestKey,
			DedupeKey:     dedupeKey,
			GroupedCount:  1,
			LatestEventAt: now,
		}
		if resolver, ok := gate.(InAppDeliveryResolver); ok {
			mode, status := resolver.InAppDelivery(notification)
			if mode != "" {
				notification.DeliveryMode = &mode
			}
			if status != "" {
				notification.DeliveryStatus = &status
			}
		}
		dedupeMapKey := ""
		claimedDedupe := false
		if noiseControlEnabled && notification.DedupeKey != nil {
			dedupeMapKey = notification.UserID.String() + "|" + *notification.DedupeKey
			if candidateIndex, ok := pendingCandidates[dedupeMapKey]; ok {
				if hasAtomicDedupe {
					if _, _, err := atomicDedupeRepo.ClaimNotificationDedupe(ctx, notification.UserID, *notification.DedupeKey, now, now.Add(-s.dedupeWindow)); err != nil {
						return nil, err
					}
				}
				grouped := &result.EmailCandidates[candidateIndex]
				mergeNotificationOccurrence(grouped, notification)
				if createIndex, exists := pendingCreates[dedupeMapKey]; exists {
					mergeNotificationOccurrence(&toCreate[createIndex], notification)
				}
				result.Skipped++
				result.Grouped++
				result.DuplicatesDropped++
				continue
			}
			if hasAtomicDedupe {
				duplicate, notificationID, err := atomicDedupeRepo.ClaimNotificationDedupe(ctx, notification.UserID, *notification.DedupeKey, now, now.Add(-s.dedupeWindow))
				if err != nil {
					return nil, err
				}
				if duplicate {
					if notificationID != nil {
						if _, err := atomicDedupeRepo.GroupNotification(ctx, *notificationID, notification.UserID, notification); err != nil {
							return nil, err
						}
					}
					result.Skipped++
					result.Grouped++
					result.DuplicatesDropped++
					continue
				}
				claimedDedupe = true
			} else {
				existing, err := legacyDedupeRepo.FindRecentNotificationByDedupeKey(ctx, notification.UserID, *notification.DedupeKey, now.Add(-s.dedupeWindow))
				if err != nil {
					return nil, err
				}
				if existing != nil {
					if _, err := legacyDedupeRepo.GroupNotification(ctx, existing.ID, notification.UserID, notification); err != nil {
						return nil, err
					}
					result.Skipped++
					result.Grouped++
					result.DuplicatesDropped++
					continue
				}
			}
			pendingCandidates[dedupeMapKey] = len(result.EmailCandidates)
		}
		result.EmailCandidates = append(result.EmailCandidates, notification)
		allowedInApp := true
		if richGate, ok := gate.(InAppNotificationGate); ok {
			allowedInApp = richGate.AllowInAppNotification(notification)
		} else if gate != nil {
			allowedInApp = gate.AllowInApp(in.UserID, in.Type)
		}
		if !allowedInApp {
			result.Skipped++
			result.SkippedByPreference++
			s.log.Debug("in-app notification skipped by user preference",
				zap.String("user_id", in.UserID.String()),
				zap.String("type", in.Type),
			)
			continue
		}
		if hasGrouping && canGroupNotification(notification) && notification.DigestKey != nil {
			groupKey := notification.UserID.String() + "|" + *notification.DigestKey
			if createIndex, ok := pendingGroups[groupKey]; ok {
				mergeNotificationOccurrence(&toCreate[createIndex], notification)
				if dedupeMapKey != "" {
					pendingCreates[dedupeMapKey] = createIndex
					if claimedDedupe {
						pendingBindings[dedupeMapKey] = dedupeBinding{notification.UserID, *notification.DedupeKey, toCreate[createIndex].ID}
					}
				}
				result.Skipped++
				result.Grouped++
				continue
			}
			existing, err := groupingRepo.FindRecentNotificationByDigestKey(ctx, notification.UserID, *notification.DigestKey, now.Add(-s.groupingWindow))
			if err != nil {
				return nil, err
			}
			if existing != nil {
				grouped, err := groupingRepo.GroupRelatedNotification(ctx, existing.ID, notification.UserID, notification)
				if err != nil {
					return nil, err
				}
				occurrence := *grouped
				occurrence.GroupedCount = 1
				result.GroupedInApp = append(result.GroupedInApp, occurrence)
				if claimedDedupe {
					s.bindDedupe(ctx, atomicDedupeRepo, dedupeBinding{notification.UserID, *notification.DedupeKey, grouped.ID}, now)
				}
				result.Skipped++
				result.Grouped++
				continue
			}
			pendingGroups[groupKey] = len(toCreate)
		}
		if dedupeMapKey != "" {
			pendingCreates[dedupeMapKey] = len(toCreate)
			if claimedDedupe {
				pendingBindings[dedupeMapKey] = dedupeBinding{notification.UserID, *notification.DedupeKey, notification.ID}
			}
		}
		toCreate = append(toCreate, notification)
	}

	if len(toCreate) == 0 {
		result.Created = []entity.Notification{}
		return result, nil
	}

	if _, err := s.repo.CreateNotifications(ctx, toCreate); err != nil {
		return nil, err
	}
	for _, binding := range pendingBindings {
		s.bindDedupe(ctx, atomicDedupeRepo, binding, now)
	}
	result.Created = toCreate
	createdByID := make(map[uuid.UUID]entity.Notification, len(toCreate))
	for _, created := range toCreate {
		createdByID[created.ID] = created
	}
	for i := range result.EmailCandidates {
		if created, ok := createdByID[result.EmailCandidates[i].ID]; ok {
			result.EmailCandidates[i].CreatedAt = created.CreatedAt
		}
	}
	return result, nil
}

func canGroupNotification(notification entity.Notification) bool {
	return notification.Priority == PriorityLow || notification.Priority == PriorityNormal
}

func mergeNotificationOccurrence(target *entity.Notification, latest entity.Notification) {
	target.Type = latest.Type
	target.Title = latest.Title
	target.Message = latest.Message
	target.EntityType = latest.EntityType
	target.EntityID = latest.EntityID
	target.Metadata = latest.Metadata
	target.Priority = latest.Priority
	target.Category = latest.Category
	target.DigestKey = latest.DigestKey
	target.DedupeKey = latest.DedupeKey
	target.DeliveryMode = latest.DeliveryMode
	target.DeliveryStatus = latest.DeliveryStatus
	target.LatestEventAt = latest.LatestEventAt
	target.GroupedCount++
}

func (s *Service) bindDedupe(ctx context.Context, repo atomicDedupeRepository, binding dedupeBinding, now time.Time) {
	if repo == nil || binding.notificationID == uuid.Nil {
		return
	}
	if err := repo.BindNotificationDedupe(ctx, binding.userID, binding.dedupeKey, binding.notificationID, now.Add(-s.dedupeWindow)); err != nil {
		s.log.Warn("notification dedupe link could not be persisted",
			zap.String("user_id", binding.userID.String()), zap.Error(err))
	}
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

func (s *Service) MarkTripRead(ctx context.Context, userID, tripID uuid.UUID) (int, error) {
	type tripReadRepository interface {
		MarkTripNotificationsRead(context.Context, uuid.UUID, uuid.UUID) (int, error)
	}
	repo, ok := s.repo.(tripReadRepository)
	if !ok {
		return 0, apperrs.NewInvalidInput("trip notification bulk actions are unavailable")
	}
	return repo.MarkTripNotificationsRead(ctx, userID, tripID)
}

func validateCreateInput(in CreateInput) error {
	if in.UserID == uuid.Nil {
		return apperrs.NewInvalidInput("userId is required")
	}
	notificationType := strings.TrimSpace(in.Type)
	if notificationType == "" {
		return apperrs.NewInvalidInput("type is required")
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
	if value := strings.ToLower(strings.TrimSpace(in.Priority)); value != "" && !IsPriority(value) {
		return apperrs.NewInvalidInput("priority must be low, normal, high, or urgent")
	}
	if in.DigestKey != nil && len(strings.TrimSpace(*in.DigestKey)) > MaxGroupingKeyLength {
		return apperrs.NewInvalidInput("digestKey must be at most %d characters", MaxGroupingKeyLength)
	}
	if in.DedupeKey != nil && len(strings.TrimSpace(*in.DedupeKey)) > MaxGroupingKeyLength {
		return apperrs.NewInvalidInput("dedupeKey must be at most %d characters", MaxGroupingKeyLength)
	}
	return nil
}
