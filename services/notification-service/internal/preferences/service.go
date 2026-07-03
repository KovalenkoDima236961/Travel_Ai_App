package preferences

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

// Repository is the persistence port for notification preferences. The concrete
// postgres repository satisfies it; tests substitute a fake.
type Repository interface {
	ListNotificationPreferencesByUsers(ctx context.Context, userIDs []uuid.UUID) ([]entity.NotificationPreference, error)
	UpsertNotificationPreferencesBatch(ctx context.Context, userID uuid.UUID, items []PreferenceInput) error
}

// Service holds the notification-preference business logic. It depends on the
// repository port and a logger. Like the other Notification Service use cases it
// performs no JWT checks itself: the HTTP layer authenticates the caller and
// passes the trusted user id, and there is no userId in any request body/path.
type Service struct {
	repo Repository
	log  *zap.Logger
}

// New constructs the preferences service.
func New(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log}
}

// GetPreferences returns the user's full effective preference matrix: the
// default matrix with any stored overrides applied. The result always contains
// every channel/category combination (8 items in v1), in a stable order.
func (s *Service) GetPreferences(ctx context.Context, userID uuid.UUID) (*PreferencesResult, error) {
	set, err := s.EffectiveForUsers(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}
	return buildResult(set.Matrix(userID)), nil
}

// UpdatePreferences validates and upserts the given preference items for the
// user, then returns the resulting full effective matrix. Preferences apply to
// future notifications only; existing notifications are never changed.
func (s *Service) UpdatePreferences(ctx context.Context, userID uuid.UUID, items []PreferenceInput) (*PreferencesResult, error) {
	if err := validateUpdateItems(items); err != nil {
		return nil, err
	}
	if err := s.repo.UpsertNotificationPreferencesBatch(ctx, userID, items); err != nil {
		return nil, err
	}
	return s.GetPreferences(ctx, userID)
}

// IsEnabled reports whether the given channel is enabled for the category that
// the notification type maps to, for the given user. Unknown types follow the
// documented unknown-type defaults (in-app allowed, email not).
func (s *Service) IsEnabled(ctx context.Context, userID uuid.UUID, channel string, notificationType string) (bool, error) {
	set, err := s.EffectiveForUsers(ctx, []uuid.UUID{userID})
	if err != nil {
		return false, err
	}
	switch channel {
	case ChannelEmail:
		return set.AllowEmail(userID, notificationType), nil
	case ChannelPush:
		return set.AllowPush(userID, notificationType), nil
	default:
		return set.AllowInApp(userID, notificationType), nil
	}
}

// GetEffectivePreferences returns the full merged channel/category matrix for a
// single user (channel -> category -> enabled).
func (s *Service) GetEffectivePreferences(ctx context.Context, userID uuid.UUID) (map[string]map[string]bool, error) {
	set, err := s.EffectiveForUsers(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}
	return set.Matrix(userID), nil
}

// EffectiveForUsers loads the stored preference rows for the given users in one
// query and merges them over the defaults, returning a snapshot usable as an
// in-app/email gate. The returned set is never nil. Duplicate and nil ids are
// ignored.
func (s *Service) EffectiveForUsers(ctx context.Context, userIDs []uuid.UUID) (*EffectiveSet, error) {
	unique := dedupeIDs(userIDs)
	if len(unique) == 0 {
		return BuildEffectiveSet(nil, nil), nil
	}
	rows, err := s.repo.ListNotificationPreferencesByUsers(ctx, unique)
	if err != nil {
		return nil, err
	}
	return BuildEffectiveSet(unique, rows), nil
}

// validateUpdateItems enforces the update-request rules independently of
// transport so they stay unit-testable: at least one item, at most
// MaxUpdateItems, known channel/category, and no duplicate (channel, category)
// pairs.
func validateUpdateItems(items []PreferenceInput) error {
	if len(items) == 0 {
		return apperrs.NewInvalidInput("items array is required")
	}
	if len(items) > MaxUpdateItems {
		return apperrs.NewInvalidInput("items must contain at most %d entries", MaxUpdateItems)
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !IsKnownChannel(item.Channel) {
			return apperrs.NewInvalidInput("channel %q is not a known channel", item.Channel)
		}
		if !IsKnownCategory(item.Category) {
			return apperrs.NewInvalidInput("category %q is not a known category", item.Category)
		}
		key := item.Channel + "|" + item.Category
		if _, dup := seen[key]; dup {
			return apperrs.NewInvalidInput("duplicate preference for channel %q and category %q", item.Channel, item.Category)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// buildResult flattens a merged matrix into the ordered item list. It ranges the
// ordered AllChannels/AllCategories slices (never the map) so the output order
// is deterministic.
func buildResult(matrix map[string]map[string]bool) *PreferencesResult {
	items := make([]PreferenceItem, 0, len(AllChannels)*len(AllCategories))
	for _, channel := range AllChannels {
		for _, category := range AllCategories {
			enabled := defaultEnabled(channel, category)
			if byCategory, ok := matrix[channel]; ok {
				if v, ok := byCategory[category]; ok {
					enabled = v
				}
			}
			items = append(items, PreferenceItem{
				Channel:  channel,
				Category: category,
				Enabled:  enabled,
			})
		}
	}
	return &PreferencesResult{Items: items}
}

func dedupeIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
