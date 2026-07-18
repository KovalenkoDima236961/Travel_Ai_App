CREATE TABLE IF NOT EXISTS data_export_jobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    export_type TEXT NOT NULL CHECK (export_type = 'account'),
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed', 'expired', 'cancelled')),
    scope_json JSONB NOT NULL DEFAULT '{}',
    file_path TEXT NULL,
    file_name TEXT NULL,
    mime_type TEXT NULL,
    size_bytes BIGINT NULL CHECK (size_bytes IS NULL OR size_bytes >= 0),
    checksum_sha256 TEXT NULL,
    error_code TEXT NULL,
    error_message_safe TEXT NULL,
    expires_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_data_export_jobs_user_created ON data_export_jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_export_jobs_status_created ON data_export_jobs (status, created_at);
CREATE INDEX IF NOT EXISTS idx_data_export_jobs_expires ON data_export_jobs (expires_at);

CREATE TABLE IF NOT EXISTS account_cleanup_requests (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'requested' CHECK (status IN ('requested', 'reviewed', 'cancelled', 'completed')),
    reason TEXT NULL,
    export_requested_first BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_account_cleanup_requests_user_created ON account_cleanup_requests (user_id, created_at DESC);
