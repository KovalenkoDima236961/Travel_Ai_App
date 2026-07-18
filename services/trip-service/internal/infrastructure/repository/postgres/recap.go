package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const recapColumns = "id, trip_id, created_by_user_id, updated_by_user_id, status, recap_json, source_summary_json, ai_metadata_json, finalized_at, archived_at, created_at, updated_at"
const recapFeedbackColumns = "id, trip_id, recap_id, user_id, feedback_type, entity_type, entity_id, label, value, approved_for_personalization, metadata_json, created_at, updated_at"

func (r *Repository) CreateTripRecap(ctx context.Context, recap *entity.TripRecap) (*entity.TripRecap, error) {
	query, args, err := r.db.Builder.Insert("trip_recaps").
		Columns("id", "trip_id", "created_by_user_id", "updated_by_user_id", "status", "recap_json", "source_summary_json", "ai_metadata_json", "finalized_at", "archived_at").
		Values(recap.ID, recap.TripID, recap.CreatedByUserID, recap.UpdatedByUserID, string(recap.Status), recap.RecapJSON, nullableJSON(recap.SourceSummary), nullableJSON(recap.AIMetadata), recap.FinalizedAt, recap.ArchivedAt).
		Suffix("RETURNING " + recapColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip recap: %w", err)
	}
	return scanTripRecap(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetActiveTripRecap(ctx context.Context, tripID uuid.UUID) (*entity.TripRecap, error) {
	query, args, err := r.db.Builder.Select(recapColumns).From("trip_recaps").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).Where("archived_at IS NULL").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active trip recap: %w", err)
	}
	return scanTripRecap(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateTripRecap(ctx context.Context, recap *entity.TripRecap) (*entity.TripRecap, error) {
	query, args, err := r.db.Builder.Update("trip_recaps").
		Set("updated_by_user_id", recap.UpdatedByUserID).
		Set("status", string(recap.Status)).
		Set("recap_json", recap.RecapJSON).
		Set("source_summary_json", nullableJSON(recap.SourceSummary)).
		Set("ai_metadata_json", nullableJSON(recap.AIMetadata)).
		Set("finalized_at", recap.FinalizedAt).
		Set("archived_at", recap.ArchivedAt).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": recap.ID, "trip_id": recap.TripID}).
		Suffix("RETURNING " + recapColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip recap: %w", err)
	}
	return scanTripRecap(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ArchiveTripRecap(ctx context.Context, tripID, recapID, actorUserID uuid.UUID) (*entity.TripRecap, error) {
	query, args, err := r.db.Builder.Update("trip_recaps").
		Set("status", string(entity.TripRecapStatusArchived)).
		Set("updated_by_user_id", actorUserID).
		Set("archived_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": recapID, "trip_id": tripID}).Where("archived_at IS NULL").
		Suffix("RETURNING " + recapColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build archive trip recap: %w", err)
	}
	return scanTripRecap(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CreateTripRecapFeedback(ctx context.Context, feedback *entity.TripRecapFeedback) (*entity.TripRecapFeedback, error) {
	metadata, err := json.Marshal(feedback.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal trip recap feedback metadata: %w", err)
	}
	query, args, err := r.db.Builder.Insert("trip_recap_feedback").
		Columns("id", "trip_id", "recap_id", "user_id", "feedback_type", "entity_type", "entity_id", "label", "value", "approved_for_personalization", "metadata_json").
		Values(feedback.ID, feedback.TripID, feedback.RecapID, feedback.UserID, feedback.FeedbackType, feedback.EntityType, feedback.EntityID, feedback.Label, feedback.Value, feedback.ApprovedForPersonalization, metadata).
		Suffix("RETURNING " + recapFeedbackColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create trip recap feedback: %w", err)
	}
	return scanTripRecapFeedback(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripRecapFeedback(ctx context.Context, recapID uuid.UUID) ([]entity.TripRecapFeedback, error) {
	query, args, err := r.db.Builder.Select(recapFeedbackColumns).From("trip_recap_feedback").
		Where(sq.Eq{"recap_id": recapID}).OrderBy("created_at ASC", "id ASC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip recap feedback: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip recap feedback: %w", err)
	}
	defer rows.Close()
	items := []entity.TripRecapFeedback{}
	for rows.Next() {
		item, scanErr := scanTripRecapFeedback(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip recap feedback: %w", err)
	}
	return items, nil
}

func (r *Repository) ApproveTripRecapFeedback(ctx context.Context, recapID, feedbackID, userID uuid.UUID) (*entity.TripRecapFeedback, error) {
	query, args, err := r.db.Builder.Update("trip_recap_feedback").
		Set("approved_for_personalization", true).Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": feedbackID, "recap_id": recapID, "user_id": userID}).
		Suffix("RETURNING " + recapFeedbackColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build approve trip recap feedback: %w", err)
	}
	return scanTripRecapFeedback(r.db.QueryRow(ctx, query, args...))
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 || string(value) == "null" {
		return nil
	}
	return []byte(value)
}

type recapScanner interface{ Scan(...any) error }

func scanTripRecap(row recapScanner) (*entity.TripRecap, error) {
	var id, tripID, createdBy, updatedBy pgtype.UUID
	var status string
	var recapJSON, sourceJSON, metadataJSON []byte
	var finalizedAt, archivedAt, createdAt, updatedAt pgtype.Timestamp
	if err := row.Scan(&id, &tripID, &createdBy, &updatedBy, &status, &recapJSON, &sourceJSON, &metadataJSON, &finalizedAt, &archivedAt, &createdAt, &updatedAt); err != nil {
		if storage.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip recap: %w", err)
	}
	result := &entity.TripRecap{ID: uuid.UUID(id.Bytes), TripID: uuid.UUID(tripID.Bytes), CreatedByUserID: uuid.UUID(createdBy.Bytes), Status: entity.TripRecapStatus(status), RecapJSON: json.RawMessage(recapJSON), SourceSummary: json.RawMessage(sourceJSON), AIMetadata: json.RawMessage(metadataJSON), CreatedAt: createdAt.Time, UpdatedAt: updatedAt.Time}
	if updatedBy.Valid {
		value := uuid.UUID(updatedBy.Bytes)
		result.UpdatedByUserID = &value
	}
	if finalizedAt.Valid {
		value := finalizedAt.Time
		result.FinalizedAt = &value
	}
	if archivedAt.Valid {
		value := archivedAt.Time
		result.ArchivedAt = &value
	}
	return result, nil
}

func scanTripRecapFeedback(row recapScanner) (*entity.TripRecapFeedback, error) {
	var id, tripID, recapID, userID pgtype.UUID
	var feedbackType, label string
	var entityType, entityID, value pgtype.Text
	var approved bool
	var metadata []byte
	var createdAt, updatedAt pgtype.Timestamp
	if err := row.Scan(&id, &tripID, &recapID, &userID, &feedbackType, &entityType, &entityID, &label, &value, &approved, &metadata, &createdAt, &updatedAt); err != nil {
		if storage.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip recap feedback: %w", err)
	}
	result := &entity.TripRecapFeedback{ID: uuid.UUID(id.Bytes), TripID: uuid.UUID(tripID.Bytes), RecapID: uuid.UUID(recapID.Bytes), UserID: uuid.UUID(userID.Bytes), FeedbackType: feedbackType, Label: label, ApprovedForPersonalization: approved, Metadata: map[string]any{}, CreatedAt: createdAt.Time, UpdatedAt: updatedAt.Time}
	if entityType.Valid {
		v := entityType.String
		result.EntityType = &v
	}
	if entityID.Valid {
		v := entityID.String
		result.EntityID = &v
	}
	if value.Valid {
		v := value.String
		result.Value = &v
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &result.Metadata); err != nil {
			return nil, fmt.Errorf("decode trip recap feedback metadata: %w", err)
		}
	}
	return result, nil
}

var _ pgx.Row
