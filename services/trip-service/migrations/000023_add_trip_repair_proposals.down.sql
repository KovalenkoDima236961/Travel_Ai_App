DROP INDEX IF EXISTS idx_trip_repair_proposals_trip_status;
DROP INDEX IF EXISTS idx_trip_repair_proposals_created_at;
DROP INDEX IF EXISTS idx_trip_repair_proposals_status;
DROP INDEX IF EXISTS idx_trip_repair_proposals_job_id;
DROP INDEX IF EXISTS idx_trip_repair_proposals_trip_id;

DROP TABLE IF EXISTS trip_repair_proposals;

ALTER TABLE trip_generation_jobs
    DROP CONSTRAINT IF EXISTS trip_generation_jobs_job_type_check;

ALTER TABLE trip_generation_jobs
    ADD CONSTRAINT trip_generation_jobs_job_type_check CHECK (
        job_type IN (
            'full_generation',
            'day_regeneration',
            'item_regeneration',
            'quality_improvement_day',
            'quality_improvement_item',
            'budget_optimization_day',
            'template_adaptation'
        )
    );
