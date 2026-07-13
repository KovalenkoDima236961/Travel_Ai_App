package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const TripChecklistColumns = "id, trip_id, status, title, summary, generated_from_itinerary_revision, generated_from_route_revision, generated_by_user_id, created_by_user_id, updated_by_user_id, metadata, created_at, updated_at, archived_at, archived_by_user_id"

const TripChecklistItemColumns = "id, checklist_id, trip_id, title, description, category, item_type, priority, quantity, assigned_to_user_id, due_date, checked, checked_at, checked_by_user_id, source, reason, related_day_number, related_item_index, related_item_id, sort_order, metadata, created_by_user_id, updated_by_user_id, created_at, updated_at, deleted_at, deleted_by_user_id"

func TripChecklistInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"status",
		"title",
		"summary",
		"generated_from_itinerary_revision",
		"generated_from_route_revision",
		"generated_by_user_id",
		"created_by_user_id",
		"updated_by_user_id",
		"metadata",
	}
}

func TripChecklistInsertValues(checklist *entity.TripChecklist) []any {
	return []any{
		IDArg(checklist.ID),
		IDArg(checklist.TripID),
		string(checklist.Status),
		checklist.Title,
		textPtrArg(checklist.Summary),
		intPtrArg(checklist.GeneratedFromItineraryRevision),
		intPtrArg(checklist.GeneratedFromRouteRevision),
		toPgUUIDPtr(checklist.GeneratedByUserID),
		IDArg(checklist.CreatedByUserID),
		toPgUUIDPtr(checklist.UpdatedByUserID),
		jsonArg(checklist.Metadata),
	}
}

func TripChecklistItemInsertColumns() []string {
	return []string{
		"id",
		"checklist_id",
		"trip_id",
		"title",
		"description",
		"category",
		"item_type",
		"priority",
		"quantity",
		"assigned_to_user_id",
		"due_date",
		"checked",
		"checked_at",
		"checked_by_user_id",
		"source",
		"reason",
		"related_day_number",
		"related_item_index",
		"related_item_id",
		"sort_order",
		"metadata",
		"created_by_user_id",
		"updated_by_user_id",
	}
}

func TripChecklistItemInsertValues(item *entity.TripChecklistItem) []any {
	return []any{
		IDArg(item.ID),
		IDArg(item.ChecklistID),
		IDArg(item.TripID),
		item.Title,
		textPtrArg(item.Description),
		string(item.Category),
		string(item.ItemType),
		string(item.Priority),
		intPtrArg(item.Quantity),
		toPgUUIDPtr(item.AssignedToUserID),
		toPgDate(item.DueDate),
		item.Checked,
		timestampArg(item.CheckedAt),
		toPgUUIDPtr(item.CheckedByUserID),
		string(item.Source),
		textPtrArg(item.Reason),
		intPtrArg(item.RelatedDayNumber),
		intPtrArg(item.RelatedItemIndex),
		textPtrArg(item.RelatedItemID),
		item.SortOrder,
		jsonArg(item.Metadata),
		toPgUUIDPtr(item.CreatedByUserID),
		toPgUUIDPtr(item.UpdatedByUserID),
	}
}

func ScanTripChecklist(row pgx.Row) (*entity.TripChecklist, error) {
	var (
		id, tripID, generatedByUserID, createdByUserID, updatedByUserID, archivedByUserID pgtype.UUID
		status, title                                                                     string
		summary                                                                           pgtype.Text
		generatedFromItineraryRevision, generatedFromRouteRevision                        pgtype.Int4
		metadataRaw                                                                       []byte
		createdAt, updatedAt, archivedAt                                                  pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&tripID,
		&status,
		&title,
		&summary,
		&generatedFromItineraryRevision,
		&generatedFromRouteRevision,
		&generatedByUserID,
		&createdByUserID,
		&updatedByUserID,
		&metadataRaw,
		&createdAt,
		&updatedAt,
		&archivedAt,
		&archivedByUserID,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip checklist: %w", err)
	}
	metadata, err := unmarshalMap(metadataRaw, "trip checklist metadata")
	if err != nil {
		return nil, err
	}
	return &entity.TripChecklist{
		ID:                             uuid.UUID(id.Bytes),
		TripID:                         uuid.UUID(tripID.Bytes),
		Status:                         entity.ChecklistStatus(status),
		Title:                          title,
		Summary:                        textPtr(summary),
		GeneratedFromItineraryRevision: int4Ptr(generatedFromItineraryRevision),
		GeneratedFromRouteRevision:     int4Ptr(generatedFromRouteRevision),
		GeneratedByUserID:              fromPgUUID(generatedByUserID),
		CreatedByUserID:                uuid.UUID(createdByUserID.Bytes),
		UpdatedByUserID:                fromPgUUID(updatedByUserID),
		Metadata:                       metadata,
		CreatedAt:                      createdAt.Time,
		UpdatedAt:                      updatedAt.Time,
		ArchivedAt:                     timestampPtr(archivedAt),
		ArchivedByUserID:               fromPgUUID(archivedByUserID),
	}, nil
}

func ScanTripChecklistItem(row pgx.Row) (*entity.TripChecklistItem, error) {
	var (
		id, checklistID, tripID, assignedToUserID, checkedByUserID pgtype.UUID
		createdByUserID, updatedByUserID, deletedByUserID          pgtype.UUID
		title, category, itemType, priority, source                string
		description, reason, relatedItemID                         pgtype.Text
		quantity, relatedDayNumber, relatedItemIndex               pgtype.Int4
		dueDate                                                    pgtype.Date
		checked                                                    bool
		sortOrder                                                  int32
		checkedAt, createdAt, updatedAt, deletedAt                 pgtype.Timestamp
		metadataRaw                                                []byte
	)
	err := row.Scan(
		&id,
		&checklistID,
		&tripID,
		&title,
		&description,
		&category,
		&itemType,
		&priority,
		&quantity,
		&assignedToUserID,
		&dueDate,
		&checked,
		&checkedAt,
		&checkedByUserID,
		&source,
		&reason,
		&relatedDayNumber,
		&relatedItemIndex,
		&relatedItemID,
		&sortOrder,
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
		return nil, fmt.Errorf("scan trip checklist item: %w", err)
	}
	metadata, err := unmarshalMap(metadataRaw, "trip checklist item metadata")
	if err != nil {
		return nil, err
	}
	return &entity.TripChecklistItem{
		ID:               uuid.UUID(id.Bytes),
		ChecklistID:      uuid.UUID(checklistID.Bytes),
		TripID:           uuid.UUID(tripID.Bytes),
		Title:            title,
		Description:      textPtr(description),
		Category:         entity.ChecklistCategory(category),
		ItemType:         entity.ChecklistItemType(itemType),
		Priority:         entity.ChecklistPriority(priority),
		Quantity:         int4Ptr(quantity),
		AssignedToUserID: fromPgUUID(assignedToUserID),
		DueDate:          fromPgDate(dueDate),
		Checked:          checked,
		CheckedAt:        timestampPtr(checkedAt),
		CheckedByUserID:  fromPgUUID(checkedByUserID),
		Source:           entity.ChecklistSource(source),
		Reason:           textPtr(reason),
		RelatedDayNumber: int4Ptr(relatedDayNumber),
		RelatedItemIndex: int4Ptr(relatedItemIndex),
		RelatedItemID:    textPtr(relatedItemID),
		SortOrder:        int(sortOrder),
		Metadata:         metadata,
		CreatedByUserID:  fromPgUUID(createdByUserID),
		UpdatedByUserID:  fromPgUUID(updatedByUserID),
		CreatedAt:        createdAt.Time,
		UpdatedAt:        updatedAt.Time,
		DeletedAt:        timestampPtr(deletedAt),
		DeletedByUserID:  fromPgUUID(deletedByUserID),
	}, nil
}

func ScanTripChecklistItemRows(rows pgx.Rows) ([]entity.TripChecklistItem, error) {
	items := make([]entity.TripChecklistItem, 0)
	for rows.Next() {
		item, err := ScanTripChecklistItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip checklist items: %w", err)
	}
	return items, nil
}

func textPtrArg(value *string) pgtype.Text {
	if value == nil || *value == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func intPtrArg(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func IntPtrArg(value *int) pgtype.Int4 {
	return intPtrArg(value)
}

func int4Ptr(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int32)
	return &v
}

func timestampArg(value *time.Time) pgtype.Timestamp {
	if value == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *value, Valid: true}
}

func jsonArg(value map[string]any) []byte {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

func JSONArg(value map[string]any) []byte {
	return jsonArg(value)
}

func unmarshalMap(raw []byte, label string) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", label, err)
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}
