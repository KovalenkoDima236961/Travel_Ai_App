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
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/infrastructure/repository/postgres/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/storage/postgres"
)

// Repository persists notifications using squirrel query building over the
// shared postgres pool.
type Repository struct {
	db *storage.DB
}

// New constructs the notification repository.
func New(db *storage.DB) *Repository {
	return &Repository{db: db}
}

// CreateNotifications inserts a batch of notifications inside a single
// transaction and returns how many rows were created. The whole batch commits
// or rolls back together, so a partial insert never leaves a user with half a
// fan-out.
func (r *Repository) CreateNotifications(ctx context.Context, notifications []entity.Notification) (int, error) {
	if len(notifications) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin notifications tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created := 0
	for i := range notifications {
		values, err := dto.InsertValues(&notifications[i])
		if err != nil {
			return 0, err
		}
		query, args, err := r.db.Builder.
			Insert("notifications").
			Columns(dto.InsertColumns()...).
			Values(values...).
			Suffix("RETURNING created_at").
			ToSql()
		if err != nil {
			return 0, fmt.Errorf("build insert notification: %w", err)
		}
		var createdAt time.Time
		if err := tx.QueryRow(ctx, query, args...).Scan(&createdAt); err != nil {
			return 0, fmt.Errorf("insert notification: %w", err)
		}
		notifications[i].CreatedAt = createdAt.UTC()
		created++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit notifications tx: %w", err)
	}
	return created, nil
}

// ListNotificationsByUser returns a page of a user's notifications ordered
// newest first (created_at DESC, id DESC). When a cursor is supplied it returns
// only rows strictly older than the cursor position for stable keyset
// pagination.
func (r *Repository) ListNotificationsByUser(ctx context.Context, in notifications.ListInput) ([]entity.Notification, error) {
	builder := r.db.Builder.
		Select(dto.Columns).
		From("notifications").
		Where(sq.Eq{"user_id": dto.IDArg(in.UserID)})

	if in.CursorCreatedAt != nil && in.CursorID != nil {
		// Keyset comparison on (created_at, id) for stable ordering across pages.
		builder = builder.Where(
			sq.Expr("(created_at, id) < (?, ?)", *in.CursorCreatedAt, dto.IDArg(*in.CursorID)),
		)
	}

	limit := in.Limit
	if limit <= 0 {
		limit = notifications.DefaultLimit
	}

	query, args, err := builder.
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list notifications by user: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notifications by user: %w", err)
	}
	defer rows.Close()

	return dto.ScanRows(rows)
}

// GetNotificationByIDAndUser loads a single notification scoped to its owner.
// Scoping by user_id prevents reading another user's notification by id.
func (r *Repository) GetNotificationByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error) {
	query, args, err := r.db.Builder.
		Select(dto.Columns).
		From("notifications").
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"user_id": dto.IDArg(userID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get notification by id and user: %w", err)
	}

	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}

// CountUnreadNotifications returns the number of unread notifications for a user.
func (r *Repository) CountUnreadNotifications(ctx context.Context, userID uuid.UUID) (int, error) {
	query, args, err := r.db.Builder.
		Select("COUNT(*)").
		From("notifications").
		Where(sq.Eq{
			"user_id": dto.IDArg(userID),
			"read_at": nil,
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count unread notifications: %w", err)
	}

	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

// MarkNotificationRead marks one unread notification (scoped to its owner) as
// read and returns the row. It is idempotent: an already-read notification is
// returned unchanged via a follow-up read so callers always get the current
// state without an error.
func (r *Repository) MarkNotificationRead(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error) {
	query, args, err := r.db.Builder.
		Update("notifications").
		Set("read_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"user_id": dto.IDArg(userID),
			"read_at": nil,
		}).
		Suffix("RETURNING " + dto.Columns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark notification read: %w", err)
	}

	updated, err := dto.Scan(r.db.QueryRow(ctx, query, args...))
	if err == nil {
		return updated, nil
	}
	// dto.Scan maps "no row" to domain ErrNotFound. A no-op UPDATE (the row is
	// already read, or absent) lands here; fall back to a plain read so marking
	// read stays idempotent and 404s only when the row is truly absent.
	if !errors.Is(err, domainerrs.ErrNotFound) {
		return nil, err
	}
	return r.GetNotificationByIDAndUser(ctx, id, userID)
}

// MarkAllNotificationsRead marks all of a user's unread notifications as read
// and returns how many rows changed.
func (r *Repository) MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	query, args, err := r.db.Builder.
		Update("notifications").
		Set("read_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"user_id": dto.IDArg(userID),
			"read_at": nil,
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build mark all notifications read: %w", err)
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ListNotificationPreferencesByUsers returns all stored preference overrides
// for the given users. Missing rows are intentionally not synthesized here; the
// preferences service merges defaults over the sparse stored overrides.
func (r *Repository) ListNotificationPreferencesByUsers(ctx context.Context, userIDs []uuid.UUID) ([]entity.NotificationPreference, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	ids := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		if id == uuid.Nil {
			continue
		}
		ids = append(ids, dto.IDArg(id))
	}
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := r.db.Builder.
		Select(dto.PreferenceColumns).
		From("notification_preferences").
		Where(sq.Eq{"user_id": ids}).
		OrderBy("user_id", "channel", "category").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list notification preferences: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notification preferences: %w", err)
	}
	defer rows.Close()

	return dto.ScanPreferenceRows(rows)
}

// UpsertNotificationPreferencesBatch stores sparse preference overrides for a
// user. The caller validates vocabulary and duplicates before reaching the
// repository. A transaction keeps a multi-item save atomic.
func (r *Repository) UpsertNotificationPreferencesBatch(ctx context.Context, userID uuid.UUID, items []preferences.PreferenceInput) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin notification preferences tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i := range items {
		item := items[i]
		query, args, err := r.db.Builder.
			Insert("notification_preferences").
			Columns(dto.PreferenceInsertColumns()...).
			Values(dto.PreferenceInsertValues(dto.IDArg(userID), item.Channel, item.Category, item.Enabled)...).
			Suffix("ON CONFLICT (user_id, channel, category) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()").
			ToSql()
		if err != nil {
			return fmt.Errorf("build upsert notification preference: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert notification preference: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit notification preferences tx: %w", err)
	}
	return nil
}
