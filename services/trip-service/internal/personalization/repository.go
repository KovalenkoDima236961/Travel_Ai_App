package personalization

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

type PostgresRepository struct{ db *storage.DB }

func NewRepository(db *storage.DB) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) Create(ctx context.Context, feedback Feedback) (Feedback, error) {
	metadata, err := json.Marshal(feedback.Metadata)
	if err != nil {
		return Feedback{}, fmt.Errorf("marshal feedback metadata: %w", err)
	}
	query, args, err := r.db.Builder.Insert("personalization_feedback").Columns("id", "user_id", "workspace_id", "trip_id", "entity_type", "entity_id", "feedback_type", "feedback_value", "metadata_json").Values(feedback.ID, feedback.UserID, feedback.WorkspaceID, feedback.TripID, feedback.EntityType, nullString(feedback.EntityID), feedback.FeedbackType, nullString(feedback.FeedbackValue), metadata).Suffix("RETURNING id, user_id, workspace_id, trip_id, entity_type, entity_id, feedback_type, feedback_value, metadata_json, created_at").ToSql()
	if err != nil {
		return Feedback{}, fmt.Errorf("build create personalization feedback: %w", err)
	}
	return scanFeedback(r.db.QueryRow(ctx, query, args...))
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]Feedback, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	query, args, err := r.db.Builder.Select("id, user_id, workspace_id, trip_id, entity_type, entity_id, feedback_type, feedback_value, metadata_json, created_at").From("personalization_feedback").Where(sq.Eq{"user_id": userID}).OrderBy("created_at DESC", "id DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list personalization feedback: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query personalization feedback: %w", err)
	}
	defer rows.Close()
	out := []Feedback{}
	for rows.Next() {
		item, err := scanFeedback(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate personalization feedback: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) ClearByUser(ctx context.Context, userID uuid.UUID) error {
	query, args, err := r.db.Builder.Delete("personalization_feedback").Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return fmt.Errorf("build clear personalization feedback: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("clear personalization feedback: %w", err)
	}
	return nil
}

type row interface{ Scan(...any) error }

func scanFeedback(row row) (Feedback, error) {
	var id, userID, workspaceID, tripID pgtype.UUID
	var entityType, feedbackType string
	var entityID, feedbackValue pgtype.Text
	var metadata []byte
	var createdAt pgtype.Timestamp
	if err := row.Scan(&id, &userID, &workspaceID, &tripID, &entityType, &entityID, &feedbackType, &feedbackValue, &metadata, &createdAt); err != nil {
		return Feedback{}, fmt.Errorf("scan personalization feedback: %w", err)
	}
	out := Feedback{ID: uuid.UUID(id.Bytes), UserID: uuid.UUID(userID.Bytes), EntityType: entityType, EntityID: entityID.String, FeedbackType: FeedbackType(feedbackType), FeedbackValue: feedbackValue.String, Metadata: map[string]any{}, CreatedAt: createdAt.Time}
	if workspaceID.Valid {
		x := uuid.UUID(workspaceID.Bytes)
		out.WorkspaceID = &x
	}
	if tripID.Valid {
		x := uuid.UUID(tripID.Bytes)
		out.TripID = &x
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &out.Metadata); err != nil {
			return Feedback{}, fmt.Errorf("decode personalization feedback metadata: %w", err)
		}
	}
	return out, nil
}
func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

var _ pgx.Row
