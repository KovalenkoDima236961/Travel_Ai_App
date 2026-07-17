package controls

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

type Repository interface {
	ListNotificationSettingsByUsers(ctx context.Context, userIDs []uuid.UUID) ([]entity.NotificationSettings, error)
	UpsertNotificationSettings(ctx context.Context, settings entity.NotificationSettings) (*entity.NotificationSettings, error)
	ListActiveTripMutesByUsers(ctx context.Context, userIDs []uuid.UUID, now time.Time) ([]entity.NotificationTripMute, error)
	ListTripMutesByUserAndTrip(ctx context.Context, userID, tripID uuid.UUID) ([]entity.NotificationTripMute, error)
	UpsertTripMute(ctx context.Context, mute entity.NotificationTripMute) (*entity.NotificationTripMute, error)
	DeleteTripMuteByIDAndUser(ctx context.Context, id, userID uuid.UUID) error
}

type Service struct {
	repo Repository
	log  *zap.Logger
}

func New(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log}
}

func (s *Service) SettingsForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]entity.NotificationSettings, error) {
	result := make(map[uuid.UUID]entity.NotificationSettings, len(userIDs))
	for _, id := range userIDs {
		if id != uuid.Nil {
			result[id] = defaultEntitySettings(id)
		}
	}
	rows, err := s.repo.ListNotificationSettingsByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserID] = row
	}
	return result, nil
}

func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) (*entity.NotificationSettings, error) {
	settings, err := s.SettingsForUsers(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}
	value := settings[userID]
	return &value, nil
}

func (s *Service) UpdateSettings(ctx context.Context, input SettingsInput) (*entity.NotificationSettings, error) {
	if input.UserID == uuid.Nil {
		return nil, apperrs.NewInvalidInput("user id is required")
	}
	input.QuietHoursTimezone = strings.TrimSpace(input.QuietHoursTimezone)
	if input.QuietHoursTimezone == "" {
		input.QuietHoursTimezone = "UTC"
	}
	if _, err := time.LoadLocation(input.QuietHoursTimezone); err != nil {
		return nil, apperrs.NewInvalidInput("quietHoursTimezone must be a valid IANA timezone")
	}
	for field, value := range map[string]string{
		"quietHoursStart": input.QuietHoursStart, "quietHoursEnd": input.QuietHoursEnd,
		"dailyDigestTime": input.DailyDigestTime, "weeklyDigestTime": input.WeeklyDigestTime,
	} {
		if _, err := time.Parse("15:04", value); err != nil {
			return nil, apperrs.NewInvalidInput("%s must use HH:MM", field)
		}
	}
	if input.WeeklyDigestDay < 0 || input.WeeklyDigestDay > 6 {
		return nil, apperrs.NewInvalidInput("weeklyDigestDay must be between 0 and 6")
	}
	return s.repo.UpsertNotificationSettings(ctx, entity.NotificationSettings{
		UserID: input.UserID, QuietHoursEnabled: input.QuietHoursEnabled,
		QuietHoursStart: input.QuietHoursStart, QuietHoursEnd: input.QuietHoursEnd,
		QuietHoursTimezone:       input.QuietHoursTimezone,
		UrgentBypassesQuietHours: input.UrgentBypassesQuietHours,
		DailyDigestTime:          input.DailyDigestTime, WeeklyDigestDay: input.WeeklyDigestDay,
		WeeklyDigestTime: input.WeeklyDigestTime,
	})
}

func (s *Service) ActiveTripMutesForUsers(ctx context.Context, userIDs []uuid.UUID, now time.Time) ([]entity.NotificationTripMute, error) {
	return s.repo.ListActiveTripMutesByUsers(ctx, userIDs, now)
}

func (s *Service) ListTripMutes(ctx context.Context, userID, tripID uuid.UUID) ([]entity.NotificationTripMute, error) {
	if tripID == uuid.Nil {
		return nil, apperrs.NewInvalidInput("tripId is required")
	}
	return s.repo.ListTripMutesByUserAndTrip(ctx, userID, tripID)
}

func (s *Service) UpsertTripMute(ctx context.Context, input TripMuteInput) (*entity.NotificationTripMute, error) {
	if input.UserID == uuid.Nil || input.TripID == uuid.Nil {
		return nil, apperrs.NewInvalidInput("tripId is required")
	}
	if input.Category != nil {
		category := strings.TrimSpace(*input.Category)
		if category == "" {
			input.Category = nil
		} else {
			if !preferences.IsKnownCategory(category) {
				return nil, apperrs.NewInvalidInput("category %q is not known", category)
			}
			input.Category = &category
		}
	}
	if input.MutedUntil != nil {
		value := input.MutedUntil.UTC()
		input.MutedUntil = &value
	}
	return s.repo.UpsertTripMute(ctx, entity.NotificationTripMute{
		ID: uuid.New(), UserID: input.UserID, TripID: input.TripID,
		Category: input.Category, MutedUntil: input.MutedUntil,
	})
}

func (s *Service) DeleteTripMute(ctx context.Context, id, userID uuid.UUID) error {
	if id == uuid.Nil {
		return apperrs.NewInvalidInput("mute id is required")
	}
	return s.repo.DeleteTripMuteByIDAndUser(ctx, id, userID)
}

func defaultEntitySettings(userID uuid.UUID) entity.NotificationSettings {
	d := DefaultSettings(userID)
	return entity.NotificationSettings{
		UserID: userID, QuietHoursEnabled: d.QuietHoursEnabled,
		QuietHoursStart: d.QuietHoursStart, QuietHoursEnd: d.QuietHoursEnd,
		QuietHoursTimezone: d.QuietHoursTimezone, UrgentBypassesQuietHours: d.UrgentBypassesQuietHours,
		DailyDigestTime: d.DailyDigestTime, WeeklyDigestDay: d.WeeklyDigestDay,
		WeeklyDigestTime: d.WeeklyDigestTime,
	}
}
