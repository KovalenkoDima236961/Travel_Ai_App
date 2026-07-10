CREATE TABLE IF NOT EXISTS route_alternative_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    trip_id UUID NULL REFERENCES trips(id) ON DELETE CASCADE,
    workspace_id UUID NULL,
    source TEXT NOT NULL,
    prompt TEXT NULL,
    output_language TEXT NOT NULL DEFAULT 'en',
    status TEXT NOT NULL DEFAULT 'completed',
    request_json JSONB NOT NULL,
    response_json JSONB NOT NULL,
    selected_alternative_id TEXT NULL,
    created_trip_id UUID NULL REFERENCES trips(id) ON DELETE SET NULL,
    applied_to_trip_id UUID NULL REFERENCES trips(id) ON DELETE SET NULL,
    parent_session_id UUID NULL REFERENCES route_alternative_sessions(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT route_alternative_sessions_source_check CHECK (
        source IN ('pre_trip', 'existing_trip', 'discovery_refinement', 'route_refinement')
    ),
    CONSTRAINT route_alternative_sessions_status_check CHECK (
        status IN ('completed', 'failed', 'created_trip', 'applied', 'archived')
    )
);

CREATE INDEX IF NOT EXISTS idx_route_alternative_sessions_user_created
    ON route_alternative_sessions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_route_alternative_sessions_trip_created
    ON route_alternative_sessions (trip_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_route_alternative_sessions_workspace_created
    ON route_alternative_sessions (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_route_alternative_sessions_parent
    ON route_alternative_sessions (parent_session_id);
