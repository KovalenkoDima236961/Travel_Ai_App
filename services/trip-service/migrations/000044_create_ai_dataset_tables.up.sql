CREATE TABLE IF NOT EXISTS ai_dataset_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT NULL,
    task_type TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    minimum_quality_score NUMERIC NOT NULL DEFAULT 0.8,
    consent_required BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMP NULL,
    CONSTRAINT ai_dataset_projects_key_check CHECK (length(trim(key)) > 0),
    CONSTRAINT ai_dataset_projects_name_check CHECK (length(trim(name)) > 0),
    CONSTRAINT ai_dataset_projects_status_check CHECK (status IN ('draft', 'active', 'frozen', 'archived')),
    CONSTRAINT ai_dataset_projects_task_type_check CHECK (task_type IN (
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
    CONSTRAINT ai_dataset_projects_minimum_quality_score_check CHECK (
        minimum_quality_score >= 0 AND minimum_quality_score <= 1
    )
);

CREATE TABLE IF NOT EXISTS ai_training_consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT NULL,
    consent_version TEXT NOT NULL,
    status TEXT NOT NULL,
    granted_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_training_consents_scope_type_check CHECK (scope_type IN (
        'global_future_examples',
        'trip',
        'itinerary_version',
        'feedback_signal',
        'template',
        'recap'
    )),
    CONSTRAINT ai_training_consents_status_check CHECK (status IN ('granted', 'revoked')),
    CONSTRAINT ai_training_consents_scope_id_check CHECK (
        scope_type = 'global_future_examples' OR length(trim(coalesce(scope_id, ''))) > 0
    )
);

CREATE TABLE IF NOT EXISTS ai_training_examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_project_id UUID NOT NULL REFERENCES ai_dataset_projects(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL,
    source_entity_type TEXT NULL,
    source_entity_id TEXT NULL,
    user_id UUID NULL,
    trip_id UUID NULL REFERENCES trips(id) ON DELETE SET NULL,
    task_type TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'en',
    schema_version TEXT NOT NULL,
    input_json JSONB NOT NULL,
    grounding_json JSONB NULL,
    expected_output_json JSONB NOT NULL,
    negative_output_json JSONB NULL,
    labels_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    provenance_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    consent_status TEXT NOT NULL DEFAULT 'not_required',
    consent_record_id UUID NULL REFERENCES ai_training_consents(id) ON DELETE SET NULL,
    sanitization_status TEXT NOT NULL DEFAULT 'pending',
    quality_status TEXT NOT NULL DEFAULT 'pending',
    review_status TEXT NOT NULL DEFAULT 'pending',
    quality_score NUMERIC NULL,
    duplicate_group_id UUID NULL,
    split TEXT NULL,
    export_status TEXT NOT NULL DEFAULT 'not_exported',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    reviewed_by_user_id UUID NULL,
    reviewed_at TIMESTAMP NULL,
    rejected_reason TEXT NULL,
    CONSTRAINT ai_training_examples_source_type_check CHECK (length(trim(source_type)) > 0),
    CONSTRAINT ai_training_examples_task_type_check CHECK (task_type IN (
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
    CONSTRAINT ai_training_examples_consent_status_check CHECK (consent_status IN (
        'not_required',
        'pending',
        'granted',
        'revoked',
        'prohibited'
    )),
    CONSTRAINT ai_training_examples_sanitization_status_check CHECK (sanitization_status IN (
        'pending',
        'passed',
        'failed',
        'needs_review'
    )),
    CONSTRAINT ai_training_examples_quality_status_check CHECK (quality_status IN (
        'pending',
        'passed',
        'failed',
        'needs_review'
    )),
    CONSTRAINT ai_training_examples_review_status_check CHECK (review_status IN (
        'pending',
        'approved',
        'rejected',
        'needs_changes'
    )),
    CONSTRAINT ai_training_examples_split_check CHECK (split IS NULL OR split IN (
        'train',
        'validation',
        'test',
        'holdout'
    )),
    CONSTRAINT ai_training_examples_export_status_check CHECK (export_status IN (
        'not_exported',
        'exported',
        'invalidated'
    )),
    CONSTRAINT ai_training_examples_quality_score_check CHECK (
        quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1)
    )
);

CREATE TABLE IF NOT EXISTS ai_dataset_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_project_id UUID NOT NULL REFERENCES ai_dataset_projects(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'building',
    schema_version TEXT NOT NULL,
    example_count INT NOT NULL DEFAULT 0,
    train_count INT NOT NULL DEFAULT 0,
    validation_count INT NOT NULL DEFAULT 0,
    test_count INT NOT NULL DEFAULT 0,
    holdout_count INT NOT NULL DEFAULT 0,
    manifest_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    checksum TEXT NULL,
    export_path TEXT NULL,
    created_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    finalized_at TIMESTAMP NULL,
    invalidated_at TIMESTAMP NULL,
    invalidated_reason TEXT NULL,
    CONSTRAINT ai_dataset_versions_project_version_unique UNIQUE (dataset_project_id, version),
    CONSTRAINT ai_dataset_versions_status_check CHECK (status IN ('building', 'ready', 'exported', 'invalidated', 'archived')),
    CONSTRAINT ai_dataset_versions_count_check CHECK (
        example_count >= 0 AND train_count >= 0 AND validation_count >= 0 AND test_count >= 0 AND holdout_count >= 0
    )
);

CREATE TABLE IF NOT EXISTS ai_dataset_review_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_example_id UUID NULL REFERENCES ai_training_examples(id) ON DELETE SET NULL,
    dataset_version_id UUID NULL REFERENCES ai_dataset_versions(id) ON DELETE SET NULL,
    actor_user_id UUID NULL,
    action TEXT NOT NULL,
    old_status TEXT NULL,
    new_status TEXT NULL,
    reason TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_dataset_review_events_action_check CHECK (action IN (
        'candidate_created',
        'sanitization_passed',
        'sanitization_failed',
        'quality_scored',
        'approved',
        'rejected',
        'split_assigned',
        'exported',
        'invalidated',
        'consent_revoked'
    ))
);

CREATE INDEX IF NOT EXISTS idx_ai_training_examples_project_review_quality
    ON ai_training_examples(dataset_project_id, review_status, quality_status);
CREATE INDEX IF NOT EXISTS idx_ai_training_examples_task_language
    ON ai_training_examples(task_type, language);
CREATE INDEX IF NOT EXISTS idx_ai_training_examples_consent_status
    ON ai_training_examples(consent_status);
CREATE INDEX IF NOT EXISTS idx_ai_training_examples_split
    ON ai_training_examples(split);
CREATE INDEX IF NOT EXISTS idx_ai_training_examples_source
    ON ai_training_examples(source_type, source_entity_id);
CREATE INDEX IF NOT EXISTS idx_ai_training_examples_quality_score_desc
    ON ai_training_examples(quality_score DESC);
CREATE INDEX IF NOT EXISTS idx_ai_training_consents_user_status
    ON ai_training_consents(user_id, status);
CREATE INDEX IF NOT EXISTS idx_ai_training_consents_user_scope_created
    ON ai_training_consents(user_id, scope_type, scope_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_dataset_versions_project_created
    ON ai_dataset_versions(dataset_project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_dataset_review_events_example_created
    ON ai_dataset_review_events(training_example_id, created_at DESC);
