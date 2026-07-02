ALTER TABLE trip_generation_jobs
    ADD COLUMN IF NOT EXISTS payload JSONB NULL;

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
            'budget_optimization_day'
        )
    );

ALTER TABLE trip_generation_jobs
    DROP CONSTRAINT IF EXISTS trip_generation_jobs_day_number_check;

ALTER TABLE trip_generation_jobs
    ADD CONSTRAINT trip_generation_jobs_day_number_check CHECK (
        (
            job_type IN ('day_regeneration', 'quality_improvement_day', 'budget_optimization_day')
            AND day_number IS NOT NULL
            AND day_number > 0
        )
        OR job_type NOT IN ('day_regeneration', 'quality_improvement_day', 'budget_optimization_day')
    );

CREATE TABLE IF NOT EXISTS budget_optimization_proposals (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    job_id UUID NULL REFERENCES trip_generation_jobs(id) ON DELETE SET NULL,
    created_by_user_id UUID NOT NULL,
    scope TEXT NOT NULL,
    day_number INT NULL,
    expected_itinerary_revision INT NOT NULL,
    base_itinerary_revision INT NOT NULL,
    status TEXT NOT NULL,
    currency TEXT NOT NULL,
    target_reduction_amount NUMERIC(12,2) NULL,
    estimated_savings_amount NUMERIC(12,2) NULL,
    proposal_json JSONB NOT NULL,
    applied_itinerary_revision INT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMP NULL,
    discarded_at TIMESTAMP NULL,
    expired_at TIMESTAMP NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT budget_optimization_proposals_scope_check CHECK (scope IN ('day')),
    CONSTRAINT budget_optimization_proposals_status_check CHECK (
        status IN ('pending', 'applied', 'discarded', 'expired', 'failed')
    ),
    CONSTRAINT budget_optimization_proposals_day_number_check CHECK (
        scope <> 'day' OR (day_number IS NOT NULL AND day_number > 0)
    ),
    CONSTRAINT budget_optimization_proposals_expected_revision_check CHECK (
        expected_itinerary_revision >= 0
    ),
    CONSTRAINT budget_optimization_proposals_base_revision_check CHECK (
        base_itinerary_revision >= 0
    ),
    CONSTRAINT budget_optimization_proposals_currency_check CHECK (
        currency ~ '^[A-Z]{3}$'
    ),
    CONSTRAINT budget_optimization_proposals_target_reduction_check CHECK (
        target_reduction_amount IS NULL OR target_reduction_amount >= 0
    ),
    CONSTRAINT budget_optimization_proposals_estimated_savings_check CHECK (
        estimated_savings_amount IS NULL OR estimated_savings_amount >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_budget_optimization_proposals_trip_id
    ON budget_optimization_proposals(trip_id);

CREATE INDEX IF NOT EXISTS idx_budget_optimization_proposals_job_id
    ON budget_optimization_proposals(job_id);

CREATE INDEX IF NOT EXISTS idx_budget_optimization_proposals_created_by_user_id
    ON budget_optimization_proposals(created_by_user_id);

CREATE INDEX IF NOT EXISTS idx_budget_optimization_proposals_status
    ON budget_optimization_proposals(status);

CREATE INDEX IF NOT EXISTS idx_budget_optimization_proposals_trip_created_at
    ON budget_optimization_proposals(trip_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_budget_optimization_proposals_trip_status
    ON budget_optimization_proposals(trip_id, status);
