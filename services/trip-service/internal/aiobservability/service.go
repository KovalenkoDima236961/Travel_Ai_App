package aiobservability

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiprivacy"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const traceColumns = "id, trip_id, job_id, user_id, workspace_id, request_id, correlation_id, generation_type, source, provider, model, ai_mode, prompt_version, planning_context_version, validator_version, status, quality_status, input_summary_json, constraints_summary_json, rag_summary_json, prompt_summary_json, generation_summary_json, validation_summary_json, repair_summary_json, output_summary_json, error_code, error_message_safe, duration_ms, queue_wait_ms, ai_call_duration_ms, validation_duration_ms, repair_duration_ms, token_prompt_estimate, token_completion_estimate, token_total_estimate, created_at, started_at, completed_at"

// Service deliberately owns only safe summaries. It never accepts a raw AI
// prompt into a trace record; optional snapshots are redacted before persistence.
type Service struct {
	db  *storage.DB
	cfg Config
	log *zap.Logger
}

func New(db *storage.DB, cfg Config, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{db: db, cfg: NormalizeConfig(cfg), log: log}
}

func (s *Service) Enabled() bool { return s != nil && s.cfg.Enabled && s.db != nil }

func (s *Service) StartTrace(ctx context.Context, in StartTraceInput) (*TraceContext, error) {
	if !s.Enabled() {
		return &TraceContext{Active: false}, nil
	}
	now := time.Now().UTC()
	provider := firstNonEmpty(in.Provider, s.cfg.Provider)
	mode := firstNonEmpty(in.AIMode, s.cfg.Mode)
	model := firstNonEmpty(in.Model, s.cfg.Model)
	trace := Trace{
		ID: uuid.New(), TripID: in.TripID, JobID: in.JobID, UserID: in.UserID, WorkspaceID: in.WorkspaceID,
		RequestID: in.RequestID, CorrelationID: in.CorrelationID,
		GenerationType: firstNonEmpty(in.GenerationType, "other"), Source: firstNonEmpty(in.Source, "other"),
		Provider: provider, AIMode: mode, Status: StatusStarted, InputSummary: safeJSON(in.InputSummary),
		ConstraintsSummary: safeJSON(in.ConstraintsSummary), RAGSummary: safeJSON(in.RAGSummary), PromptSummary: safeJSON(in.PromptSummary),
		CreatedAt: now, StartedAt: &now, QueueWaitMS: in.QueueWaitMS,
	}
	if model != "" {
		trace.Model = stringPtr(model)
	}
	if in.PromptVersion != "" {
		trace.PromptVersion = stringPtr(in.PromptVersion)
	}
	if in.PlanningContextVersion != "" {
		trace.PlanningContextVersion = stringPtr(in.PlanningContextVersion)
	}
	if in.ValidatorVersion != "" {
		trace.ValidatorVersion = stringPtr(in.ValidatorVersion)
	}

	err := s.write("start trace", func() error { return s.insertTrace(ctx, trace) })
	if err != nil {
		return nil, err
	}
	tracesStarted.WithLabelValues(trace.GenerationType, trace.Provider, stringValue(trace.Model)).Inc()
	traceCtx := &TraceContext{TraceID: trace.ID, GenerationType: trace.GenerationType, StartedAt: now, Active: true}
	if trace.CorrelationID != nil {
		traceCtx.CorrelationID = *trace.CorrelationID
	}
	if trace.RequestID != nil {
		traceCtx.RequestID = *trace.RequestID
	}
	_ = s.RecordEvent(ctx, traceCtx.TraceID, TraceEventInput{EventType: "trace_started", Status: "completed", Title: "Trace started"})
	return traceCtx, nil
}

func (s *Service) RecordEvent(ctx context.Context, traceID uuid.UUID, event TraceEventInput) error {
	if !s.Enabled() || traceID == uuid.Nil || !s.cfg.TraceEventsEnabled {
		return nil
	}
	status := firstNonEmpty(event.Status, "completed")
	message := safeText(event.Message, 1000)
	err := s.write("record trace event", func() error {
		_, err := s.db.Exec(ctx, `INSERT INTO ai_generation_trace_events (id, trace_id, event_type, event_status, title, message, metadata_json, duration_ms) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			idArg(uuid.New()), idArg(traceID), firstNonEmpty(event.EventType, "other"), status, safeText(event.Title, 200), nullableText(message), nullableJSON(safeJSON(event.Metadata)), nullableInt(event.DurationMS))
		return err
	})
	if err == nil {
		traceEvents.WithLabelValues(firstNonEmpty(event.EventType, "other"), status).Inc()
	}
	return err
}

func (s *Service) CompleteTrace(ctx context.Context, traceID uuid.UUID, in CompleteTraceInput) error {
	if !s.Enabled() || traceID == uuid.Nil {
		return nil
	}
	trace, err := s.Get(ctx, traceID)
	if err != nil {
		return s.write("load trace for completion", func() error { return err })
	}
	now := time.Now().UTC()
	duration := int(now.Sub(trace.StartedAtOrCreated()).Milliseconds())
	if duration < 0 {
		duration = 0
	}
	status := firstNonEmpty(in.Status, StatusCompleted)
	quality := strings.TrimSpace(in.QualityStatus)
	err = s.write("complete trace", func() error {
		_, err := s.db.Exec(ctx, `UPDATE ai_generation_traces SET status=$2, quality_status=$3, generation_summary_json=$4, validation_summary_json=$5, repair_summary_json=$6, output_summary_json=$7, duration_ms=$8, ai_call_duration_ms=$9, validation_duration_ms=$10, repair_duration_ms=$11, token_prompt_estimate=$12, token_completion_estimate=$13, token_total_estimate=$14, completed_at=$15 WHERE id=$1`,
			idArg(traceID), status, nullableText(quality), nullableJSON(safeJSON(in.GenerationSummary)), nullableJSON(safeJSON(in.ValidationSummary)), nullableJSON(safeJSON(in.RepairSummary)), nullableJSON(safeJSON(in.OutputSummary)), duration, nullableInt(in.AICallDurationMS), nullableInt(in.ValidationDurationMS), nullableInt(in.RepairDurationMS), nullableInt(in.TokenPromptEstimate), nullableInt(in.TokenCompletionEstimate), nullableInt(in.TokenTotalEstimate), now)
		return err
	})
	if err != nil {
		return err
	}
	model := stringValue(trace.Model)
	tracesCompleted.WithLabelValues(trace.GenerationType, trace.Provider, model, status, quality).Inc()
	generationDuration.WithLabelValues(trace.GenerationType, trace.Provider, model, status).Observe(float64(duration) / 1000)
	if in.AICallDurationMS != nil && *in.AICallDurationMS >= 0 {
		aiCallDuration.WithLabelValues(trace.GenerationType, trace.Provider, model).Observe(float64(*in.AICallDurationMS) / 1000)
	}
	if in.RepairDurationMS != nil && *in.RepairDurationMS >= 0 {
		repairDuration.WithLabelValues(trace.GenerationType, quality).Observe(float64(*in.RepairDurationMS) / 1000)
	}
	if in.TokenPromptEstimate != nil {
		promptTokens.WithLabelValues(trace.GenerationType, trace.Provider, model).Add(float64(*in.TokenPromptEstimate))
	}
	if in.TokenCompletionEstimate != nil {
		completionTokens.WithLabelValues(trace.GenerationType, trace.Provider, model).Add(float64(*in.TokenCompletionEstimate))
	}
	_ = s.RecordEvent(ctx, traceID, TraceEventInput{EventType: "trace_completed", Status: status, Title: "Trace completed", DurationMS: intPtr(duration)})
	return nil
}

func (s *Service) FailTrace(ctx context.Context, traceID uuid.UUID, in FailTraceInput) error {
	if !s.Enabled() || traceID == uuid.Nil {
		return nil
	}
	trace, err := s.Get(ctx, traceID)
	if err != nil {
		return s.write("load trace for failure", func() error { return err })
	}
	now := time.Now().UTC()
	duration := int(now.Sub(trace.StartedAtOrCreated()).Milliseconds())
	if duration < 0 {
		duration = 0
	}
	status := firstNonEmpty(in.Status, StatusFailed)
	code := safeText(in.ErrorCode, 120)
	// A provider error can echo a prompt or generated output. Persist only a
	// code-derived, operator-safe classification, never the upstream string.
	message := safeFailureMessage(code)
	err = s.write("fail trace", func() error {
		_, err := s.db.Exec(ctx, `UPDATE ai_generation_traces SET status=$2, quality_status=$3, error_code=$4, error_message_safe=$5, duration_ms=$6, completed_at=$7 WHERE id=$1`, idArg(traceID), status, nullableText(in.QualityStatus), nullableText(code), nullableText(message), duration, now)
		return err
	})
	if err != nil {
		return err
	}
	tracesFailed.WithLabelValues(trace.GenerationType, trace.Provider, stringValue(trace.Model), firstNonEmpty(code, "unknown")).Inc()
	generationDuration.WithLabelValues(trace.GenerationType, trace.Provider, stringValue(trace.Model), status).Observe(float64(duration) / 1000)
	_ = s.RecordEvent(ctx, traceID, TraceEventInput{EventType: "trace_failed", Status: status, Title: "Trace failed", Message: message, DurationMS: intPtr(duration)})
	return nil
}

func (s *Service) StorePromptSnapshot(ctx context.Context, traceID uuid.UUID, snapshotType, content string, tokenEstimate *int) error {
	if !s.Enabled() || traceID == uuid.Nil || !s.cfg.RedactionEnabled {
		return nil
	}
	if snapshotType != "redacted_prompt" && snapshotType != "redacted_ai_request" && snapshotType != "redacted_ai_response" {
		return fmt.Errorf("unsupported prompt snapshot type")
	}
	if (snapshotType == "redacted_prompt" || snapshotType == "redacted_ai_request") && !s.cfg.StoreRedactedPrompts {
		return nil
	}
	if snapshotType == "redacted_ai_response" && !s.cfg.StoreRedactedResponses {
		return nil
	}
	redacted := safeText(content, s.cfg.MaxPromptSnapshotChars)
	digest := sha256.Sum256([]byte(redacted))
	return s.write("store prompt snapshot", func() error {
		_, err := s.db.Exec(ctx, `INSERT INTO ai_prompt_snapshots (id, trace_id, snapshot_type, content_redacted, content_hash, token_estimate) VALUES ($1,$2,$3,$4,$5,$6)`, idArg(uuid.New()), idArg(traceID), snapshotType, redacted, hex.EncodeToString(digest[:]), nullableInt(tokenEstimate))
		return err
	})
}

func (s *Service) List(ctx context.Context, filters ListFilters) (ListResult, error) {
	if !s.Enabled() {
		return ListResult{Items: []Trace{}}, nil
	}
	filters = normalizeFilters(filters)
	where := make([]string, 0, 12)
	args := make([]any, 0, 16)
	arg := func(value any) string { args = append(args, value); return fmt.Sprintf("$%d", len(args)) }
	if filters.Status != "" {
		where = append(where, "status="+arg(filters.Status))
	}
	if filters.GenerationType != "" {
		where = append(where, "generation_type="+arg(filters.GenerationType))
	}
	if filters.Provider != "" {
		where = append(where, "provider="+arg(filters.Provider))
	}
	if filters.Model != "" {
		where = append(where, "model="+arg(filters.Model))
	}
	if filters.QualityStatus != "" {
		where = append(where, "quality_status="+arg(filters.QualityStatus))
	}
	if filters.ErrorOnly {
		where = append(where, "error_code IS NOT NULL")
	}
	for _, field := range []struct {
		name string
		id   *uuid.UUID
	}{{"trip_id", filters.TripID}, {"job_id", filters.JobID}, {"user_id", filters.UserID}, {"workspace_id", filters.WorkspaceID}} {
		if field.id != nil {
			where = append(where, field.name+"="+arg(idArg(*field.id)))
		}
	}
	if filters.From != nil {
		where = append(where, "created_at >= "+arg(*filters.From))
	}
	if filters.To != nil {
		where = append(where, "created_at <= "+arg(*filters.To))
	}
	if cursorTime, cursorID, ok := decodeCursor(filters.Cursor); ok {
		where = append(where, "(created_at, id) < ("+arg(cursorTime)+", "+arg(idArg(cursorID))+")")
	}
	query := "SELECT " + traceColumns + " FROM ai_generation_traces"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT " + arg(filters.Limit+1)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list ai generation traces: %w", err)
	}
	defer rows.Close()
	items, err := scanTraceRows(rows)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{Items: items}
	if len(items) > filters.Limit {
		result.Items = items[:filters.Limit]
		cursor := encodeCursor(result.Items[len(result.Items)-1])
		result.NextCursor = &cursor
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, traceID uuid.UUID) (*Trace, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("ai observability is disabled")
	}
	trace, err := scanTrace(s.db.QueryRow(ctx, "SELECT "+traceColumns+" FROM ai_generation_traces WHERE id=$1", idArg(traceID)))
	if err != nil {
		return nil, err
	}
	return &trace, nil
}

func (s *Service) Detail(ctx context.Context, traceID uuid.UUID, includeSnapshot bool) (*Detail, error) {
	trace, err := s.Get(ctx, traceID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT id, trace_id, event_type, event_status, title, message, metadata_json, duration_ms, created_at FROM ai_generation_trace_events WHERE trace_id=$1 ORDER BY created_at ASC, id ASC`, idArg(traceID))
	if err != nil {
		return nil, fmt.Errorf("list trace events: %w", err)
	}
	defer rows.Close()
	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	detail := &Detail{Trace: *trace, Events: events}
	if includeSnapshot && s.cfg.StoreRedactedPrompts {
		snapshot, err := scanSnapshot(s.db.QueryRow(ctx, `SELECT id, trace_id, snapshot_type, content_redacted, content_hash, token_estimate, created_at FROM ai_prompt_snapshots WHERE trace_id=$1 AND snapshot_type='redacted_prompt' ORDER BY created_at DESC LIMIT 1`, idArg(traceID)))
		if err == nil {
			detail.PromptSnapshot = &snapshot
		} else if err != pgx.ErrNoRows {
			return nil, err
		}
	}
	return detail, nil
}

func (s *Service) AuditAccess(ctx context.Context, actorID uuid.UUID, actorEmail, action string, traceID uuid.UUID) {
	if !s.Enabled() {
		return
	}
	_ = s.write("audit ai trace access", func() error {
		_, err := s.db.Exec(ctx, `INSERT INTO ops_audit_events (id, actor_user_id, actor_email, action, entity_type, entity_id, reason, metadata_json) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, idArg(uuid.New()), idArg(actorID), strings.ToLower(strings.TrimSpace(actorEmail)), action, "ai_generation_trace", idArg(traceID), "ops trace access", []byte(`{"safe":true}`))
		return err
	})
}

func (s *Service) Cleanup(ctx context.Context) (int64, error) {
	if !s.Enabled() {
		return 0, nil
	}
	command, err := s.db.Exec(ctx, `DELETE FROM ai_generation_traces WHERE created_at < NOW() - ($1 * INTERVAL '1 day')`, s.cfg.RetentionDays)
	if err != nil {
		return 0, fmt.Errorf("cleanup ai generation traces: %w", err)
	}
	return command.RowsAffected(), nil
}

func (s *Service) insertTrace(ctx context.Context, trace Trace) error {
	_, err := s.db.Exec(ctx, `INSERT INTO ai_generation_traces (`+traceColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39)`,
		idArg(trace.ID), nullableUUID(trace.TripID), nullableUUID(trace.JobID), nullableUUID(trace.UserID), nullableUUID(trace.WorkspaceID), nullableTextPtr(trace.RequestID), nullableTextPtr(trace.CorrelationID), trace.GenerationType, trace.Source, trace.Provider, nullableTextPtr(trace.Model), trace.AIMode, nullableTextPtr(trace.PromptVersion), nullableTextPtr(trace.PlanningContextVersion), nullableTextPtr(trace.ValidatorVersion), trace.Status, nullableTextPtr(trace.QualityStatus), nullableJSON(trace.InputSummary), nullableJSON(trace.ConstraintsSummary), nullableJSON(trace.RAGSummary), nullableJSON(trace.PromptSummary), nullableJSON(trace.GenerationSummary), nullableJSON(trace.ValidationSummary), nullableJSON(trace.RepairSummary), nullableJSON(trace.OutputSummary), nullableTextPtr(trace.ErrorCode), nullableTextPtr(trace.ErrorMessageSafe), nullableInt(trace.DurationMS), nullableInt(trace.QueueWaitMS), nullableInt(trace.AICallDurationMS), nullableInt(trace.ValidationDurationMS), nullableInt(trace.RepairDurationMS), nullableInt(trace.TokenPromptEstimate), nullableInt(trace.TokenCompletionEstimate), nullableInt(trace.TokenTotalEstimate), trace.CreatedAt, nullableTime(trace.StartedAt), nullableTime(trace.CompletedAt))
	return err
}

func (s *Service) write(operation string, fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}
	traceWriteFailures.Inc()
	s.log.Warn("ai observability write failed", zap.String("operation", operation), zap.Error(err))
	if s.cfg.FailOpen {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func (t Trace) StartedAtOrCreated() time.Time {
	if t.StartedAt != nil {
		return *t.StartedAt
	}
	return t.CreatedAt
}

func safeJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	clean, _, err := aiprivacy.SanitizeJSON(raw)
	if err != nil {
		return nil
	}
	return clean
}

func safeText(value string, max int) string {
	value, _ = aiprivacy.RedactText(strings.TrimSpace(value))
	if max > 0 && len(value) > max {
		return value[:max] + "…[truncated]"
	}
	return value
}

func safeFailureMessage(code string) string {
	switch strings.TrimSpace(code) {
	case "validation_failed", "ai_repair_failed", "proposal_build_failed":
		return "AI output did not satisfy validation requirements."
	case "ai_generation_failed", "ai_adaptation_failed", "enrichment_failed":
		return "The AI provider could not complete this request."
	case "provider_rate_limited":
		return "The AI provider is temporarily rate limited."
	case "provider_quota_exceeded":
		return "The AI provider quota has been reached."
	case "provider_limits_unavailable":
		return "The provider limit service is unavailable."
	case "itinerary_conflict":
		return "The itinerary changed while this generation was running."
	case "permission_denied", "trip_not_found":
		return "The requested trip is no longer available."
	case "cancelled", "ops_cancelled":
		return "Generation was cancelled."
	case "worker_restarted", "worker_interrupted":
		return "Generation was interrupted by a worker restart."
	default:
		return "Generation did not complete; use the trace ID and error code for investigation."
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func stringPtr(value string) *string { return &value }
func intPtr(value int) *int          { return &value }
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func nullableTextPtr(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}
func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return []byte(value)
}
func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return idArg(*value)
}
func idArg(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: [16]byte(id), Valid: true} }
func normalizeFilters(filters ListFilters) ListFilters {
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	if filters.Limit > 200 {
		filters.Limit = 200
	}
	return filters
}
func encodeCursor(trace Trace) string {
	return base64.RawURLEncoding.EncodeToString([]byte(trace.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + trace.ID.String()))
}
func decodeCursor(value string) (time.Time, uuid.UUID, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, uuid.Nil, false
	}
	pieces := strings.Split(string(raw), "|")
	if len(pieces) != 2 {
		return time.Time{}, uuid.Nil, false
	}
	created, err := time.Parse(time.RFC3339Nano, pieces[0])
	if err != nil {
		return time.Time{}, uuid.Nil, false
	}
	id, err := uuid.Parse(pieces[1])
	return created, id, err == nil
}
