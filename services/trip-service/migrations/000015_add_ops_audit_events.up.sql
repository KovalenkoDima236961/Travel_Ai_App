ALTER TABLE trip_generation_jobs
    ADD COLUMN IF NOT EXISTS retried_from_job_id UUID NULL REFERENCES trip_generation_jobs(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_trip_generation_jobs_retried_from_job_id
    ON trip_generation_jobs(retried_from_job_id)
    WHERE retried_from_job_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS ops_audit_events (
    id UUID PRIMARY KEY,
    actor_user_id UUID NOT NULL,
    actor_email TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    reason TEXT NOT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT ops_audit_events_action_check CHECK (
        action IN (
            'ops_job_retried',
            'ops_job_cancelled',
            'ops_job_marked_failed'
        )
    ),
    CONSTRAINT ops_audit_events_reason_length_check CHECK (
        length(reason) BETWEEN 1 AND 500
    )
);

CREATE INDEX IF NOT EXISTS idx_ops_audit_events_entity
    ON ops_audit_events(entity_type, entity_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_audit_events_actor
    ON ops_audit_events(actor_user_id, created_at DESC);
