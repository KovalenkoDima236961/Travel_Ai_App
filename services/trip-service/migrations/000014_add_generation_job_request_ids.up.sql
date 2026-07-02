ALTER TABLE trip_generation_jobs
    ADD COLUMN IF NOT EXISTS correlation_id TEXT NULL,
    ADD COLUMN IF NOT EXISTS request_id TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_trip_generation_jobs_correlation_id
    ON trip_generation_jobs(correlation_id)
    WHERE correlation_id IS NOT NULL;

