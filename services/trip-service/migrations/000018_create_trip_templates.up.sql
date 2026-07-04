CREATE TABLE IF NOT EXISTS trip_templates (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id             UUID NULL,
    created_by_user_id       UUID NOT NULL,
    source_trip_id           UUID NULL REFERENCES trips(id) ON DELETE SET NULL,
    title                    TEXT NOT NULL,
    description              TEXT NULL,
    destination_hint         TEXT NULL,
    duration_days            INT NOT NULL,
    default_currency         TEXT NULL,
    visibility               TEXT NOT NULL,
    template_json            JSONB NOT NULL,
    tags                     TEXT[] NOT NULL DEFAULT '{}',
    estimated_total_amount   NUMERIC(12, 2) NULL,
    estimated_total_currency TEXT NULL,
    status                   TEXT NOT NULL DEFAULT 'active',
    created_at               TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMP NOT NULL DEFAULT NOW(),
    archived_at              TIMESTAMP NULL,
    archived_by_user_id      UUID NULL,
    CONSTRAINT trip_templates_duration_days_positive CHECK (duration_days > 0),
    CONSTRAINT trip_templates_visibility_check CHECK (visibility IN ('private', 'workspace')),
    CONSTRAINT trip_templates_status_check CHECK (status IN ('active', 'archived')),
    CONSTRAINT trip_templates_visibility_workspace_check CHECK (
        (visibility = 'private' AND workspace_id IS NULL)
        OR (visibility = 'workspace' AND workspace_id IS NOT NULL)
    ),
    CONSTRAINT trip_templates_default_currency_check CHECK (
        default_currency IS NULL OR default_currency ~ '^[A-Z]{3}$'
    ),
    CONSTRAINT trip_templates_estimated_total_currency_check CHECK (
        estimated_total_currency IS NULL OR estimated_total_currency ~ '^[A-Z]{3}$'
    ),
    CONSTRAINT trip_templates_estimated_total_amount_check CHECK (
        estimated_total_amount IS NULL OR estimated_total_amount >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_templates_created_by_user_id
    ON trip_templates (created_by_user_id);

CREATE INDEX IF NOT EXISTS idx_trip_templates_workspace_id
    ON trip_templates (workspace_id);

CREATE INDEX IF NOT EXISTS idx_trip_templates_source_trip_id
    ON trip_templates (source_trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_templates_visibility
    ON trip_templates (visibility);

CREATE INDEX IF NOT EXISTS idx_trip_templates_status
    ON trip_templates (status);

CREATE INDEX IF NOT EXISTS idx_trip_templates_created_at_desc
    ON trip_templates (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_trip_templates_tags_gin
    ON trip_templates USING GIN (tags);
