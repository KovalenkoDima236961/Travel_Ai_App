-- Template adaptation reuses the existing trip_generation_jobs table. A draft
-- trip is created up front (like full_generation), the AI adaptation request is
-- stored in the existing `payload` column, and the adaptation summary is stored
-- in a new `result_payload` column on completion.

ALTER TABLE trip_generation_jobs
    ADD COLUMN IF NOT EXISTS result_payload JSONB NULL;

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
