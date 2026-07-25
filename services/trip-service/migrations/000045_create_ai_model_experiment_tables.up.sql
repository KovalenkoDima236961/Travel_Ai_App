ALTER TABLE ai_dataset_projects
    DROP CONSTRAINT IF EXISTS ai_dataset_projects_task_type_check;
ALTER TABLE ai_dataset_projects
    ADD CONSTRAINT ai_dataset_projects_task_type_check CHECK (task_type IN (
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
    ));

ALTER TABLE ai_training_examples
    DROP CONSTRAINT IF EXISTS ai_training_examples_task_type_check;
ALTER TABLE ai_training_examples
    ADD CONSTRAINT ai_training_examples_task_type_check CHECK (task_type IN (
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
    ));

CREATE TABLE IF NOT EXISTS ai_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    model_family TEXT NOT NULL,
    base_model_name TEXT NOT NULL,
    base_model_revision TEXT NULL,
    license_name TEXT NULL,
    parameter_count TEXT NULL,
    context_length INT NULL,
    source_type TEXT NOT NULL,
    source_uri TEXT NULL,
    local_path TEXT NULL,
    model_checksum TEXT NULL,
    status TEXT NOT NULL DEFAULT 'available',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_models_key_check CHECK (length(trim(key)) > 0),
    CONSTRAINT ai_models_display_name_check CHECK (length(trim(display_name)) > 0),
    CONSTRAINT ai_models_model_family_check CHECK (length(trim(model_family)) > 0),
    CONSTRAINT ai_models_base_model_name_check CHECK (length(trim(base_model_name)) > 0),
    CONSTRAINT ai_models_context_length_check CHECK (context_length IS NULL OR context_length > 0),
    CONSTRAINT ai_models_source_type_check CHECK (source_type IN ('ollama', 'huggingface', 'local_path', 'gguf', 'mock')),
    CONSTRAINT ai_models_status_check CHECK (status IN ('available', 'unavailable', 'archived'))
);

CREATE TABLE IF NOT EXISTS ai_training_experiments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    task_type TEXT NOT NULL,
    base_model_id UUID NOT NULL REFERENCES ai_models(id) ON DELETE RESTRICT,
    dataset_version_id UUID NOT NULL REFERENCES ai_dataset_versions(id) ON DELETE RESTRICT,
    method TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    config_json JSONB NOT NULL,
    seed BIGINT NOT NULL,
    hardware_json JSONB NULL,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    failed_at TIMESTAMP NULL,
    error_code TEXT NULL,
    error_message TEXT NULL,
    training_metrics_json JSONB NULL,
    artifact_path TEXT NULL,
    artifact_checksum TEXT NULL,
    created_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_training_experiments_key_check CHECK (length(trim(key)) > 0),
    CONSTRAINT ai_training_experiments_name_check CHECK (length(trim(name)) > 0),
    CONSTRAINT ai_training_experiments_task_type_check CHECK (task_type IN (
        'grounded_itinerary_generation'
    )),
    CONSTRAINT ai_training_experiments_method_check CHECK (method IN ('lora', 'qlora')),
    CONSTRAINT ai_training_experiments_status_check CHECK (status IN (
        'draft',
        'validated',
        'queued',
        'running',
        'completed',
        'failed',
        'cancelled',
        'evaluating',
        'rejected',
        'promoted_staging',
        'promoted_production'
    )),
    CONSTRAINT ai_training_experiments_config_json_check CHECK (jsonb_typeof(config_json) = 'object'),
    CONSTRAINT ai_training_experiments_hardware_json_check CHECK (hardware_json IS NULL OR jsonb_typeof(hardware_json) = 'object'),
    CONSTRAINT ai_training_experiments_training_metrics_json_check CHECK (training_metrics_json IS NULL OR jsonb_typeof(training_metrics_json) = 'object')
);

CREATE TABLE IF NOT EXISTS ai_model_adapters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    base_model_id UUID NOT NULL REFERENCES ai_models(id) ON DELETE RESTRICT,
    experiment_id UUID NOT NULL REFERENCES ai_training_experiments(id) ON DELETE RESTRICT,
    adapter_type TEXT NOT NULL,
    adapter_path TEXT NOT NULL,
    adapter_checksum TEXT NOT NULL,
    task_type TEXT NOT NULL,
    dataset_version_id UUID NOT NULL REFERENCES ai_dataset_versions(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'candidate',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    promoted_at TIMESTAMP NULL,
    rejected_at TIMESTAMP NULL,
    rejection_reason TEXT NULL,
    CONSTRAINT ai_model_adapters_key_check CHECK (length(trim(key)) > 0),
    CONSTRAINT ai_model_adapters_adapter_type_check CHECK (adapter_type IN ('lora', 'qlora')),
    CONSTRAINT ai_model_adapters_adapter_path_check CHECK (length(trim(adapter_path)) > 0),
    CONSTRAINT ai_model_adapters_adapter_checksum_check CHECK (length(trim(adapter_checksum)) >= 32),
    CONSTRAINT ai_model_adapters_task_type_check CHECK (task_type IN (
        'grounded_itinerary_generation'
    )),
    CONSTRAINT ai_model_adapters_status_check CHECK (status IN (
        'candidate',
        'approved_for_staging',
        'approved_for_production',
        'rejected',
        'archived'
    ))
);

CREATE TABLE IF NOT EXISTS ai_model_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID NULL REFERENCES ai_training_experiments(id) ON DELETE SET NULL,
    model_variant TEXT NOT NULL,
    model_key TEXT NOT NULL,
    dataset_version_id UUID NOT NULL REFERENCES ai_dataset_versions(id) ON DELETE RESTRICT,
    evaluation_split TEXT NOT NULL,
    evaluator_version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    metrics_json JSONB NULL,
    report_path TEXT NULL,
    report_checksum TEXT NULL,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_model_evaluations_model_variant_check CHECK (model_variant IN (
        'base',
        'grounded_baseline',
        'fine_tuned_candidate'
    )),
    CONSTRAINT ai_model_evaluations_model_key_check CHECK (length(trim(model_key)) > 0),
    CONSTRAINT ai_model_evaluations_evaluation_split_check CHECK (evaluation_split IN (
        'validation',
        'test',
        'holdout'
    )),
    CONSTRAINT ai_model_evaluations_status_check CHECK (status IN (
        'queued',
        'running',
        'completed',
        'failed',
        'cancelled'
    )),
    CONSTRAINT ai_model_evaluations_metrics_json_check CHECK (metrics_json IS NULL OR jsonb_typeof(metrics_json) = 'object')
);

CREATE TABLE IF NOT EXISTS ai_model_promotion_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID NOT NULL REFERENCES ai_training_experiments(id) ON DELETE RESTRICT,
    adapter_id UUID NULL REFERENCES ai_model_adapters(id) ON DELETE SET NULL,
    decision TEXT NOT NULL,
    reviewer_user_id UUID NULL,
    reason TEXT NOT NULL,
    metrics_snapshot JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_model_promotion_decisions_decision_check CHECK (decision IN (
        'approve_staging',
        'approve_production',
        'reject',
        'needs_more_data',
        'needs_retraining'
    )),
    CONSTRAINT ai_model_promotion_decisions_reason_check CHECK (length(trim(reason)) > 0),
    CONSTRAINT ai_model_promotion_decisions_metrics_snapshot_check CHECK (jsonb_typeof(metrics_snapshot) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_ai_models_status
    ON ai_models(status);
CREATE INDEX IF NOT EXISTS idx_ai_training_experiments_dataset_status
    ON ai_training_experiments(dataset_version_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_training_experiments_task_status
    ON ai_training_experiments(task_type, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_model_adapters_experiment
    ON ai_model_adapters(experiment_id);
CREATE INDEX IF NOT EXISTS idx_ai_model_adapters_status
    ON ai_model_adapters(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_model_evaluations_experiment_split
    ON ai_model_evaluations(experiment_id, evaluation_split, model_variant);
CREATE INDEX IF NOT EXISTS idx_ai_model_promotion_decisions_experiment_created
    ON ai_model_promotion_decisions(experiment_id, created_at DESC);
