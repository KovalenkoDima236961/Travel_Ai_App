package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateTripReminder(ctx context.Context, reminder *entity.TripReminder) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Insert("trip_reminders").
		Columns(dto.TripReminderInsertColumns()...).
		Values(dto.TripReminderInsertValues(reminder)...).
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) BatchCreateTripReminders(ctx context.Context, reminders []entity.TripReminder) ([]entity.TripReminder, error) {
	created := make([]entity.TripReminder, 0, len(reminders))
	for i := range reminders {
		reminder, err := r.CreateTripReminder(ctx, &reminders[i])
		if err != nil {
			return nil, err
		}
		created = append(created, *reminder)
	}
	return created, nil
}

func (r *Repository) GetTripReminderByID(ctx context.Context, tripID, reminderID uuid.UUID) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripReminderColumns).
		From("trip_reminders").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(reminderID),
		}).
		Where("deleted_at IS NULL").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripRemindersByTrip(ctx context.Context, tripID uuid.UUID, filters entity.TripReminderFilters) ([]entity.TripReminder, error) {
	builder := r.db.Builder.
		Select(dto.TripReminderColumns).
		From("trip_reminders").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		Where("deleted_at IS NULL")
	builder = applyReminderFilters(builder, filters)
	query, args, err := builder.
		OrderBy("trigger_date ASC", "trigger_time ASC NULLS FIRST", "created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip reminders: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip reminders: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripReminderRows(rows)
}

func (r *Repository) ListTripRemindersAssignedToUser(ctx context.Context, userID uuid.UUID, filters entity.TripReminderFilters) ([]entity.TripReminder, error) {
	filters.AssignedToUserID = &userID
	builder := r.db.Builder.
		Select(dto.TripReminderColumns).
		From("trip_reminders").
		Where("deleted_at IS NULL")
	builder = applyReminderFilters(builder, filters)
	query, args, err := builder.
		OrderBy("trigger_date ASC", "trigger_time ASC NULLS FIRST", "created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list assigned trip reminders: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query assigned trip reminders: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripReminderRows(rows)
}

func (r *Repository) ListDueTripReminders(ctx context.Context, now time.Time, limit int) ([]entity.TripReminder, error) {
	if limit <= 0 {
		limit = 100
	}
	query, args, err := r.db.Builder.
		Select(dto.TripReminderColumns).
		From("trip_reminders").
		Where(sq.Eq{"status": string(entity.ReminderStatusPending)}).
		Where("deleted_at IS NULL").
		Where(sq.LtOrEq{"trigger_date": now.UTC().Add(24 * time.Hour)}).
		OrderBy("trigger_date ASC", "trigger_time ASC NULLS FIRST", "created_at ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list due trip reminders: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query due trip reminders: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripReminderRows(rows)
}

func (r *Repository) ListRemindersByChecklistItemID(ctx context.Context, checklistItemID uuid.UUID) ([]entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripReminderColumns).
		From("trip_reminders").
		Where(sq.Eq{"checklist_item_id": dto.IDArg(checklistItemID)}).
		Where("deleted_at IS NULL").
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list reminders by checklist item: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query reminders by checklist item: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripReminderRows(rows)
}

func (r *Repository) UpdateTripReminder(ctx context.Context, reminder *entity.TripReminder) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("title", reminder.Title).
		Set("description", dto.TextArg(valueOrEmpty(reminder.Description))).
		Set("category", string(reminder.Category)).
		Set("priority", string(reminder.Priority)).
		Set("trigger_date", reminder.TriggerDate).
		Set("trigger_time", dtoTimeArg(reminder.TriggerTime)).
		Set("timezone", dto.TextArg(valueOrEmpty(reminder.Timezone))).
		Set("relative_offset_days", intPtrSQL(reminder.RelativeOffsetDays)).
		Set("assigned_to_user_id", dtoIDPtr(reminder.AssignedToUserID)).
		Set("related_day_number", intPtrSQL(reminder.RelatedDayNumber)).
		Set("related_item_index", intPtrSQL(reminder.RelatedItemIndex)).
		Set("related_item_id", dto.TextArg(valueOrEmpty(reminder.RelatedItemID))).
		Set("metadata", jsonMapSQL(reminder.Metadata)).
		Set("updated_by_user_id", dtoIDPtr(reminder.UpdatedByUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(reminder.TripID),
			"id":      dto.IDArg(reminder.ID),
		}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkTripReminderSent(ctx context.Context, tripID, reminderID uuid.UUID) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("status", string(entity.ReminderStatusSent)).
		Set("sent_at", sq.Expr("NOW()")).
		Set("failed_at", nil).
		Set("failure_reason", nil).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(reminderID),
			"status":  string(entity.ReminderStatusPending),
		}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark trip reminder sent: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkTripReminderFailed(ctx context.Context, tripID, reminderID uuid.UUID, reason string) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("status", string(entity.ReminderStatusFailed)).
		Set("failed_at", sq.Expr("NOW()")).
		Set("failure_reason", dto.TextArg(reason)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(reminderID),
			"status":  string(entity.ReminderStatusPending),
		}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark trip reminder failed: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CompleteTripReminder(ctx context.Context, tripID, reminderID, actorUserID uuid.UUID) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("status", string(entity.ReminderStatusCompleted)).
		Set("completed_at", sq.Expr("NOW()")).
		Set("completed_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(reminderID)}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build complete trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ReopenTripReminder(ctx context.Context, tripID, reminderID, actorUserID uuid.UUID) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("status", string(entity.ReminderStatusPending)).
		Set("completed_at", nil).
		Set("completed_by_user_id", nil).
		Set("sent_at", nil).
		Set("failed_at", nil).
		Set("failure_reason", nil).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(reminderID)}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build reopen trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) DisableTripReminder(ctx context.Context, tripID, reminderID, actorUserID uuid.UUID) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("status", string(entity.ReminderStatusDisabled)).
		Set("disabled_at", sq.Expr("NOW()")).
		Set("disabled_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(reminderID)}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build disable trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) EnableTripReminder(ctx context.Context, tripID, reminderID, actorUserID uuid.UUID) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("status", string(entity.ReminderStatusPending)).
		Set("disabled_at", nil).
		Set("disabled_by_user_id", nil).
		Set("failed_at", nil).
		Set("failure_reason", nil).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(reminderID)}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build enable trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) SoftDeleteTripReminder(ctx context.Context, tripID, reminderID, actorUserID uuid.UUID) (*entity.TripReminder, error) {
	query, args, err := r.db.Builder.
		Update("trip_reminders").
		Set("deleted_at", sq.Expr("NOW()")).
		Set("deleted_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "id": dto.IDArg(reminderID)}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripReminderColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build soft delete trip reminder: %w", err)
	}
	return dto.ScanTripReminder(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) DeleteGeneratedPendingRemindersForTrip(ctx context.Context, tripID, actorUserID uuid.UUID, categories []entity.ReminderCategory) (int64, error) {
	builder := r.db.Builder.
		Update("trip_reminders").
		Set("deleted_at", sq.Expr("NOW()")).
		Set("deleted_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		Where(sq.Eq{"status": string(entity.ReminderStatusPending)}).
		Where(sq.Eq{"source": []string{
			string(entity.ReminderSourceChecklist),
			string(entity.ReminderSourceRoute),
			string(entity.ReminderSourceTransport),
			string(entity.ReminderSourceAccommodation),
			string(entity.ReminderSourceWeather),
			string(entity.ReminderSourceSystem),
			string(entity.ReminderSourceRegenerated),
		}}).
		Where("deleted_at IS NULL")
	if len(categories) > 0 {
		values := make([]string, 0, len(categories))
		for _, category := range categories {
			values = append(values, string(category))
		}
		builder = builder.Where(sq.Eq{"category": values})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete generated pending reminders: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete generated pending reminders: %w", err)
	}
	return tag.RowsAffected(), nil
}

func applyReminderFilters(builder sq.SelectBuilder, filters entity.TripReminderFilters) sq.SelectBuilder {
	if filters.Status != nil {
		builder = builder.Where(sq.Eq{"status": string(*filters.Status)})
	}
	if filters.Category != nil {
		builder = builder.Where(sq.Eq{"category": string(*filters.Category)})
	}
	if filters.AssignedToUserID != nil {
		builder = builder.Where(sq.Eq{"assigned_to_user_id": dto.IDArg(*filters.AssignedToUserID)})
	}
	if filters.UpcomingOnly {
		builder = builder.Where(sq.GtOrEq{"trigger_date": time.Now().UTC()})
	}
	if filters.FromDate != nil {
		builder = builder.Where(sq.GtOrEq{"trigger_date": *filters.FromDate})
	}
	if filters.ToDate != nil {
		builder = builder.Where(sq.LtOrEq{"trigger_date": *filters.ToDate})
	}
	if filters.HighPriorityOnly {
		builder = builder.Where(sq.Eq{"priority": []string{string(entity.ReminderPriorityHigh), string(entity.ReminderPriorityCritical)}})
	}
	return builder
}

func dtoTimeArg(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}
