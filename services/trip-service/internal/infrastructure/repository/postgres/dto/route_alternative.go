package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/routealternatives"
)

const RouteAlternativeSessionColumns = "id, user_id, trip_id, workspace_id, source, prompt, output_language, status, request_json, response_json, selected_alternative_id, created_trip_id, applied_to_trip_id, parent_session_id, created_at, updated_at"

func ScanRouteAlternativeSession(row pgx.Row) (*routealternatives.Session, error) {
	var (
		id, userID, tripID, workspaceID                 pgtype.UUID
		createdTripID, appliedToTripID, parentSessionID pgtype.UUID
		source, outputLanguage, status                  string
		prompt, selectedAlternativeID                   pgtype.Text
		requestJSON, responseJSON                       []byte
		createdAt, updatedAt                            pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&userID,
		&tripID,
		&workspaceID,
		&source,
		&prompt,
		&outputLanguage,
		&status,
		&requestJSON,
		&responseJSON,
		&selectedAlternativeID,
		&createdTripID,
		&appliedToTripID,
		&parentSessionID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan route alternative session: %w", err)
	}
	return &routealternatives.Session{
		ID:                    uuid.UUID(id.Bytes),
		UserID:                uuid.UUID(userID.Bytes),
		TripID:                uuidPtr(tripID),
		WorkspaceID:           uuidPtr(workspaceID),
		Source:                source,
		Prompt:                textValue(prompt),
		OutputLanguage:        outputLanguage,
		Status:                status,
		RequestJSON:           requestJSON,
		ResponseJSON:          responseJSON,
		SelectedAlternativeID: textValue(selectedAlternativeID),
		CreatedTripID:         uuidPtr(createdTripID),
		AppliedToTripID:       uuidPtr(appliedToTripID),
		ParentSessionID:       uuidPtr(parentSessionID),
		CreatedAt:             createdAt.Time,
		UpdatedAt:             updatedAt.Time,
	}, nil
}

func ScanRouteAlternativeSessionRows(rows pgx.Rows) ([]routealternatives.Session, error) {
	items := make([]routealternatives.Session, 0)
	for rows.Next() {
		item, err := ScanRouteAlternativeSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate route alternative sessions: %w", err)
	}
	return items, nil
}
