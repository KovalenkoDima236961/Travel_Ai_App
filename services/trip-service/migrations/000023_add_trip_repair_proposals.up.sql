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
            'template_adaptation',
            'policy_repair'
        )
    );

CREATE TABLE IF NOT EXISTS trip_repair_proposals (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    job_id UUID NULL REFERENCES trip_generation_jobs(id) ON DELETE SET NULL,
    created_by_user_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    repair_mode TEXT NOT NULL,
    base_itinerary_revision INT NOT NULL,
    base_risk_score INT NULL,
    proposed_risk_score INT NULL,
    base_policy_status TEXT NULL,
    proposed_policy_status TEXT NULL,
    issues_json JSONB NOT NULL,
    proposal_json JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMP NULL,
    applied_by_user_id UUID NULL,
    discarded_at TIMESTAMP NULL,
    discarded_by_user_id UUID NULL,
    expired_at TIMESTAMP NULL,

    CONSTRAINT trip_repair_proposals_status_check CHECK (
        status IN ('pending', 'applied', 'discarded', 'expired', 'failed')
    ),
    CONSTRAINT trip_repair_proposals_repair_mode_check CHECK (
        repair_mode IN (
            'policy_compliance',
            'reduce_budget_risk',
            'fix_schedule_risk',
            'reduce_walking',
            'add_rest_time',
            'replace_disallowed_items',
            'selected_issues'
        )
    ),
    CONSTRAINT trip_repair_proposals_base_revision_check CHECK (
        base_itinerary_revision >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_repair_proposals_trip_id
    ON trip_repair_proposals(trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_repair_proposals_job_id
    ON trip_repair_proposals(job_id);

CREATE INDEX IF NOT EXISTS idx_trip_repair_proposals_status
    ON trip_repair_proposals(status);

CREATE INDEX IF NOT EXISTS idx_trip_repair_proposals_created_at
    ON trip_repair_proposals(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_trip_repair_proposals_trip_status
    ON trip_repair_proposals(trip_id, status);
