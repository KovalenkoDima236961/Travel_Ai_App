CREATE TABLE IF NOT EXISTS ai_model_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment TEXT NOT NULL,
    deployment_key TEXT UNIQUE NOT NULL,
    model_id UUID NOT NULL REFERENCES ai_models(id) ON DELETE RESTRICT,
    adapter_id UUID NULL REFERENCES ai_model_adapters(id) ON DELETE RESTRICT,
    experiment_id UUID NULL REFERENCES ai_training_experiments(id) ON DELETE SET NULL,
    model_variant TEXT NOT NULL,
    status TEXT NOT NULL,
    task_type TEXT NOT NULL,
    traffic_mode TEXT NOT NULL,
    shadow_sample_percent NUMERIC NOT NULL DEFAULT 0,
    rollout_percent NUMERIC NOT NULL DEFAULT 0,
    allowlisted_user_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    allowlisted_workspace_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    internal_only BOOLEAN NOT NULL DEFAULT true,
    feature_flag_key TEXT NULL,
    assignment_salt TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    grounding_version TEXT NULL,
    validator_version TEXT NULL,
    config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by_user_id UUID NULL,
    updated_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMP NULL,
    paused_at TIMESTAMP NULL,
    retired_at TIMESTAMP NULL,
    CONSTRAINT ai_model_deployments_environment_check CHECK (length(trim(environment)) > 0),
    CONSTRAINT ai_model_deployments_deployment_key_check CHECK (length(trim(deployment_key)) > 0),
    CONSTRAINT ai_model_deployments_model_variant_check CHECK (model_variant IN (
        'grounded_baseline',
        'fine_tuned_candidate'
    )),
    CONSTRAINT ai_model_deployments_status_check CHECK (status IN (
        'registered',
        'candidate',
        'shadow',
        'internal',
        'allowlist',
        'staged_rollout',
        'active',
        'paused',
        'rejected',
        'retired'
    )),
    CONSTRAINT ai_model_deployments_task_type_check CHECK (task_type IN (
        'grounded_itinerary_generation',
        'itinerary_generation',
        'day_regeneration',
        'item_regeneration',
        'place_replacement',
        'policy_repair',
        'budget_optimization',
        'route_alternatives',
        'checklist_generation',
        'copilot_response',
        'recap_generation'
    )),
    CONSTRAINT ai_model_deployments_traffic_mode_check CHECK (traffic_mode IN (
        'disabled',
        'shadow',
        'internal',
        'allowlist',
        'percentage',
        'active'
    )),
    CONSTRAINT ai_model_deployments_shadow_sample_percent_check CHECK (
        shadow_sample_percent >= 0 AND shadow_sample_percent <= 100
    ),
    CONSTRAINT ai_model_deployments_rollout_percent_check CHECK (
        rollout_percent >= 0 AND rollout_percent <= 100
    ),
    CONSTRAINT ai_model_deployments_allowlisted_user_ids_check CHECK (
        jsonb_typeof(allowlisted_user_ids) = 'array'
    ),
    CONSTRAINT ai_model_deployments_allowlisted_workspace_ids_check CHECK (
        jsonb_typeof(allowlisted_workspace_ids) = 'array'
    ),
    CONSTRAINT ai_model_deployments_assignment_salt_check CHECK (length(trim(assignment_salt)) > 0),
    CONSTRAINT ai_model_deployments_prompt_version_check CHECK (length(trim(prompt_version)) > 0),
    CONSTRAINT ai_model_deployments_config_json_check CHECK (jsonb_typeof(config_json) = 'object'),
    CONSTRAINT ai_model_deployments_adapter_required_for_candidate_check CHECK (
        model_variant <> 'fine_tuned_candidate' OR adapter_id IS NOT NULL
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_model_deployments_one_active_per_env_task
    ON ai_model_deployments(environment, task_type)
    WHERE status = 'active' AND traffic_mode = 'active';

CREATE INDEX IF NOT EXISTS idx_ai_model_deployments_environment_task_status
    ON ai_model_deployments(environment, task_type, status);

CREATE TABLE IF NOT EXISTS ai_model_deployment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES ai_model_deployments(id) ON DELETE CASCADE,
    actor_user_id UUID NULL,
    action TEXT NOT NULL,
    old_status TEXT NULL,
    new_status TEXT NULL,
    old_config JSONB NULL,
    new_config JSONB NULL,
    reason TEXT NOT NULL,
    request_id TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_model_deployment_events_action_check CHECK (action IN (
        'created',
        'configured',
        'enabled_shadow',
        'enabled_internal',
        'enabled_allowlist',
        'rollout_changed',
        'paused',
        'resumed',
        'activated',
        'rejected',
        'retired',
        'guardrail_paused',
        'rollback'
    )),
    CONSTRAINT ai_model_deployment_events_reason_check CHECK (length(trim(reason)) > 0),
    CONSTRAINT ai_model_deployment_events_old_config_check CHECK (
        old_config IS NULL OR jsonb_typeof(old_config) = 'object'
    ),
    CONSTRAINT ai_model_deployment_events_new_config_check CHECK (
        new_config IS NULL OR jsonb_typeof(new_config) = 'object'
    )
);

CREATE INDEX IF NOT EXISTS idx_ai_model_deployment_events_deployment_created
    ON ai_model_deployment_events(deployment_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_model_request_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_key TEXT UNIQUE NOT NULL,
    user_id UUID NULL,
    workspace_id UUID NULL,
    trip_id UUID NULL,
    task_type TEXT NOT NULL,
    environment TEXT NOT NULL,
    baseline_deployment_id UUID NOT NULL REFERENCES ai_model_deployments(id) ON DELETE RESTRICT,
    candidate_deployment_id UUID NULL REFERENCES ai_model_deployments(id) ON DELETE SET NULL,
    assignment_type TEXT NOT NULL,
    bucket INT NULL,
    candidate_user_visible BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_model_request_assignments_request_key_check CHECK (length(trim(request_key)) > 0),
    CONSTRAINT ai_model_request_assignments_task_type_check CHECK (length(trim(task_type)) > 0),
    CONSTRAINT ai_model_request_assignments_environment_check CHECK (length(trim(environment)) > 0),
    CONSTRAINT ai_model_request_assignments_assignment_type_check CHECK (assignment_type IN (
        'baseline_only',
        'shadow',
        'internal_candidate',
        'allowlist_candidate',
        'percentage_candidate',
        'forced_ops_test'
    )),
    CONSTRAINT ai_model_request_assignments_bucket_check CHECK (bucket IS NULL OR (bucket >= 0 AND bucket <= 9999)),
    CONSTRAINT ai_model_request_assignments_candidate_visibility_check CHECK (
        candidate_user_visible = false OR candidate_deployment_id IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_ai_model_request_assignments_user_created
    ON ai_model_request_assignments(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_model_request_assignments_trip_created
    ON ai_model_request_assignments(trip_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_model_online_comparisons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_assignment_id UUID NOT NULL REFERENCES ai_model_request_assignments(id) ON DELETE CASCADE,
    request_key TEXT NOT NULL,
    trip_id UUID NULL,
    task_type TEXT NOT NULL,
    baseline_deployment_id UUID NOT NULL REFERENCES ai_model_deployments(id) ON DELETE RESTRICT,
    candidate_deployment_id UUID NOT NULL REFERENCES ai_model_deployments(id) ON DELETE RESTRICT,
    baseline_result_status TEXT NOT NULL,
    candidate_result_status TEXT NOT NULL,
    baseline_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    candidate_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    metric_deltas JSONB NOT NULL DEFAULT '{}'::jsonb,
    baseline_latency_ms INT NULL,
    candidate_latency_ms INT NULL,
    baseline_repair_attempted BOOLEAN NOT NULL DEFAULT false,
    candidate_repair_attempted BOOLEAN NOT NULL DEFAULT false,
    baseline_error_code TEXT NULL,
    candidate_error_code TEXT NULL,
    guardrail_status TEXT NOT NULL DEFAULT 'passed',
    comparison_status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP NULL,
    CONSTRAINT ai_model_online_comparisons_request_assignment_unique UNIQUE (request_assignment_id),
    CONSTRAINT ai_model_online_comparisons_request_key_check CHECK (length(trim(request_key)) > 0),
    CONSTRAINT ai_model_online_comparisons_task_type_check CHECK (length(trim(task_type)) > 0),
    CONSTRAINT ai_model_online_comparisons_metrics_check CHECK (
        jsonb_typeof(baseline_metrics) = 'object'
        AND jsonb_typeof(candidate_metrics) = 'object'
        AND jsonb_typeof(metric_deltas) = 'object'
    ),
    CONSTRAINT ai_model_online_comparisons_latency_check CHECK (
        (baseline_latency_ms IS NULL OR baseline_latency_ms >= 0)
        AND (candidate_latency_ms IS NULL OR candidate_latency_ms >= 0)
    ),
    CONSTRAINT ai_model_online_comparisons_comparison_status_check CHECK (comparison_status IN (
        'pending',
        'completed',
        'baseline_failed',
        'candidate_failed',
        'timed_out',
        'skipped',
        'invalid'
    )),
    CONSTRAINT ai_model_online_comparisons_guardrail_status_check CHECK (guardrail_status IN (
        'passed',
        'warning',
        'failed',
        'critical',
        'not_evaluated'
    ))
);

CREATE INDEX IF NOT EXISTS idx_ai_model_online_comparisons_candidate_created
    ON ai_model_online_comparisons(candidate_deployment_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_model_online_comparisons_guardrail_created
    ON ai_model_online_comparisons(guardrail_status, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_model_rollout_windows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES ai_model_deployments(id) ON DELETE CASCADE,
    environment TEXT NOT NULL,
    window_start TIMESTAMP NOT NULL,
    window_end TIMESTAMP NOT NULL,
    sample_count INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    failure_count INT NOT NULL DEFAULT 0,
    metrics_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    guardrail_status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_model_rollout_windows_unique UNIQUE (deployment_id, window_start, window_end),
    CONSTRAINT ai_model_rollout_windows_environment_check CHECK (length(trim(environment)) > 0),
    CONSTRAINT ai_model_rollout_windows_window_check CHECK (window_end > window_start),
    CONSTRAINT ai_model_rollout_windows_counts_check CHECK (
        sample_count >= 0 AND success_count >= 0 AND failure_count >= 0
        AND success_count + failure_count <= sample_count
    ),
    CONSTRAINT ai_model_rollout_windows_metrics_json_check CHECK (jsonb_typeof(metrics_json) = 'object'),
    CONSTRAINT ai_model_rollout_windows_guardrail_status_check CHECK (guardrail_status IN (
        'insufficient_data',
        'passing',
        'warning',
        'failing',
        'critical'
    ))
);

CREATE INDEX IF NOT EXISTS idx_ai_model_rollout_windows_deployment_start
    ON ai_model_rollout_windows(deployment_id, window_start DESC);

CREATE TABLE IF NOT EXISTS ai_shadow_input_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_assignment_id UUID NOT NULL REFERENCES ai_model_request_assignments(id) ON DELETE CASCADE,
    encrypted_payload BYTEA NOT NULL,
    payload_checksum TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    consumed_at TIMESTAMP NULL,
    CONSTRAINT ai_shadow_input_snapshots_assignment_unique UNIQUE (request_assignment_id),
    CONSTRAINT ai_shadow_input_snapshots_payload_checksum_check CHECK (length(trim(payload_checksum)) >= 32),
    CONSTRAINT ai_shadow_input_snapshots_expires_check CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS idx_ai_shadow_input_snapshots_expires
    ON ai_shadow_input_snapshots(expires_at);

CREATE TABLE IF NOT EXISTS ai_model_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    generation_job_id UUID NULL REFERENCES trip_generation_jobs(id) ON DELETE SET NULL,
    itinerary_version_id UUID NULL REFERENCES itinerary_versions(id) ON DELETE SET NULL,
    request_assignment_id UUID NULL REFERENCES ai_model_request_assignments(id) ON DELETE SET NULL,
    deployment_id UUID NULL REFERENCES ai_model_deployments(id) ON DELETE SET NULL,
    user_id UUID NOT NULL,
    feedback TEXT NOT NULL,
    note_sanitized TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_model_feedback_feedback_check CHECK (feedback IN (
        'better_than_standard',
        'worse_than_standard',
        'bad_places',
        'bad_schedule',
        'too_slow',
        'wrong_language',
        'formatting_problem',
        'other'
    )),
    CONSTRAINT ai_model_feedback_note_sanitized_check CHECK (
        note_sanitized IS NULL OR length(note_sanitized) <= 500
    )
);

CREATE INDEX IF NOT EXISTS idx_ai_model_feedback_deployment_created
    ON ai_model_feedback(deployment_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_model_feedback_trip_created
    ON ai_model_feedback(trip_id, created_at DESC);
