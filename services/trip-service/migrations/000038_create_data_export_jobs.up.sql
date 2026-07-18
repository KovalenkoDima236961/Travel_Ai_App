CREATE TABLE IF NOT EXISTS data_export_jobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    export_type TEXT NOT NULL,
    status TEXT NOT NULL,
    scope_json JSONB NOT NULL DEFAULT '{}',
    file_path TEXT NULL,
    file_name TEXT NULL,
    mime_type TEXT NULL,
    size_bytes BIGINT NULL,
    checksum_sha256 TEXT NULL,
    error_code TEXT NULL,
    error_message_safe TEXT NULL,
    expires_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT data_export_jobs_type_check CHECK (export_type IN (
        'trip_archive', 'trip_expenses_csv', 'trip_budget_csv', 'trip_recap'
    )),
    CONSTRAINT data_export_jobs_status_check CHECK (status IN (
        'queued', 'running', 'completed', 'failed', 'expired', 'cancelled'
    )),
    CONSTRAINT data_export_jobs_size_check CHECK (size_bytes IS NULL OR size_bytes >= 0)
);

CREATE INDEX IF NOT EXISTS idx_data_export_jobs_user_created
    ON data_export_jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_export_jobs_status_created
    ON data_export_jobs (status, created_at);
CREATE INDEX IF NOT EXISTS idx_data_export_jobs_expires
    ON data_export_jobs (expires_at);
