package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const TripReminderColumns = "id, trip_id, title, description, category, priority, source, status, trigger_date, trigger_time, timezone, relative_offset_days, assigned_to_user_id, checklist_item_id, related_day_number, related_item_index, related_item_id, sent_at, completed_at, completed_by_user_id, disabled_at, disabled_by_user_id, cancelled_at, cancelled_by_user_id, failed_at, failure_reason, metadata, created_by_user_id, updated_by_user_id, created_at, updated_at, deleted_at, deleted_by_user_id"

func TripReminderInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"title",
		"description",
		"category",
		"priority",
		"source",
		"status",
		"trigger_date",
		"trigger_time",
		"timezone",
		"relative_offset_days",
		"assigned_to_user_id",
		"checklist_item_id",
		"related_day_number",
		"related_item_index",
		"related_item_id",
		"metadata",
		"created_by_user_id",
		"updated_by_user_id",
	}
}

func TripReminderInsertValues(reminder *entity.TripReminder) []any {
	return []any{
		IDArg(reminder.ID),
		IDArg(reminder.TripID),
		reminder.Title,
		textPtrArg(reminder.Description),
		string(reminder.Category),
		string(reminder.Priority),
		string(reminder.Source),
		string(reminder.Status),
		toPgDate(&reminder.TriggerDate),
		toPgTime(reminder.TriggerTime),
		textPtrArg(reminder.Timezone),
		intPtrArg(reminder.RelativeOffsetDays),
		toPgUUIDPtr(reminder.AssignedToUserID),
		toPgUUIDPtr(reminder.ChecklistItemID),
		intPtrArg(reminder.RelatedDayNumber),
		intPtrArg(reminder.RelatedItemIndex),
		textPtrArg(reminder.RelatedItemID),
		jsonArg(reminder.Metadata),
		toPgUUIDPtr(reminder.CreatedByUserID),
		toPgUUIDPtr(reminder.UpdatedByUserID),
	}
}

func ScanTripReminder(row pgx.Row) (*entity.TripReminder, error) {
	var (
		id, tripID, assignedToUserID, checklistItemID          pgtype.UUID
		completedByUserID, disabledByUserID, cancelledByUserID pgtype.UUID
		createdByUserID, updatedByUserID, deletedByUserID      pgtype.UUID
		title, category, priority, source, status              string
		description, timezone, relatedItemID, failureReason    pgtype.Text
		relativeOffsetDays, relatedDayNumber, relatedItemIndex pgtype.Int4
		triggerDate                                            pgtype.Date
		triggerTime                                            pgtype.Time
		sentAt, completedAt, disabledAt, cancelledAt, failedAt pgtype.Timestamp
		createdAt, updatedAt, deletedAt                        pgtype.Timestamp
		metadataRaw                                            []byte
	)
	err := row.Scan(
		&id,
		&tripID,
		&title,
		&description,
		&category,
		&priority,
		&source,
		&status,
		&triggerDate,
		&triggerTime,
		&timezone,
		&relativeOffsetDays,
		&assignedToUserID,
		&checklistItemID,
		&relatedDayNumber,
		&relatedItemIndex,
		&relatedItemID,
		&sentAt,
		&completedAt,
		&completedByUserID,
		&disabledAt,
		&disabledByUserID,
		&cancelledAt,
		&cancelledByUserID,
		&failedAt,
		&failureReason,
		&metadataRaw,
		&createdByUserID,
		&updatedByUserID,
		&createdAt,
		&updatedAt,
		&deletedAt,
		&deletedByUserID,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip reminder: %w", err)
	}
	trigger := fromPgDate(triggerDate)
	if trigger == nil {
		return nil, fmt.Errorf("scan trip reminder: trigger_date is null")
	}
	metadata, err := unmarshalMap(metadataRaw, "trip reminder metadata")
	if err != nil {
		return nil, err
	}
	return &entity.TripReminder{
		ID:                 uuid.UUID(id.Bytes),
		TripID:             uuid.UUID(tripID.Bytes),
		Title:              title,
		Description:        textPtr(description),
		Category:           entity.ReminderCategory(category),
		Priority:           entity.ReminderPriority(priority),
		Source:             entity.ReminderSource(source),
		Status:             entity.ReminderStatus(status),
		TriggerDate:        *trigger,
		TriggerTime:        fromPgTime(triggerTime),
		Timezone:           textPtr(timezone),
		RelativeOffsetDays: int4Ptr(relativeOffsetDays),
		AssignedToUserID:   fromPgUUID(assignedToUserID),
		ChecklistItemID:    fromPgUUID(checklistItemID),
		RelatedDayNumber:   int4Ptr(relatedDayNumber),
		RelatedItemIndex:   int4Ptr(relatedItemIndex),
		RelatedItemID:      textPtr(relatedItemID),
		SentAt:             timestampPtr(sentAt),
		CompletedAt:        timestampPtr(completedAt),
		CompletedByUserID:  fromPgUUID(completedByUserID),
		DisabledAt:         timestampPtr(disabledAt),
		DisabledByUserID:   fromPgUUID(disabledByUserID),
		CancelledAt:        timestampPtr(cancelledAt),
		CancelledByUserID:  fromPgUUID(cancelledByUserID),
		FailedAt:           timestampPtr(failedAt),
		FailureReason:      textPtr(failureReason),
		Metadata:           metadata,
		CreatedByUserID:    fromPgUUID(createdByUserID),
		UpdatedByUserID:    fromPgUUID(updatedByUserID),
		CreatedAt:          createdAt.Time,
		UpdatedAt:          updatedAt.Time,
		DeletedAt:          timestampPtr(deletedAt),
		DeletedByUserID:    fromPgUUID(deletedByUserID),
	}, nil
}

func ScanTripReminderRows(rows pgx.Rows) ([]entity.TripReminder, error) {
	reminders := make([]entity.TripReminder, 0)
	for rows.Next() {
		reminder, err := ScanTripReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, *reminder)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip reminders: %w", err)
	}
	return reminders, nil
}

func toPgTime(value *string) pgtype.Time {
	if value == nil || *value == "" {
		return pgtype.Time{Valid: false}
	}
	parsed, err := time.Parse("15:04", *value)
	if err != nil {
		return pgtype.Time{Valid: false}
	}
	microseconds := int64(parsed.Hour())*int64(time.Hour/time.Microsecond) +
		int64(parsed.Minute())*int64(time.Minute/time.Microsecond)
	return pgtype.Time{Microseconds: microseconds, Valid: true}
}

func fromPgTime(value pgtype.Time) *string {
	if !value.Valid {
		return nil
	}
	totalMinutes := value.Microseconds / int64(time.Minute/time.Microsecond)
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	formatted := fmt.Sprintf("%02d:%02d", hours, minutes)
	return &formatted
}
