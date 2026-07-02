DROP INDEX IF EXISTS idx_trip_generation_jobs_correlation_id;

ALTER TABLE trip_generation_jobs
    DROP COLUMN IF EXISTS request_id,
    DROP COLUMN IF EXISTS correlation_id;

