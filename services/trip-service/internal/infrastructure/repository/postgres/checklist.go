package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) GetActiveChecklistByTripID(ctx context.Context, tripID uuid.UUID) (*entity.TripChecklist, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripChecklistColumns).
		From("trip_checklists").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.ChecklistStatusActive),
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active checklist: %w", err)
	}
	return dto.ScanTripChecklist(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetChecklistByID(ctx context.Context, checklistID uuid.UUID) (*entity.TripChecklist, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripChecklistColumns).
		From("trip_checklists").
		Where(sq.Eq{"id": dto.IDArg(checklistID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get checklist: %w", err)
	}
	return dto.ScanTripChecklist(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CreateChecklist(ctx context.Context, checklist *entity.TripChecklist) (*entity.TripChecklist, error) {
	query, args, err := r.db.Builder.
		Insert("trip_checklists").
		Columns(dto.TripChecklistInsertColumns()...).
		Values(dto.TripChecklistInsertValues(checklist)...).
		Suffix("RETURNING " + dto.TripChecklistColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create checklist: %w", err)
	}
	return dto.ScanTripChecklist(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateChecklist(ctx context.Context, checklist *entity.TripChecklist) (*entity.TripChecklist, error) {
	query, args, err := r.db.Builder.
		Update("trip_checklists").
		Set("title", checklist.Title).
		Set("summary", dto.TextArg(valueOrEmpty(checklist.Summary))).
		Set("generated_from_itinerary_revision", intPtrSQL(checklist.GeneratedFromItineraryRevision)).
		Set("generated_from_route_revision", intPtrSQL(checklist.GeneratedFromRouteRevision)).
		Set("generated_by_user_id", dtoIDPtr(checklist.GeneratedByUserID)).
		Set("updated_by_user_id", dtoIDPtr(checklist.UpdatedByUserID)).
		Set("metadata", jsonMapSQL(checklist.Metadata)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(checklist.ID)}).
		Suffix("RETURNING " + dto.TripChecklistColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update checklist: %w", err)
	}
	return dto.ScanTripChecklist(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ArchiveActiveChecklistForTrip(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.TripChecklist, error) {
	query, args, err := r.db.Builder.
		Update("trip_checklists").
		Set("status", string(entity.ChecklistStatusArchived)).
		Set("archived_at", sq.Expr("NOW()")).
		Set("archived_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"status":  string(entity.ChecklistStatusActive),
		}).
		Suffix("RETURNING " + dto.TripChecklistColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build archive active checklist: %w", err)
	}
	return dto.ScanTripChecklist(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CreateChecklistItem(ctx context.Context, item *entity.TripChecklistItem) (*entity.TripChecklistItem, error) {
	query, args, err := r.db.Builder.
		Insert("trip_checklist_items").
		Columns(dto.TripChecklistItemInsertColumns()...).
		Values(dto.TripChecklistItemInsertValues(item)...).
		Suffix("RETURNING " + dto.TripChecklistItemColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create checklist item: %w", err)
	}
	return dto.ScanTripChecklistItem(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) BatchCreateChecklistItems(ctx context.Context, items []entity.TripChecklistItem) ([]entity.TripChecklistItem, error) {
	created := make([]entity.TripChecklistItem, 0, len(items))
	for i := range items {
		item, err := r.CreateChecklistItem(ctx, &items[i])
		if err != nil {
			return nil, err
		}
		created = append(created, *item)
	}
	return created, nil
}

func (r *Repository) ListChecklistItemsByChecklist(ctx context.Context, checklistID uuid.UUID) ([]entity.TripChecklistItem, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripChecklistItemColumns).
		From("trip_checklist_items").
		Where(sq.Eq{"checklist_id": dto.IDArg(checklistID)}).
		Where("deleted_at IS NULL").
		OrderBy("sort_order ASC", "created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list checklist items: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query checklist items: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripChecklistItemRows(rows)
}

func (r *Repository) ListChecklistItemsByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripChecklistItem, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripChecklistItemColumns).
		From("trip_checklist_items").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		Where("deleted_at IS NULL").
		OrderBy("sort_order ASC", "created_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip checklist items: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip checklist items: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripChecklistItemRows(rows)
}

func (r *Repository) ListAssignedChecklistItemsByUser(ctx context.Context, userID uuid.UUID) ([]entity.TripChecklistItem, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripChecklistItemColumns).
		From("trip_checklist_items").
		Where(sq.Eq{"assigned_to_user_id": dto.IDArg(userID)}).
		Where("deleted_at IS NULL").
		OrderBy("due_date ASC NULLS LAST", "created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list assigned checklist items: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query assigned checklist items: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripChecklistItemRows(rows)
}

func (r *Repository) GetChecklistItemByID(ctx context.Context, tripID, itemID uuid.UUID) (*entity.TripChecklistItem, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripChecklistItemColumns).
		From("trip_checklist_items").
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(itemID),
		}).
		Where("deleted_at IS NULL").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get checklist item: %w", err)
	}
	return dto.ScanTripChecklistItem(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateChecklistItem(ctx context.Context, item *entity.TripChecklistItem) (*entity.TripChecklistItem, error) {
	query, args, err := r.db.Builder.
		Update("trip_checklist_items").
		Set("title", item.Title).
		Set("description", dto.TextArg(valueOrEmpty(item.Description))).
		Set("category", string(item.Category)).
		Set("item_type", string(item.ItemType)).
		Set("priority", string(item.Priority)).
		Set("quantity", intPtrSQL(item.Quantity)).
		Set("assigned_to_user_id", dtoIDPtr(item.AssignedToUserID)).
		Set("due_date", datePtrSQL(item.DueDate)).
		Set("reason", dto.TextArg(valueOrEmpty(item.Reason))).
		Set("related_day_number", intPtrSQL(item.RelatedDayNumber)).
		Set("related_item_index", intPtrSQL(item.RelatedItemIndex)).
		Set("related_item_id", dto.TextArg(valueOrEmpty(item.RelatedItemID))).
		Set("sort_order", item.SortOrder).
		Set("metadata", jsonMapSQL(item.Metadata)).
		Set("updated_by_user_id", dtoIDPtr(item.UpdatedByUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(item.TripID),
			"id":      dto.IDArg(item.ID),
		}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripChecklistItemColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update checklist item: %w", err)
	}
	return dto.ScanTripChecklistItem(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) SetChecklistItemChecked(ctx context.Context, tripID, itemID, actorUserID uuid.UUID, checked bool) (*entity.TripChecklistItem, error) {
	builder := r.db.Builder.
		Update("trip_checklist_items").
		Set("checked", checked).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(itemID),
		}).
		Where("deleted_at IS NULL")
	if checked {
		builder = builder.
			Set("checked_at", sq.Expr("NOW()")).
			Set("checked_by_user_id", dto.IDArg(actorUserID))
	} else {
		builder = builder.
			Set("checked_at", nil).
			Set("checked_by_user_id", nil)
	}
	query, args, err := builder.
		Suffix("RETURNING " + dto.TripChecklistItemColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build set checklist item checked: %w", err)
	}
	return dto.ScanTripChecklistItem(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) SoftDeleteChecklistItem(ctx context.Context, tripID, itemID, actorUserID uuid.UUID) (*entity.TripChecklistItem, error) {
	query, args, err := r.db.Builder.
		Update("trip_checklist_items").
		Set("deleted_at", sq.Expr("NOW()")).
		Set("deleted_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"trip_id": dto.IDArg(tripID),
			"id":      dto.IDArg(itemID),
		}).
		Where("deleted_at IS NULL").
		Suffix("RETURNING " + dto.TripChecklistItemColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build soft delete checklist item: %w", err)
	}
	return dto.ScanTripChecklistItem(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) SoftDeleteGeneratedChecklistItems(ctx context.Context, checklistID, actorUserID uuid.UUID, categories []entity.ChecklistCategory, preserveChecked bool) (int64, error) {
	builder := r.db.Builder.
		Update("trip_checklist_items").
		Set("deleted_at", sq.Expr("NOW()")).
		Set("deleted_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"checklist_id": dto.IDArg(checklistID)}).
		Where(sq.Eq{"source": []string{string(entity.ChecklistSourceAI), string(entity.ChecklistSourceRegenerated)}}).
		Where("deleted_at IS NULL")
	if preserveChecked {
		builder = builder.Where(sq.Eq{"checked": false})
	}
	if len(categories) > 0 {
		values := make([]string, 0, len(categories))
		for _, category := range categories {
			values = append(values, string(category))
		}
		builder = builder.Where(sq.Eq{"category": values})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build soft delete generated checklist items: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("soft delete generated checklist items: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) ReorderChecklistItems(ctx context.Context, tripID uuid.UUID, itemIDs []uuid.UUID, actorUserID uuid.UUID) error {
	for index, itemID := range itemIDs {
		query, args, err := r.db.Builder.
			Update("trip_checklist_items").
			Set("sort_order", index).
			Set("updated_by_user_id", dto.IDArg(actorUserID)).
			Set("updated_at", sq.Expr("NOW()")).
			Where(sq.Eq{
				"trip_id": dto.IDArg(tripID),
				"id":      dto.IDArg(itemID),
			}).
			Where("deleted_at IS NULL").
			ToSql()
		if err != nil {
			return fmt.Errorf("build reorder checklist item: %w", err)
		}
		if _, err := r.db.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("reorder checklist item: %w", err)
		}
	}
	return nil
}

func dtoIDPtr(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return dto.IDArg(*id)
}

func intPtrSQL(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func datePtrSQL(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func jsonMapSQL(value map[string]any) any {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return raw
}
