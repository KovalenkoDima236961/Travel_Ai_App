package aiobservability

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

func scanTrace(row pgx.Row) (Trace, error) {
	var (
		id, tripID, jobID, userID, workspaceID                                                                                          pgtype.UUID
		requestID, correlationID, model, promptVersion, contextVersion, validatorVersion                                                pgtype.Text
		generationType, source, provider, mode, status                                                                                  string
		qualityStatus, errorCode, errorMessage                                                                                          pgtype.Text
		inputSummary, constraintsSummary, ragSummary, promptSummary, generationSummary, validationSummary, repairSummary, outputSummary []byte
		duration, queueWait, aiCallDuration, validationDuration, repairDuration, promptTokens, completionTokens, totalTokens            pgtype.Int4
		createdAt, startedAt, completedAt                                                                                               pgtype.Timestamp
	)
	err := row.Scan(&id, &tripID, &jobID, &userID, &workspaceID, &requestID, &correlationID, &generationType, &source, &provider, &model, &mode, &promptVersion, &contextVersion, &validatorVersion, &status, &qualityStatus, &inputSummary, &constraintsSummary, &ragSummary, &promptSummary, &generationSummary, &validationSummary, &repairSummary, &outputSummary, &errorCode, &errorMessage, &duration, &queueWait, &aiCallDuration, &validationDuration, &repairDuration, &promptTokens, &completionTokens, &totalTokens, &createdAt, &startedAt, &completedAt)
	if err != nil {
		if storage.NoRowsFound(err) {
			return Trace{}, domainerrs.ErrNotFound
		}
		return Trace{}, fmt.Errorf("scan ai generation trace: %w", err)
	}
	return Trace{
		ID: uuid.UUID(id.Bytes), TripID: uuidPtr(tripID), JobID: uuidPtr(jobID), UserID: uuidPtr(userID), WorkspaceID: uuidPtr(workspaceID),
		RequestID: textPtr(requestID), CorrelationID: textPtr(correlationID), GenerationType: generationType, Source: source, Provider: provider, Model: textPtr(model), AIMode: mode,
		PromptVersion: textPtr(promptVersion), PlanningContextVersion: textPtr(contextVersion), ValidatorVersion: textPtr(validatorVersion), Status: status, QualityStatus: textPtr(qualityStatus),
		InputSummary: inputSummary, ConstraintsSummary: constraintsSummary, RAGSummary: ragSummary, PromptSummary: promptSummary, GenerationSummary: generationSummary, ValidationSummary: validationSummary, RepairSummary: repairSummary, OutputSummary: outputSummary,
		ErrorCode: textPtr(errorCode), ErrorMessageSafe: textPtr(errorMessage), DurationMS: intPtrFromPG(duration), QueueWaitMS: intPtrFromPG(queueWait), AICallDurationMS: intPtrFromPG(aiCallDuration), ValidationDurationMS: intPtrFromPG(validationDuration), RepairDurationMS: intPtrFromPG(repairDuration), TokenPromptEstimate: intPtrFromPG(promptTokens), TokenCompletionEstimate: intPtrFromPG(completionTokens), TokenTotalEstimate: intPtrFromPG(totalTokens),
		CreatedAt: createdAt.Time, StartedAt: timePtr(startedAt), CompletedAt: timePtr(completedAt),
	}, nil
}

func scanTraceRows(rows pgx.Rows) ([]Trace, error) {
	items := make([]Trace, 0)
	for rows.Next() {
		trace, err := scanTrace(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, trace)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai generation traces: %w", err)
	}
	return items, nil
}

func scanEvents(rows pgx.Rows) ([]TraceEvent, error) {
	items := make([]TraceEvent, 0)
	for rows.Next() {
		var id, traceID pgtype.UUID
		var eventType, status, title string
		var message pgtype.Text
		var metadata []byte
		var duration pgtype.Int4
		var createdAt pgtype.Timestamp
		if err := rows.Scan(&id, &traceID, &eventType, &status, &title, &message, &metadata, &duration, &createdAt); err != nil {
			return nil, fmt.Errorf("scan trace event: %w", err)
		}
		items = append(items, TraceEvent{ID: uuid.UUID(id.Bytes), TraceID: uuid.UUID(traceID.Bytes), EventType: eventType, Status: status, Title: title, Message: textPtr(message), Metadata: metadata, DurationMS: intPtrFromPG(duration), CreatedAt: createdAt.Time})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trace events: %w", err)
	}
	return items, nil
}

func scanSnapshot(row pgx.Row) (PromptSnapshot, error) {
	var id, traceID pgtype.UUID
	var snapshotType, content, hash string
	var tokenEstimate pgtype.Int4
	var createdAt pgtype.Timestamp
	if err := row.Scan(&id, &traceID, &snapshotType, &content, &hash, &tokenEstimate, &createdAt); err != nil {
		return PromptSnapshot{}, err
	}
	return PromptSnapshot{ID: uuid.UUID(id.Bytes), TraceID: uuid.UUID(traceID.Bytes), SnapshotType: snapshotType, ContentRedacted: content, ContentHash: hash, TokenEstimate: intPtrFromPG(tokenEstimate), CreatedAt: createdAt.Time}, nil
}

func uuidPtr(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}
func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	out := value.String
	return &out
}
func intPtrFromPG(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	out := int(value.Int32)
	return &out
}
func timePtr(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time
	return &out
}
