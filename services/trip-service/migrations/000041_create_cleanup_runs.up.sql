-- Central, non-sensitive execution history for Worker Service cleanup tasks.
-- Data remains owned and deleted by its service; this table records only
-- aggregate outcomes so operators can verify lifecycle work safely.
CREATE TABLE IF NOT EXISTS cleanup_runs (
    id UUID PRIMARY KEY,
    task_name TEXT NOT NULL,
    status TEXT NOT NULL,
    dry_run BOOLEAN NOT NULL,
    started_by TEXT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NULL,
    scanned_count BIGINT NOT NULL DEFAULT 0,
    deleted_count BIGINT NOT NULL DEFAULT 0,
    archived_count BIGINT NOT NULL DEFAULT 0,
    skipped_count BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    file_deleted_count BIGINT NOT NULL DEFAULT 0,
    bytes_freed BIGINT NOT NULL DEFAULT 0,
    warnings JSONB NULL,
    error_message TEXT NULL,
    request_id TEXT NULL,
    lock_expires_at TIMESTAMPTZ NULL,
    CONSTRAINT cleanup_runs_status_check CHECK (status IN ('running', 'succeeded', 'failed')),
    CONSTRAINT cleanup_runs_counts_check CHECK (
        scanned_count >= 0 AND deleted_count >= 0 AND archived_count >= 0 AND
        skipped_count >= 0 AND error_count >= 0 AND file_deleted_count >= 0 AND bytes_freed >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_cleanup_runs_task_started_at
    ON cleanup_runs (task_name, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_cleanup_runs_status_started_at
    ON cleanup_runs (status, started_at DESC);
-- A running row is both the distributed lock and the observable lock holder.
CREATE UNIQUE INDEX IF NOT EXISTS idx_cleanup_runs_one_running_task
    ON cleanup_runs (task_name) WHERE status = 'running';
