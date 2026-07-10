ALTER TABLE trips
    ADD COLUMN IF NOT EXISTS creation_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE IF NOT EXISTS trip_discovery_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    workspace_id        UUID NULL,
    parent_session_id   UUID NULL REFERENCES trip_discovery_sessions(id) ON DELETE SET NULL,
    mode                TEXT NOT NULL,
    prompt              TEXT NULL,
    output_language     TEXT NOT NULL DEFAULT 'en',
    status              TEXT NOT NULL DEFAULT 'completed',
    request_json        JSONB NOT NULL,
    response_json       JSONB NOT NULL,
    created_trip_id     UUID NULL REFERENCES trips(id) ON DELETE SET NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_discovery_mode_check
        CHECK (mode IN ('prompt', 'surprise', 'refine')),
    CONSTRAINT trip_discovery_status_check
        CHECK (status IN ('completed', 'failed', 'created_trip')),
    CONSTRAINT trip_discovery_language_check
        CHECK (output_language IN ('en', 'es', 'uk', 'fr'))
);

CREATE INDEX IF NOT EXISTS idx_trip_discovery_sessions_user_created
    ON trip_discovery_sessions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trip_discovery_sessions_workspace_created
    ON trip_discovery_sessions (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trip_discovery_sessions_created_trip
    ON trip_discovery_sessions (created_trip_id);
