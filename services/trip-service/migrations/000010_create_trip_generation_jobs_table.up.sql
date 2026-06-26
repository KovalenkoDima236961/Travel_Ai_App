CREATE TABLE IF NOT EXISTS trip_generation_jobs (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    requested_by_user_id UUID NOT NULL,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL,
    expected_itinerary_revision INT NOT NULL,
    instruction TEXT NULL,
    day_number INT NULL,
    item_index INT NULL,
    error_code TEXT NULL,
    error_message TEXT NULL,
    result_itinerary_revision INT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    cancelled_at TIMESTAMP NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT trip_generation_jobs_job_type_check CHECK (
        job_type IN (
            'full_generation',
            'day_regeneration',
            'item_regeneration',
            'quality_improvement_day',
            'quality_improvement_item'
        )
    ),
    CONSTRAINT trip_generation_jobs_status_check CHECK (
        status IN ('queued', 'running', 'completed', 'failed', 'cancelled')
    ),
    CONSTRAINT trip_generation_jobs_expected_revision_check CHECK (
        expected_itinerary_revision >= 0
    ),
    CONSTRAINT trip_generation_jobs_day_number_check CHECK (
        (
            job_type IN ('day_regeneration', 'quality_improvement_day')
            AND day_number IS NOT NULL
            AND day_number > 0
        )
        OR job_type NOT IN ('day_regeneration', 'quality_improvement_day')
    ),
    CONSTRAINT trip_generation_jobs_item_target_check CHECK (
        (
            job_type IN ('item_regeneration', 'quality_improvement_item')
            AND day_number IS NOT NULL
            AND day_number > 0
            AND item_index IS NOT NULL
            AND item_index >= 0
        )
        OR job_type NOT IN ('item_regeneration', 'quality_improvement_item')
    ),
    CONSTRAINT trip_generation_jobs_instruction_length_check CHECK (
        instruction IS NULL OR length(instruction) <= 2000
    ),
    CONSTRAINT trip_generation_jobs_error_message_length_check CHECK (
        error_message IS NULL OR length(error_message) <= 2000
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_generation_jobs_trip_id
    ON trip_generation_jobs(trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_generation_jobs_requested_by_user_id
    ON trip_generation_jobs(requested_by_user_id);

CREATE INDEX IF NOT EXISTS idx_trip_generation_jobs_status
    ON trip_generation_jobs(status);

CREATE INDEX IF NOT EXISTS idx_trip_generation_jobs_status_created_at
    ON trip_generation_jobs(status, created_at);

CREATE INDEX IF NOT EXISTS idx_trip_generation_jobs_trip_created_at
    ON trip_generation_jobs(trip_id, created_at DESC);
