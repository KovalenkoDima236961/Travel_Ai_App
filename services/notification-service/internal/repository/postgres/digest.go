package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/digests"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/repository/postgres/dto"
)

const digestBatchProjection = "id,user_id,channel,mode,status,scheduled_for,sent_at,attempts,next_attempt_at,error_code,error_message_safe,created_at,updated_at"
const digestItemProjection = "id,batch_id,notification_id,user_id,trip_id,category,priority,digest_key,title,message,metadata_json,event_count,latest_event_at,created_at,updated_at"

func (r *Repository) QueueDigestItem(ctx context.Context, input digests.QueueInput) (*entity.NotificationDigestBatch, bool, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, false, false, fmt.Errorf("begin queue digest: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	batchID := uuid.New()
	batchQuery := `INSERT INTO notification_digest_batches (id,user_id,channel,mode,status,scheduled_for)
VALUES ($1,$2,$3,$4,'pending',$5)
ON CONFLICT (user_id,channel,mode,scheduled_for) WHERE status = 'pending'
DO UPDATE SET updated_at=NOW()
RETURNING ` + digestBatchProjection
	batch, err := scanDigestBatch(tx.QueryRow(ctx, batchQuery, dto.IDArg(batchID), dto.IDArg(input.Notification.UserID), input.Channel, input.Mode, input.ScheduledFor.UTC()))
	if err != nil {
		return nil, false, false, err
	}
	batchCreated := batch.ID == batchID
	metadata, err := json.Marshal(input.Notification.Metadata)
	if err != nil {
		return nil, false, false, fmt.Errorf("marshal digest metadata: %w", err)
	}
	var notificationID any
	if !input.Notification.CreatedAt.IsZero() {
		notificationID = dto.IDArg(input.Notification.ID)
	}
	digestKey := "category:" + input.Notification.Category
	if input.Notification.DigestKey != nil && *input.Notification.DigestKey != "" {
		digestKey = *input.Notification.DigestKey
	}
	latestAt := input.Notification.LatestEventAt
	if latestAt.IsZero() {
		latestAt = time.Now().UTC()
	}
	itemID := uuid.New()
	eventCount := input.Notification.GroupedCount
	if eventCount < 1 {
		eventCount = 1
	}
	itemQuery := `INSERT INTO notification_digest_items
(id,batch_id,notification_id,user_id,trip_id,category,priority,digest_key,title,message,metadata_json,event_count,latest_event_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (batch_id,digest_key) DO UPDATE SET
 event_count=notification_digest_items.event_count+EXCLUDED.event_count,
 latest_event_at=GREATEST(notification_digest_items.latest_event_at,EXCLUDED.latest_event_at),
 title=EXCLUDED.title,message=EXCLUDED.message,metadata_json=EXCLUDED.metadata_json,
 priority=EXCLUDED.priority,updated_at=NOW()
RETURNING (xmax <> 0) AS grouped`
	var grouped bool
	if err := tx.QueryRow(ctx, itemQuery, dto.IDArg(itemID), dto.IDArg(batch.ID), notificationID,
		dto.IDArg(input.Notification.UserID), nullableUUID(input.Notification.TripID), input.Notification.Category,
		input.Notification.Priority, digestKey, input.Notification.Title, input.Notification.Message,
		metadata, eventCount, latestAt.UTC()).Scan(&grouped); err != nil {
		return nil, false, false, fmt.Errorf("queue digest item: %w", err)
	}
	if !input.Notification.CreatedAt.IsZero() && input.Channel == "in_app" {
		_, _ = tx.Exec(ctx, `UPDATE notifications SET digest_batch_id=$1,delivery_mode=$2,delivery_status='pending' WHERE id=$3 AND user_id=$4`,
			dto.IDArg(batch.ID), input.Mode, dto.IDArg(input.Notification.ID), dto.IDArg(input.Notification.UserID))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, false, fmt.Errorf("commit queue digest: %w", err)
	}
	return batch, grouped, batchCreated, nil
}

func (r *Repository) ClaimDueDigestBatch(ctx context.Context, now time.Time) (*entity.NotificationDigestBatch, error) {
	query := `WITH candidate AS (
 SELECT id FROM notification_digest_batches
 WHERE status='pending' AND COALESCE(next_attempt_at,scheduled_for) <= $1
 ORDER BY COALESCE(next_attempt_at,scheduled_for),created_at
 FOR UPDATE SKIP LOCKED LIMIT 1
)
UPDATE notification_digest_batches b SET status='processing',attempts=b.attempts+1,updated_at=NOW()
FROM candidate WHERE b.id=candidate.id RETURNING ` + prefixedProjection("b", digestBatchProjection)
	batch, err := scanDigestBatch(r.db.QueryRow(ctx, query, now.UTC()))
	if errors.Is(err, domainerrs.ErrNotFound) {
		return nil, nil
	}
	return batch, err
}

func (r *Repository) GetDigestBatchByID(ctx context.Context, id uuid.UUID) (*entity.NotificationDigestBatch, error) {
	query, args, err := r.db.Builder.Select(digestBatchProjection).From("notification_digest_batches").Where(sq.Eq{"id": dto.IDArg(id)}).ToSql()
	if err != nil {
		return nil, err
	}
	batch, err := scanDigestBatch(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	batch.Items, err = r.listDigestItems(ctx, batch.ID)
	return batch, err
}

func (r *Repository) GetDigestBatchByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*entity.NotificationDigestBatch, error) {
	query, args, err := r.db.Builder.Select(digestBatchProjection).From("notification_digest_batches").Where(sq.Eq{"id": dto.IDArg(id), "user_id": dto.IDArg(userID)}).ToSql()
	if err != nil {
		return nil, err
	}
	batch, err := scanDigestBatch(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	batch.Items, err = r.listDigestItems(ctx, batch.ID)
	return batch, err
}

func (r *Repository) ListDigestBatchesByUser(ctx context.Context, input digests.ListInput) ([]entity.NotificationDigestBatch, error) {
	builder := r.db.Builder.Select(digestBatchProjection).From("notification_digest_batches").Where(sq.Eq{"user_id": dto.IDArg(input.UserID)})
	if input.Status == digests.StatusPending {
		builder = builder.Where(sq.Eq{"status": []string{digests.StatusPending, digests.StatusProcessing}})
	} else if input.Status == "history" {
		builder = builder.Where(sq.Eq{"status": []string{digests.StatusSent, digests.StatusFailed, digests.StatusCancelled}})
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query, args, err := builder.OrderBy("scheduled_for DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list digest batches: %w", err)
	}
	defer rows.Close()
	out := make([]entity.NotificationDigestBatch, 0)
	for rows.Next() {
		batch, err := scanDigestBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *batch)
	}
	return out, rows.Err()
}

func (r *Repository) MarkDigestBatchSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	_, err := r.db.Exec(ctx, `WITH updated_batch AS (
 UPDATE notification_digest_batches
 SET status='sent',sent_at=$2,next_attempt_at=NULL,error_code=NULL,error_message_safe=NULL,updated_at=NOW()
 WHERE id=$1 AND status='processing' RETURNING id
)
UPDATE notifications n SET delivery_status='sent'
FROM updated_batch b WHERE n.digest_batch_id=b.id`, dto.IDArg(id), sentAt.UTC())
	return err
}
func (r *Repository) MarkDigestBatchFailed(ctx context.Context, id uuid.UUID, retry bool, nextAttempt *time.Time, code, safeMessage string) error {
	status := digests.StatusFailed
	if retry {
		status = digests.StatusPending
	}
	deliveryStatus := "failed"
	if retry {
		deliveryStatus = "pending"
	}
	_, err := r.db.Exec(ctx, `WITH updated_batch AS (
 UPDATE notification_digest_batches
 SET status=$2,next_attempt_at=$3,error_code=$4,error_message_safe=$5,updated_at=NOW()
 WHERE id=$1 AND status='processing' RETURNING id
)
UPDATE notifications n SET delivery_status=$6
FROM updated_batch b WHERE n.digest_batch_id=b.id`, dto.IDArg(id), status, nextAttempt, code, safeMessage, deliveryStatus)
	return err
}

func (r *Repository) listDigestItems(ctx context.Context, batchID uuid.UUID) ([]entity.NotificationDigestItem, error) {
	query, args, err := r.db.Builder.Select(digestItemProjection).From("notification_digest_items").Where(sq.Eq{"batch_id": dto.IDArg(batchID)}).OrderBy("trip_id NULLS LAST", "category", "latest_event_at DESC").ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list digest items: %w", err)
	}
	defer rows.Close()
	out := make([]entity.NotificationDigestItem, 0)
	for rows.Next() {
		item, err := scanDigestItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func scanDigestBatch(row pgx.Row) (*entity.NotificationDigestBatch, error) {
	var value entity.NotificationDigestBatch
	if err := row.Scan(&value.ID, &value.UserID, &value.Channel, &value.Mode, &value.Status, &value.ScheduledFor, &value.SentAt, &value.Attempts, &value.NextAttemptAt, &value.ErrorCode, &value.ErrorMessageSafe, &value.CreatedAt, &value.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan digest batch: %w", err)
	}
	return &value, nil
}
func scanDigestItem(row pgx.Row) (*entity.NotificationDigestItem, error) {
	var value entity.NotificationDigestItem
	var raw []byte
	if err := row.Scan(&value.ID, &value.BatchID, &value.NotificationID, &value.UserID, &value.TripID, &value.Category, &value.Priority, &value.DigestKey, &value.Title, &value.Message, &raw, &value.EventCount, &value.LatestEventAt, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan digest item: %w", err)
	}
	if err := json.Unmarshal(raw, &value.Metadata); err != nil {
		return nil, fmt.Errorf("decode digest metadata: %w", err)
	}
	if value.Metadata == nil {
		value.Metadata = map[string]any{}
	}
	return &value, nil
}
func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return dto.IDArg(*value)
}
func prefixedProjection(prefix, projection string) string {
	out := ""
	start := 0
	for i := 0; i <= len(projection); i++ {
		if i == len(projection) || projection[i] == ',' {
			if out != "" {
				out += ","
			}
			out += prefix + "." + projection[start:i]
			start = i + 1
		}
	}
	return out
}
