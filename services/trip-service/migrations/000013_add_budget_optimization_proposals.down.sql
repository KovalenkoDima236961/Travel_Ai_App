DROP INDEX IF EXISTS idx_budget_optimization_proposals_trip_status;
DROP INDEX IF EXISTS idx_budget_optimization_proposals_trip_created_at;
DROP INDEX IF EXISTS idx_budget_optimization_proposals_status;
DROP INDEX IF EXISTS idx_budget_optimization_proposals_created_by_user_id;
DROP INDEX IF EXISTS idx_budget_optimization_proposals_job_id;
DROP INDEX IF EXISTS idx_budget_optimization_proposals_trip_id;

DROP TABLE IF EXISTS budget_optimization_proposals;

ALTER TABLE trip_generation_jobs
    DROP CONSTRAINT IF EXISTS trip_generation_jobs_day_number_check;

ALTER TABLE trip_generation_jobs
    ADD CONSTRAINT trip_generation_jobs_day_number_check CHECK (
        (
            job_type IN ('day_regeneration', 'quality_improvement_day')
            AND day_number IS NOT NULL
            AND day_number > 0
        )
        OR job_type NOT IN ('day_regeneration', 'quality_improvement_day')
    );

ALTER TABLE trip_generation_jobs
    DROP CONSTRAINT IF EXISTS trip_generation_jobs_job_type_check;

ALTER TABLE trip_generation_jobs
    ADD CONSTRAINT trip_generation_jobs_job_type_check CHECK (
        job_type IN (
            'full_generation',
            'day_regeneration',
            'item_regeneration',
            'quality_improvement_day',
            'quality_improvement_item'
        )
    );

ALTER TABLE trip_generation_jobs
    DROP COLUMN IF EXISTS payload;
