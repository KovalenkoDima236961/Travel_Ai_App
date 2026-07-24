package aidataset

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

type Repository struct {
	db *storage.DB
}

func NewRepository(db *storage.DB) *Repository {
	return &Repository{db: db}
}

const projectColumns = "id, key, name, description, task_type, schema_version, status, minimum_quality_score, consent_required, created_by_user_id, created_at, updated_at, archived_at"

const consentColumns = "id, user_id, scope_type, scope_id, consent_version, status, granted_at, revoked_at, created_at, updated_at"

const exampleColumns = "id, dataset_project_id, source_type, source_entity_type, source_entity_id, user_id, trip_id, task_type, language, schema_version, input_json, grounding_json, expected_output_json, negative_output_json, labels_json, provenance_json, consent_status, consent_record_id, sanitization_status, quality_status, review_status, quality_score, duplicate_group_id, split, export_status, created_at, updated_at, reviewed_by_user_id, reviewed_at, rejected_reason"

const versionColumns = "id, dataset_project_id, version, status, schema_version, example_count, train_count, validation_count, test_count, holdout_count, manifest_json, checksum, export_path, created_by_user_id, created_at, finalized_at, invalidated_at, invalidated_reason"

func (r *Repository) CreateProject(ctx context.Context, input CreateProjectInput, actorUserID *uuid.UUID) (*DatasetProject, error) {
	if input.SchemaVersion == "" {
		input.SchemaVersion = SchemaVersion
	}
	if input.MinimumQualityScore <= 0 {
		input.MinimumQualityScore = 0.8
	}
	query, args, err := r.db.Builder.
		Insert("ai_dataset_projects").
		Columns("id", "key", "name", "description", "task_type", "schema_version", "status", "minimum_quality_score", "consent_required", "created_by_user_id").
		Values(uuid.New(), strings.TrimSpace(input.Key), strings.TrimSpace(input.Name), textPtr(input.Description), strings.TrimSpace(input.TaskType), strings.TrimSpace(input.SchemaVersion), ProjectStatusDraft, input.MinimumQualityScore, input.ConsentRequired, uuidPtrArg(actorUserID)).
		Suffix("ON CONFLICT (key) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, task_type = EXCLUDED.task_type, schema_version = EXCLUDED.schema_version, minimum_quality_score = EXCLUDED.minimum_quality_score, consent_required = EXCLUDED.consent_required, updated_at = NOW() RETURNING " + projectColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create dataset project: %w", err)
	}
	return scanProject(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListProjects(ctx context.Context) ([]DatasetProject, error) {
	query, args, err := r.db.Builder.
		Select(projectColumns).
		From("ai_dataset_projects").
		OrderBy("created_at DESC", "key ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list dataset projects: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query dataset projects: %w", err)
	}
	defer rows.Close()
	return scanProjectRows(rows)
}

func (r *Repository) GetProject(ctx context.Context, id uuid.UUID) (*DatasetProject, error) {
	query, args, err := r.db.Builder.
		Select(projectColumns).
		From("ai_dataset_projects").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get dataset project: %w", err)
	}
	return scanProject(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetProjectByKey(ctx context.Context, key string) (*DatasetProject, error) {
	query, args, err := r.db.Builder.
		Select(projectColumns).
		From("ai_dataset_projects").
		Where(sq.Eq{"key": strings.TrimSpace(key)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get dataset project by key: %w", err)
	}
	return scanProject(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CreateConsent(ctx context.Context, consent TrainingConsent) (*TrainingConsent, error) {
	if consent.ID == uuid.Nil {
		consent.ID = uuid.New()
	}
	query, args, err := r.db.Builder.
		Insert("ai_training_consents").
		Columns("id", "user_id", "scope_type", "scope_id", "consent_version", "status", "granted_at", "revoked_at").
		Values(consent.ID, consent.UserID, consent.ScopeType, textPtr(consent.ScopeID), consent.ConsentVersion, consent.Status, timePtrArg(consent.GrantedAt), timePtrArg(consent.RevokedAt)).
		Suffix("RETURNING " + consentColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create training consent: %w", err)
	}
	return scanConsent(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) LatestConsent(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *string) (*TrainingConsent, error) {
	builder := r.db.Builder.
		Select(consentColumns).
		From("ai_training_consents").
		Where(sq.Eq{"user_id": userID, "scope_type": scopeType}).
		OrderBy("created_at DESC", "id DESC").
		Limit(1)
	if scopeID == nil || strings.TrimSpace(*scopeID) == "" {
		builder = builder.Where("scope_id IS NULL")
	} else {
		builder = builder.Where(sq.Eq{"scope_id": strings.TrimSpace(*scopeID)})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest training consent: %w", err)
	}
	consent, err := scanConsent(r.db.QueryRow(ctx, query, args...))
	if err == domainerrs.ErrNotFound {
		return nil, nil
	}
	return consent, err
}

func (r *Repository) MarkUserConsentRevoked(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.MarkConsentRevoked(ctx, userID, ScopeGlobalFutureExamples, nil)
}

func (r *Repository) MarkConsentRevoked(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *string) (int64, error) {
	predicate, args, err := consentRevocationPredicate(userID, scopeType, scopeID)
	if err != nil {
		return 0, err
	}
	updateQuery := `
UPDATE ai_training_examples
SET consent_status = 'revoked',
    review_status = CASE WHEN review_status = 'approved' THEN 'needs_changes' ELSE review_status END,
    export_status = CASE WHEN export_status = 'exported' THEN 'invalidated' ELSE export_status END,
    updated_at = NOW()
WHERE ` + predicate + ` AND consent_status IN ('pending', 'granted')`
	tag, err := r.db.Exec(ctx, updateQuery, args...)
	if err != nil {
		return 0, fmt.Errorf("mark examples consent revoked: %w", err)
	}
	if tag.RowsAffected() > 0 {
		invalidateQuery := `
UPDATE ai_dataset_versions
SET status = 'invalidated',
    invalidated_at = NOW(),
    invalidated_reason = 'training consent revoked'
WHERE status IN ('ready', 'exported')
  AND dataset_project_id IN (
      SELECT DISTINCT dataset_project_id FROM ai_training_examples WHERE ` + predicate + `
  )`
		_, err = r.db.Exec(ctx, invalidateQuery, args...)
		if err != nil {
			return tag.RowsAffected(), fmt.Errorf("invalidate dataset versions after consent revocation: %w", err)
		}
	}
	return tag.RowsAffected(), nil
}

func consentRevocationPredicate(userID uuid.UUID, scopeType string, scopeID *string) (string, []any, error) {
	scopeType = strings.TrimSpace(scopeType)
	if scopeType == "" {
		scopeType = ScopeGlobalFutureExamples
	}
	args := []any{userID}
	switch scopeType {
	case ScopeGlobalFutureExamples:
		return "user_id = $1", args, nil
	case ScopeTrip:
		if scopeID == nil || strings.TrimSpace(*scopeID) == "" {
			return "", nil, fmt.Errorf("trip scope id is required")
		}
		tripID, err := uuid.Parse(strings.TrimSpace(*scopeID))
		if err != nil {
			return "", nil, fmt.Errorf("parse trip scope id: %w", err)
		}
		args = append(args, tripID)
		return "user_id = $1 AND trip_id = $2", args, nil
	case ScopeItineraryVersion:
		if scopeID == nil || strings.TrimSpace(*scopeID) == "" {
			return "", nil, fmt.Errorf("itinerary version scope id is required")
		}
		args = append(args, strings.TrimSpace(*scopeID))
		return "user_id = $1 AND (source_entity_id = $2 OR provenance_json->>'itineraryVersionId' = $2)", args, nil
	default:
		if scopeID == nil || strings.TrimSpace(*scopeID) == "" {
			return "user_id = $1", args, nil
		}
		args = append(args, strings.TrimSpace(*scopeID))
		return "user_id = $1 AND source_entity_id = $2", args, nil
	}
}

func (r *Repository) CreateTrainingExample(ctx context.Context, input CreateExampleInput) (*TrainingExample, error) {
	if input.SchemaVersion == "" {
		input.SchemaVersion = SchemaVersion
	}
	if input.Language == "" {
		input.Language = "en"
	}
	if input.ConsentStatus == "" {
		input.ConsentStatus = ConsentNotRequired
	}
	query, args, err := r.db.Builder.
		Insert("ai_training_examples").
		Columns(
			"id",
			"dataset_project_id",
			"source_type",
			"source_entity_type",
			"source_entity_id",
			"user_id",
			"trip_id",
			"task_type",
			"language",
			"schema_version",
			"input_json",
			"grounding_json",
			"expected_output_json",
			"negative_output_json",
			"labels_json",
			"provenance_json",
			"consent_status",
			"consent_record_id",
		).
		Values(
			uuid.New(),
			input.DatasetProjectID,
			strings.TrimSpace(input.SourceType),
			textPtr(input.SourceEntityType),
			textPtr(input.SourceEntityID),
			uuidPtrArg(input.UserID),
			uuidPtrArg(input.TripID),
			strings.TrimSpace(input.TaskType),
			strings.TrimSpace(input.Language),
			strings.TrimSpace(input.SchemaVersion),
			rawOrEmptyObject(input.InputJSON),
			rawOrNull(input.GroundingJSON),
			rawOrEmptyObject(input.ExpectedOutputJSON),
			rawOrNull(input.NegativeOutputJSON),
			rawOrEmptyObject(input.LabelsJSON),
			rawOrEmptyObject(input.ProvenanceJSON),
			input.ConsentStatus,
			uuidPtrArg(input.ConsentRecordID),
		).
		Suffix("RETURNING " + exampleColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create training example: %w", err)
	}
	return scanExample(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) FindExampleBySource(ctx context.Context, projectID uuid.UUID, sourceType string, sourceEntityID string) (*TrainingExample, error) {
	query, args, err := r.db.Builder.
		Select(exampleColumns).
		From("ai_training_examples").
		Where(sq.Eq{"dataset_project_id": projectID, "source_type": sourceType, "source_entity_id": sourceEntityID}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find example by source: %w", err)
	}
	example, err := scanExample(r.db.QueryRow(ctx, query, args...))
	if err == domainerrs.ErrNotFound {
		return nil, nil
	}
	return example, err
}

func (r *Repository) GetExample(ctx context.Context, id uuid.UUID) (*TrainingExample, error) {
	query, args, err := r.db.Builder.
		Select(exampleColumns).
		From("ai_training_examples").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get training example: %w", err)
	}
	return scanExample(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListExamples(ctx context.Context, filters ExampleFilters) ([]TrainingExample, error) {
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	if filters.Limit > 200 {
		filters.Limit = 200
	}
	builder := r.db.Builder.
		Select(exampleColumns).
		From("ai_training_examples").
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(filters.Limit)).
		Offset(uint64(maxInt(filters.Offset, 0)))
	if filters.DatasetProjectID != nil {
		builder = builder.Where(sq.Eq{"dataset_project_id": *filters.DatasetProjectID})
	}
	if filters.ReviewStatus != "" {
		builder = builder.Where(sq.Eq{"review_status": filters.ReviewStatus})
	}
	if filters.SanitizationStatus != "" {
		builder = builder.Where(sq.Eq{"sanitization_status": filters.SanitizationStatus})
	}
	if filters.QualityStatus != "" {
		builder = builder.Where(sq.Eq{"quality_status": filters.QualityStatus})
	}
	if filters.ConsentStatus != "" {
		builder = builder.Where(sq.Eq{"consent_status": filters.ConsentStatus})
	}
	if filters.Language != "" {
		builder = builder.Where(sq.Eq{"language": filters.Language})
	}
	if filters.TaskType != "" {
		builder = builder.Where(sq.Eq{"task_type": filters.TaskType})
	}
	if filters.Split != "" {
		builder = builder.Where(sq.Eq{"split": filters.Split})
	}
	if filters.SourceType != "" {
		builder = builder.Where(sq.Eq{"source_type": filters.SourceType})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list training examples: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query training examples: %w", err)
	}
	defer rows.Close()
	return scanExampleRows(rows)
}

func (r *Repository) UpdateExampleSanitization(ctx context.Context, id uuid.UUID, result SanitizationResult) (*TrainingExample, error) {
	labels := mustJSON(map[string]any{"privateDataDetected": result.PrivateDetected})
	query, args, err := r.db.Builder.
		Update("ai_training_examples").
		Set("input_json", rawOrEmptyObject(result.InputJSON)).
		Set("grounding_json", rawOrNull(result.GroundingJSON)).
		Set("expected_output_json", rawOrEmptyObject(result.OutputJSON)).
		Set("sanitization_status", result.Status).
		Set("labels_json", sq.Expr("labels_json || ?::jsonb", labels)).
		Set("provenance_json", sq.Expr("provenance_json || jsonb_build_object('sanitization', ?::jsonb)", SanitizationMetadata(result))).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING " + exampleColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update sanitization: %w", err)
	}
	return scanExample(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateExampleQuality(ctx context.Context, id uuid.UUID, result QualityResult, duplicateGroupID uuid.UUID) (*TrainingExample, error) {
	query, args, err := r.db.Builder.
		Update("ai_training_examples").
		Set("quality_status", result.Status).
		Set("quality_score", result.Score).
		Set("duplicate_group_id", duplicateGroupID).
		Set("provenance_json", sq.Expr("provenance_json || jsonb_build_object('quality', ?::jsonb)", mustJSON(result))).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING " + exampleColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update quality: %w", err)
	}
	return scanExample(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateExampleReview(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID, reason string) (*TrainingExample, error) {
	var rejectedReason any
	if status == ReviewRejected || status == ReviewNeedsChanges {
		rejectedReason = strings.TrimSpace(reason)
	}
	query, args, err := r.db.Builder.
		Update("ai_training_examples").
		Set("review_status", status).
		Set("reviewed_by_user_id", reviewerID).
		Set("reviewed_at", sq.Expr("NOW()")).
		Set("rejected_reason", rejectedReason).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING " + exampleColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update example review: %w", err)
	}
	return scanExample(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListApprovedExamples(ctx context.Context, projectID uuid.UUID, minimumQualityScore float64, languages, taskTypes []string) ([]TrainingExample, error) {
	builder := r.db.Builder.
		Select(exampleColumns).
		From("ai_training_examples").
		Where(sq.Eq{
			"dataset_project_id":  projectID,
			"review_status":       ReviewApproved,
			"sanitization_status": SanitizationPassed,
		}).
		Where("quality_score IS NOT NULL AND quality_score >= ?", minimumQualityScore).
		Where(sq.Eq{"consent_status": []string{ConsentNotRequired, ConsentGranted}}).
		OrderBy("quality_score DESC", "created_at ASC")
	if len(languages) > 0 {
		builder = builder.Where(sq.Eq{"language": cleanStrings(languages)})
	}
	if len(taskTypes) > 0 {
		builder = builder.Where(sq.Eq{"task_type": cleanStrings(taskTypes)})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list approved examples: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query approved examples: %w", err)
	}
	defer rows.Close()
	return scanExampleRows(rows)
}

func (r *Repository) ListExamplesByIDs(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]TrainingExample, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := r.db.Builder.
		Select(exampleColumns).
		From("ai_training_examples").
		Where(sq.Eq{"dataset_project_id": projectID, "id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list examples by ids: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query examples by ids: %w", err)
	}
	defer rows.Close()
	return scanExampleRows(rows)
}

func (r *Repository) AssignExampleSplits(ctx context.Context, assignments []SplitAssignment) error {
	for _, assignment := range assignments {
		_, err := r.db.Exec(ctx, `
UPDATE ai_training_examples
SET duplicate_group_id = $2,
    split = $3,
    updated_at = NOW()
WHERE id = $1`, assignment.ExampleID, assignment.DuplicateGroupID, assignment.Split)
		if err != nil {
			return fmt.Errorf("assign example split: %w", err)
		}
	}
	return nil
}

func (r *Repository) CreateDatasetVersion(ctx context.Context, projectID uuid.UUID, version string, schemaVersion string, actorUserID *uuid.UUID) (*DatasetVersion, error) {
	if schemaVersion == "" {
		schemaVersion = SchemaVersion
	}
	query, args, err := r.db.Builder.
		Insert("ai_dataset_versions").
		Columns("id", "dataset_project_id", "version", "status", "schema_version", "created_by_user_id").
		Values(uuid.New(), projectID, strings.TrimSpace(version), VersionStatusBuilding, schemaVersion, uuidPtrArg(actorUserID)).
		Suffix("RETURNING " + versionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create dataset version: %w", err)
	}
	return scanVersion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkDatasetVersionReady(ctx context.Context, id uuid.UUID, manifest DatasetManifest, checksum string) (*DatasetVersion, error) {
	query, args, err := r.db.Builder.
		Update("ai_dataset_versions").
		Set("status", VersionStatusReady).
		Set("example_count", manifest.ExampleCount).
		Set("train_count", manifest.SplitCounts[SplitTrain]).
		Set("validation_count", manifest.SplitCounts[SplitValidation]).
		Set("test_count", manifest.SplitCounts[SplitTest]).
		Set("holdout_count", manifest.SplitCounts[SplitHoldout]).
		Set("manifest_json", mustJSON(manifest)).
		Set("checksum", checksum).
		Set("finalized_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING " + versionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark dataset version ready: %w", err)
	}
	return scanVersion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetDatasetVersion(ctx context.Context, id uuid.UUID) (*DatasetVersion, error) {
	query, args, err := r.db.Builder.
		Select(versionColumns).
		From("ai_dataset_versions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get dataset version: %w", err)
	}
	return scanVersion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkDatasetVersionExported(ctx context.Context, id uuid.UUID, exportPath, checksum string) (*DatasetVersion, error) {
	query, args, err := r.db.Builder.
		Update("ai_dataset_versions").
		Set("status", VersionStatusExported).
		Set("export_path", exportPath).
		Set("checksum", checksum).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING " + versionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark dataset version exported: %w", err)
	}
	return scanVersion(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) InsertReviewEvent(ctx context.Context, event ReviewEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx, `
INSERT INTO ai_dataset_review_events (
    id, training_example_id, dataset_version_id, actor_user_id, action, old_status, new_status, reason, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		event.ID,
		uuidPtrArg(event.TrainingExampleID),
		uuidPtrArg(event.DatasetVersionID),
		uuidPtrArg(event.ActorUserID),
		event.Action,
		textPtr(event.OldStatus),
		textPtr(event.NewStatus),
		textPtr(event.Reason),
		rawOrEmptyObject(event.Metadata),
	)
	if err != nil {
		return fmt.Errorf("insert dataset review event: %w", err)
	}
	return nil
}

type ReadinessData struct {
	ApprovedCount            int
	TaskDistribution         map[string]int
	LanguageDistribution     map[string]int
	ConsentCoverage          map[string]int
	SanitizationFailureCount int
	DuplicateCount           int
	HoldoutCount             int
}

func (r *Repository) ReadinessData(ctx context.Context) (ReadinessData, error) {
	data := ReadinessData{
		TaskDistribution:     map[string]int{},
		LanguageDistribution: map[string]int{},
		ConsentCoverage:      map[string]int{},
	}
	rows, err := r.db.Query(ctx, `
SELECT task_type, language, consent_status, split, duplicate_group_id
FROM ai_training_examples
WHERE review_status = 'approved'`)
	if err != nil {
		return data, fmt.Errorf("query readiness approved examples: %w", err)
	}
	defer rows.Close()
	duplicateGroups := map[uuid.UUID]int{}
	for rows.Next() {
		var task, language, consent string
		var split pgtype.Text
		var duplicate pgtype.UUID
		if err := rows.Scan(&task, &language, &consent, &split, &duplicate); err != nil {
			return data, fmt.Errorf("scan readiness approved example: %w", err)
		}
		data.ApprovedCount++
		data.TaskDistribution[task]++
		data.LanguageDistribution[language]++
		data.ConsentCoverage[consent]++
		if split.Valid && split.String == SplitHoldout {
			data.HoldoutCount++
		}
		if duplicate.Valid {
			id := uuid.UUID(duplicate.Bytes)
			duplicateGroups[id]++
		}
	}
	if err := rows.Err(); err != nil {
		return data, fmt.Errorf("iterate readiness examples: %w", err)
	}
	for _, count := range duplicateGroups {
		if count > 1 {
			data.DuplicateCount += count - 1
		}
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ai_training_examples WHERE sanitization_status = 'failed'`).Scan(&data.SanitizationFailureCount); err != nil {
		return data, fmt.Errorf("count sanitization failures: %w", err)
	}
	return data, nil
}

func scanProject(row pgx.Row) (*DatasetProject, error) {
	var id, createdBy pgtype.UUID
	var key, name, taskType, schemaVersion, status string
	var description pgtype.Text
	var minimum pgtype.Numeric
	var consentRequired bool
	var createdAt, updatedAt time.Time
	var archivedAt pgtype.Timestamp
	if err := row.Scan(&id, &key, &name, &description, &taskType, &schemaVersion, &status, &minimum, &consentRequired, &createdBy, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, scanErr("dataset project", err)
	}
	minimumFloat := numericToFloat(minimum)
	return &DatasetProject{
		ID:                  uuid.UUID(id.Bytes),
		Key:                 key,
		Name:                name,
		Description:         textFromPg(description),
		TaskType:            taskType,
		SchemaVersion:       schemaVersion,
		Status:              status,
		MinimumQualityScore: minimumFloat,
		ConsentRequired:     consentRequired,
		CreatedByUserID:     uuidFromPg(createdBy),
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
		ArchivedAt:          timeFromPg(archivedAt),
	}, nil
}

func scanProjectRows(rows pgx.Rows) ([]DatasetProject, error) {
	out := make([]DatasetProject, 0)
	for rows.Next() {
		item, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func scanConsent(row pgx.Row) (*TrainingConsent, error) {
	var id, userID pgtype.UUID
	var scopeType, consentVersion, status string
	var scopeID pgtype.Text
	var grantedAt, revokedAt pgtype.Timestamp
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &userID, &scopeType, &scopeID, &consentVersion, &status, &grantedAt, &revokedAt, &createdAt, &updatedAt); err != nil {
		return nil, scanErr("training consent", err)
	}
	return &TrainingConsent{
		ID:             uuid.UUID(id.Bytes),
		UserID:         uuid.UUID(userID.Bytes),
		ScopeType:      scopeType,
		ScopeID:        textFromPg(scopeID),
		ConsentVersion: consentVersion,
		Status:         status,
		GrantedAt:      timeFromPg(grantedAt),
		RevokedAt:      timeFromPg(revokedAt),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func scanExample(row pgx.Row) (*TrainingExample, error) {
	var id, projectID, userID, tripID, consentRecordID, duplicateGroupID, reviewedBy pgtype.UUID
	var sourceType, taskType, language, schemaVersion, consentStatus, sanitizationStatus, qualityStatus, reviewStatus, exportStatus string
	var sourceEntityType, sourceEntityID, split, rejectedReason pgtype.Text
	var inputRaw, groundingRaw, outputRaw, negativeRaw, labelsRaw, provenanceRaw []byte
	var qualityScore pgtype.Numeric
	var createdAt, updatedAt time.Time
	var reviewedAt pgtype.Timestamp
	if err := row.Scan(
		&id, &projectID, &sourceType, &sourceEntityType, &sourceEntityID, &userID, &tripID,
		&taskType, &language, &schemaVersion, &inputRaw, &groundingRaw, &outputRaw, &negativeRaw,
		&labelsRaw, &provenanceRaw, &consentStatus, &consentRecordID, &sanitizationStatus, &qualityStatus,
		&reviewStatus, &qualityScore, &duplicateGroupID, &split, &exportStatus, &createdAt, &updatedAt,
		&reviewedBy, &reviewedAt, &rejectedReason,
	); err != nil {
		return nil, scanErr("training example", err)
	}
	score := numericPtr(qualityScore)
	return &TrainingExample{
		ID:                 uuid.UUID(id.Bytes),
		DatasetProjectID:   uuid.UUID(projectID.Bytes),
		SourceType:         sourceType,
		SourceEntityType:   textFromPg(sourceEntityType),
		SourceEntityID:     textFromPg(sourceEntityID),
		UserID:             uuidFromPg(userID),
		TripID:             uuidFromPg(tripID),
		TaskType:           taskType,
		Language:           language,
		SchemaVersion:      schemaVersion,
		InputJSON:          inputRaw,
		GroundingJSON:      groundingRaw,
		ExpectedOutputJSON: outputRaw,
		NegativeOutputJSON: negativeRaw,
		LabelsJSON:         rawOrEmptyObject(labelsRaw),
		ProvenanceJSON:     rawOrEmptyObject(provenanceRaw),
		ConsentStatus:      consentStatus,
		ConsentRecordID:    uuidFromPg(consentRecordID),
		SanitizationStatus: sanitizationStatus,
		QualityStatus:      qualityStatus,
		ReviewStatus:       reviewStatus,
		QualityScore:       score,
		DuplicateGroupID:   uuidFromPg(duplicateGroupID),
		Split:              textFromPg(split),
		ExportStatus:       exportStatus,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		ReviewedByUserID:   uuidFromPg(reviewedBy),
		ReviewedAt:         timeFromPg(reviewedAt),
		RejectedReason:     textFromPg(rejectedReason),
	}, nil
}

func scanExampleRows(rows pgx.Rows) ([]TrainingExample, error) {
	out := make([]TrainingExample, 0)
	for rows.Next() {
		item, err := scanExample(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func scanVersion(row pgx.Row) (*DatasetVersion, error) {
	var id, projectID, createdBy pgtype.UUID
	var version, status, schemaVersion string
	var exampleCount, trainCount, validationCount, testCount, holdoutCount int
	var manifestRaw []byte
	var checksum, exportPath, invalidatedReason pgtype.Text
	var createdAt time.Time
	var finalizedAt, invalidatedAt pgtype.Timestamp
	if err := row.Scan(&id, &projectID, &version, &status, &schemaVersion, &exampleCount, &trainCount, &validationCount, &testCount, &holdoutCount, &manifestRaw, &checksum, &exportPath, &createdBy, &createdAt, &finalizedAt, &invalidatedAt, &invalidatedReason); err != nil {
		return nil, scanErr("dataset version", err)
	}
	return &DatasetVersion{
		ID:                uuid.UUID(id.Bytes),
		DatasetProjectID:  uuid.UUID(projectID.Bytes),
		Version:           version,
		Status:            status,
		SchemaVersion:     schemaVersion,
		ExampleCount:      exampleCount,
		TrainCount:        trainCount,
		ValidationCount:   validationCount,
		TestCount:         testCount,
		HoldoutCount:      holdoutCount,
		ManifestJSON:      rawOrEmptyObject(manifestRaw),
		Checksum:          textFromPg(checksum),
		ExportPath:        textFromPg(exportPath),
		CreatedByUserID:   uuidFromPg(createdBy),
		CreatedAt:         createdAt,
		FinalizedAt:       timeFromPg(finalizedAt),
		InvalidatedAt:     timeFromPg(invalidatedAt),
		InvalidatedReason: textFromPg(invalidatedReason),
	}, nil
}

func scanErr(label string, err error) error {
	if storage.NoRowsFound(err) {
		return domainerrs.ErrNotFound
	}
	return fmt.Errorf("scan %s: %w", label, err)
}

func uuidPtrArg(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return *id
}

func timePtrArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func textPtr(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func uuidFromPg(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func textFromPg(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	out := value.String
	return &out
}

func timeFromPg(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time
	return &out
}

func numericToFloat(value pgtype.Numeric) float64 {
	f, err := value.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

func numericPtr(value pgtype.Numeric) *float64 {
	if !value.Valid {
		return nil
	}
	out := numericToFloat(value)
	return &out
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mergeJSONObjects(base json.RawMessage, patch map[string]any) json.RawMessage {
	obj := jsonMap(base)
	for key, value := range patch {
		obj[key] = value
	}
	return mustJSON(obj)
}
