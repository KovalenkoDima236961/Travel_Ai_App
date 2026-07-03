DROP INDEX IF EXISTS idx_ops_audit_events_actor;
DROP INDEX IF EXISTS idx_ops_audit_events_entity;
DROP TABLE IF EXISTS ops_audit_events;

DROP INDEX IF EXISTS idx_trip_generation_jobs_retried_from_job_id;
ALTER TABLE trip_generation_jobs
    DROP COLUMN IF EXISTS retried_from_job_id;
