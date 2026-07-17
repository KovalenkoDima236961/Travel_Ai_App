CREATE TABLE IF NOT EXISTS ai_generation_traces (
    id UUID PRIMARY KEY,
    trip_id UUID NULL,
    job_id UUID NULL,
    user_id UUID NULL,
    workspace_id UUID NULL,
    request_id TEXT NULL,
    correlation_id TEXT NULL,
    generation_type TEXT NOT NULL,
    source TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NULL,
    ai_mode TEXT NOT NULL,
    prompt_version TEXT NULL,
    planning_context_version TEXT NULL,
    validator_version TEXT NULL,
    status TEXT NOT NULL,
    quality_status TEXT NULL,
    input_summary_json JSONB NULL,
    constraints_summary_json JSONB NULL,
    rag_summary_json JSONB NULL,
    prompt_summary_json JSONB NULL,
    generation_summary_json JSONB NULL,
    validation_summary_json JSONB NULL,
    repair_summary_json JSONB NULL,
    output_summary_json JSONB NULL,
    error_code TEXT NULL,
    error_message_safe TEXT NULL,
    duration_ms INTEGER NULL,
    queue_wait_ms INTEGER NULL,
    ai_call_duration_ms INTEGER NULL,
    validation_duration_ms INTEGER NULL,
    repair_duration_ms INTEGER NULL,
    token_prompt_estimate INTEGER NULL,
    token_completion_estimate INTEGER NULL,
    token_total_estimate INTEGER NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    CONSTRAINT ai_generation_traces_status_check CHECK (status IN ('started', 'completed', 'completed_with_warnings', 'failed', 'cancelled', 'blocked'))
);

CREATE TABLE IF NOT EXISTS ai_generation_trace_events (
    id UUID PRIMARY KEY,
    trace_id UUID NOT NULL REFERENCES ai_generation_traces(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    event_status TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NULL,
    metadata_json JSONB NULL,
    duration_ms INTEGER NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_prompt_snapshots (
    id UUID PRIMARY KEY,
    trace_id UUID NOT NULL REFERENCES ai_generation_traces(id) ON DELETE CASCADE,
    snapshot_type TEXT NOT NULL,
    content_redacted TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    token_estimate INTEGER NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_prompt_snapshots_type_check CHECK (snapshot_type IN ('redacted_prompt', 'redacted_ai_request', 'redacted_ai_response'))
);

CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_trip_id ON ai_generation_traces(trip_id);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_job_id ON ai_generation_traces(job_id);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_user_id ON ai_generation_traces(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_workspace_id ON ai_generation_traces(workspace_id);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_generation_type ON ai_generation_traces(generation_type);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_status ON ai_generation_traces(status);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_quality_status ON ai_generation_traces(quality_status);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_created_at ON ai_generation_traces(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_correlation_id ON ai_generation_traces(correlation_id);
CREATE INDEX IF NOT EXISTS idx_ai_generation_trace_events_trace_id_created_at ON ai_generation_trace_events(trace_id, created_at);
CREATE INDEX IF NOT EXISTS idx_ai_prompt_snapshots_trace_id ON ai_prompt_snapshots(trace_id);
