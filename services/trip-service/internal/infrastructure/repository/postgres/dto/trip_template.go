package dto

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

const TripTemplateColumns = "id, workspace_id, created_by_user_id, source_trip_id, title, " +
	"description, destination_hint, duration_days, default_currency, visibility, template_json, " +
	"tags, estimated_total_amount, estimated_total_currency, status, created_at, updated_at, " +
	"archived_at, archived_by_user_id"

const TripTemplateSummaryColumns = "id, workspace_id, created_by_user_id, source_trip_id, title, " +
	"description, destination_hint, duration_days, default_currency, visibility, " +
	"NULL::jsonb AS template_json, tags, estimated_total_amount, estimated_total_currency, " +
	"status, created_at, updated_at, archived_at, archived_by_user_id"

func TripTemplateInsertColumns() []string {
	return []string{
		"id",
		"workspace_id",
		"created_by_user_id",
		"source_trip_id",
		"title",
		"description",
		"destination_hint",
		"duration_days",
		"default_currency",
		"visibility",
		"template_json",
		"tags",
		"estimated_total_amount",
		"estimated_total_currency",
		"status",
	}
}

func TripTemplateInsertValues(t *entity.TripTemplate) []any {
	return []any{
		IDArg(t.ID),
		toPgUUIDPtr(t.WorkspaceID),
		IDArg(t.CreatedByUserID),
		toPgUUIDPtr(t.SourceTripID),
		t.Title,
		toPgTextPtr(t.Description),
		toPgTextPtr(t.DestinationHint),
		t.DurationDays,
		toPgTextPtr(t.DefaultCurrency),
		string(t.Visibility),
		[]byte(t.TemplateJSON),
		t.Tags,
		NumericArg(t.EstimatedTotalAmount),
		toPgTextPtr(t.EstimatedTotalCurrency),
		string(t.Status),
	}
}

func ScanTripTemplate(row pgx.Row) (*entity.TripTemplate, error) {
	var (
		id, workspaceID, createdByUserID, sourceTripID, archivedByUserID pgtype.UUID
		title, visibility, status                                        string
		description, destinationHint, defaultCurrency                    pgtype.Text
		durationDays                                                     int32
		templateRaw                                                      []byte
		tags                                                             []string
		estimatedTotalAmount                                             pgtype.Numeric
		estimatedTotalCurrency                                           pgtype.Text
		createdAt, updatedAt, archivedAt                                 pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&workspaceID,
		&createdByUserID,
		&sourceTripID,
		&title,
		&description,
		&destinationHint,
		&durationDays,
		&defaultCurrency,
		&visibility,
		&templateRaw,
		&tags,
		&estimatedTotalAmount,
		&estimatedTotalCurrency,
		&status,
		&createdAt,
		&updatedAt,
		&archivedAt,
		&archivedByUserID,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip template: %w", err)
	}

	var raw json.RawMessage
	if len(templateRaw) > 0 {
		raw = json.RawMessage(templateRaw)
	}

	return &entity.TripTemplate{
		ID:                     uuid.UUID(id.Bytes),
		WorkspaceID:            fromPgUUID(workspaceID),
		CreatedByUserID:        uuid.UUID(createdByUserID.Bytes),
		SourceTripID:           fromPgUUID(sourceTripID),
		Title:                  title,
		Description:            fromPgText(description),
		DestinationHint:        fromPgText(destinationHint),
		DurationDays:           durationDays,
		DefaultCurrency:        fromPgText(defaultCurrency),
		Visibility:             entity.TripTemplateVisibility(visibility),
		TemplateJSON:           raw,
		Tags:                   tags,
		EstimatedTotalAmount:   fromPgNumeric(estimatedTotalAmount),
		EstimatedTotalCurrency: fromPgText(estimatedTotalCurrency),
		Status:                 entity.TripTemplateStatus(status),
		CreatedAt:              createdAt.Time,
		UpdatedAt:              updatedAt.Time,
		ArchivedAt:             fromPgTimestampPtr(archivedAt),
		ArchivedByUserID:       fromPgUUID(archivedByUserID),
	}, nil
}

func ScanTripTemplateRows(rows pgx.Rows) ([]entity.TripTemplate, error) {
	out := make([]entity.TripTemplate, 0)
	for rows.Next() {
		t, err := ScanTripTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip templates: %w", err)
	}
	return out, nil
}
