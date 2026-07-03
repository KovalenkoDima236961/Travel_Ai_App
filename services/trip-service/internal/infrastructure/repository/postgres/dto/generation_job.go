package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

const GenerationJobColumns = "id, trip_id, requested_by_user_id, job_type, status, " +
	"expected_itinerary_revision, instruction, day_number, item_index, payload, correlation_id, request_id, retried_from_job_id, error_code, " +
	"error_message, result_itinerary_revision, created_at, started_at, completed_at, " +
	"cancelled_at, updated_at"

func GenerationJobInsertColumns() []string {
	return []string{
		"id",
		"trip_id",
		"requested_by_user_id",
		"job_type",
		"status",
		"expected_itinerary_revision",
		"instruction",
		"day_number",
		"item_index",
		"payload",
		"correlation_id",
		"request_id",
		"retried_from_job_id",
	}
}

func GenerationJobInsertValues(job *entity.GenerationJob) []any {
	return []any{
		toPgUUID(job.ID),
		toPgUUID(job.TripID),
		toPgUUID(job.RequestedByUserID),
		string(job.JobType),
		string(job.Status),
		job.ExpectedItineraryRevision,
		toPgTextPtr(job.Instruction),
		toPgIntPtr(job.DayNumber),
		toPgIntPtr(job.ItemIndex),
		rawJSONArg(job.Payload),
		toPgTextPtr(job.CorrelationID),
		toPgTextPtr(job.RequestID),
		toPgUUIDPtr(job.RetriedFromJobID),
	}
}

func ScanGenerationJob(row pgx.Row) (*entity.GenerationJob, error) {
	var (
		id, tripID, requestedByUserID pgtype.UUID
		jobType, status               string
		expectedRevision              int
		instruction                   pgtype.Text
		dayNumber                     pgtype.Int4
		itemIndex                     pgtype.Int4
		payloadRaw                    []byte
		correlationID                 pgtype.Text
		requestID                     pgtype.Text
		retriedFromJobID              pgtype.UUID
		errorCode                     pgtype.Text
		errorMessage                  pgtype.Text
		resultRevision                pgtype.Int4
		createdAt                     pgtype.Timestamp
		startedAt                     pgtype.Timestamp
		completedAt                   pgtype.Timestamp
		cancelledAt                   pgtype.Timestamp
		updatedAt                     pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&requestedByUserID,
		&jobType,
		&status,
		&expectedRevision,
		&instruction,
		&dayNumber,
		&itemIndex,
		&payloadRaw,
		&correlationID,
		&requestID,
		&retriedFromJobID,
		&errorCode,
		&errorMessage,
		&resultRevision,
		&createdAt,
		&startedAt,
		&completedAt,
		&cancelledAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan generation job: %w", err)
	}

	return &entity.GenerationJob{
		ID:                        uuid.UUID(id.Bytes),
		TripID:                    uuid.UUID(tripID.Bytes),
		RequestedByUserID:         uuid.UUID(requestedByUserID.Bytes),
		JobType:                   entity.GenerationJobType(jobType),
		Status:                    entity.GenerationJobStatus(status),
		ExpectedItineraryRevision: expectedRevision,
		Instruction:               fromPgText(instruction),
		DayNumber:                 fromPgIntPtr(dayNumber),
		ItemIndex:                 fromPgIntPtr(itemIndex),
		Payload:                   payloadRaw,
		CorrelationID:             fromPgText(correlationID),
		RequestID:                 fromPgText(requestID),
		RetriedFromJobID:          fromPgUUID(retriedFromJobID),
		ErrorCode:                 fromPgText(errorCode),
		ErrorMessage:              fromPgText(errorMessage),
		ResultItineraryRevision:   fromPgIntPtr(resultRevision),
		CreatedAt:                 createdAt.Time,
		StartedAt:                 fromPgTimestampPtr(startedAt),
		CompletedAt:               fromPgTimestampPtr(completedAt),
		CancelledAt:               fromPgTimestampPtr(cancelledAt),
		UpdatedAt:                 updatedAt.Time,
	}, nil
}

func rawJSONArg(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func ScanGenerationJobRows(rows pgx.Rows) ([]entity.GenerationJob, error) {
	jobs := make([]entity.GenerationJob, 0)
	for rows.Next() {
		job, err := ScanGenerationJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate generation jobs: %w", err)
	}
	return jobs, nil
}

func toPgIntPtr(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func fromPgIntPtr(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int32)
	return &v
}

func fromPgTimestampPtr(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}
