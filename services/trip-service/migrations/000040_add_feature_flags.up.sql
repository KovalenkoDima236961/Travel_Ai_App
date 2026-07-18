CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL,
    value_type TEXT NOT NULL,
    bool_value BOOLEAN NULL,
    string_value TEXT NULL,
    int_value BIGINT NULL,
    environment TEXT NULL,
    scope_type TEXT NOT NULL DEFAULT 'global',
    scope_id TEXT NULL,
    description TEXT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    source TEXT NOT NULL DEFAULT 'db',
    created_by_user_id UUID NULL,
    updated_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT feature_flags_key_not_empty CHECK (length(trim(key)) > 0),
    CONSTRAINT feature_flags_value_type_check CHECK (value_type IN ('boolean', 'string', 'int')),
    CONSTRAINT feature_flags_scope_type_check CHECK (scope_type IN ('global', 'workspace', 'user'))
);

-- PostgreSQL treats NULLs as distinct in a standard UNIQUE constraint. The
-- expression index makes a global (NULL scope_id/environment) override unique.
CREATE UNIQUE INDEX IF NOT EXISTS idx_feature_flags_scope_unique
    ON feature_flags (key, COALESCE(environment, ''), scope_type, COALESCE(scope_id, ''));
CREATE INDEX IF NOT EXISTS idx_feature_flags_lookup
    ON feature_flags (key, environment, scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_feature_flags_environment_scope
    ON feature_flags (environment, scope_type);

CREATE TABLE IF NOT EXISTS feature_flag_audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_key TEXT NOT NULL,
    environment TEXT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT NULL,
    actor_user_id UUID NULL,
    action TEXT NOT NULL,
    old_value JSONB NULL,
    new_value JSONB NULL,
    reason TEXT NULL,
    request_id TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT feature_flag_audit_action_check CHECK (action IN ('created', 'updated', 'enabled', 'disabled', 'deleted', 'reset_to_default')),
    CONSTRAINT feature_flag_audit_scope_type_check CHECK (scope_type IN ('global', 'workspace', 'user'))
);

CREATE INDEX IF NOT EXISTS idx_feature_flag_audit_events_flag_created
    ON feature_flag_audit_events (flag_key, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feature_flag_audit_events_created
    ON feature_flag_audit_events (created_at DESC);
