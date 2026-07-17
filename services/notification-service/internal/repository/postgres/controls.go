package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/repository/postgres/dto"
)

const settingsProjection = "user_id, quiet_hours_enabled, to_char(quiet_hours_start, 'HH24:MI'), " +
	"to_char(quiet_hours_end, 'HH24:MI'), quiet_hours_timezone, urgent_bypasses_quiet_hours, " +
	"to_char(daily_digest_time, 'HH24:MI'), weekly_digest_day, to_char(weekly_digest_time, 'HH24:MI'), created_at, updated_at"

func (r *Repository) ListNotificationSettingsByUsers(ctx context.Context, userIDs []uuid.UUID) ([]entity.NotificationSettings, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	ids := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		if id != uuid.Nil {
			ids = append(ids, dto.IDArg(id))
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := r.db.Builder.Select(settingsProjection).From("notification_settings").Where(sq.Eq{"user_id": ids}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list notification settings: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notification settings: %w", err)
	}
	defer rows.Close()
	out := make([]entity.NotificationSettings, 0)
	for rows.Next() {
		value, err := scanSettings(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *value)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertNotificationSettings(ctx context.Context, value entity.NotificationSettings) (*entity.NotificationSettings, error) {
	query := `INSERT INTO notification_settings
(user_id, quiet_hours_enabled, quiet_hours_start, quiet_hours_end, quiet_hours_timezone,
 urgent_bypasses_quiet_hours, daily_digest_time, weekly_digest_day, weekly_digest_time)
VALUES ($1,$2,$3::time,$4::time,$5,$6,$7::time,$8,$9::time)
ON CONFLICT (user_id) DO UPDATE SET
 quiet_hours_enabled=EXCLUDED.quiet_hours_enabled, quiet_hours_start=EXCLUDED.quiet_hours_start,
 quiet_hours_end=EXCLUDED.quiet_hours_end, quiet_hours_timezone=EXCLUDED.quiet_hours_timezone,
 urgent_bypasses_quiet_hours=EXCLUDED.urgent_bypasses_quiet_hours,
 daily_digest_time=EXCLUDED.daily_digest_time, weekly_digest_day=EXCLUDED.weekly_digest_day,
 weekly_digest_time=EXCLUDED.weekly_digest_time, updated_at=NOW()
RETURNING ` + settingsProjection
	return scanSettings(r.db.QueryRow(ctx, query, dto.IDArg(value.UserID), value.QuietHoursEnabled,
		value.QuietHoursStart, value.QuietHoursEnd, value.QuietHoursTimezone,
		value.UrgentBypassesQuietHours, value.DailyDigestTime, value.WeeklyDigestDay, value.WeeklyDigestTime))
}

func scanSettings(row pgx.Row) (*entity.NotificationSettings, error) {
	var value entity.NotificationSettings
	if err := row.Scan(&value.UserID, &value.QuietHoursEnabled, &value.QuietHoursStart, &value.QuietHoursEnd,
		&value.QuietHoursTimezone, &value.UrgentBypassesQuietHours, &value.DailyDigestTime,
		&value.WeeklyDigestDay, &value.WeeklyDigestTime, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan notification settings: %w", err)
	}
	return &value, nil
}

const muteProjection = "id, user_id, trip_id, category, muted_until, created_at, updated_at"

func (r *Repository) ListActiveTripMutesByUsers(ctx context.Context, userIDs []uuid.UUID, now time.Time) ([]entity.NotificationTripMute, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	ids := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		if id != uuid.Nil {
			ids = append(ids, dto.IDArg(id))
		}
	}
	query, args, err := r.db.Builder.Select(muteProjection).From("notification_trip_mutes").
		Where(sq.Eq{"user_id": ids}).Where(sq.Or{sq.Eq{"muted_until": nil}, sq.Gt{"muted_until": now.UTC()}}).ToSql()
	if err != nil {
		return nil, err
	}
	return r.scanTripMutes(ctx, query, args...)
}

func (r *Repository) ListTripMutesByUserAndTrip(ctx context.Context, userID, tripID uuid.UUID) ([]entity.NotificationTripMute, error) {
	query, args, err := r.db.Builder.Select(muteProjection).From("notification_trip_mutes").
		Where(sq.Eq{"user_id": dto.IDArg(userID), "trip_id": dto.IDArg(tripID)}).OrderBy("category NULLS FIRST").ToSql()
	if err != nil {
		return nil, err
	}
	return r.scanTripMutes(ctx, query, args...)
}

func (r *Repository) UpsertTripMute(ctx context.Context, value entity.NotificationTripMute) (*entity.NotificationTripMute, error) {
	query := `INSERT INTO notification_trip_mutes (id,user_id,trip_id,category,muted_until)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (user_id,trip_id,(COALESCE(category,''))) DO UPDATE SET muted_until=EXCLUDED.muted_until, updated_at=NOW()
RETURNING ` + muteProjection
	return scanTripMute(r.db.QueryRow(ctx, query, dto.IDArg(value.ID), dto.IDArg(value.UserID), dto.IDArg(value.TripID), value.Category, value.MutedUntil))
}

func (r *Repository) DeleteTripMuteByIDAndUser(ctx context.Context, id, userID uuid.UUID) error {
	query, args, err := r.db.Builder.Delete("notification_trip_mutes").Where(sq.Eq{"id": dto.IDArg(id), "user_id": dto.IDArg(userID)}).ToSql()
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete trip mute: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerrs.ErrNotFound
	}
	return nil
}

func (r *Repository) scanTripMutes(ctx context.Context, query string, args ...any) ([]entity.NotificationTripMute, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip mutes: %w", err)
	}
	defer rows.Close()
	out := make([]entity.NotificationTripMute, 0)
	for rows.Next() {
		value, err := scanTripMute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *value)
	}
	return out, rows.Err()
}

func scanTripMute(row pgx.Row) (*entity.NotificationTripMute, error) {
	var value entity.NotificationTripMute
	if err := row.Scan(&value.ID, &value.UserID, &value.TripID, &value.Category, &value.MutedUntil, &value.CreatedAt, &value.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip mute: %w", err)
	}
	return &value, nil
}
