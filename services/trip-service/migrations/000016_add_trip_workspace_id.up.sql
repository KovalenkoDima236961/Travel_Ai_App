ALTER TABLE trips
    ADD COLUMN IF NOT EXISTS workspace_id UUID;

CREATE INDEX IF NOT EXISTS idx_trips_workspace_id ON trips (workspace_id);
CREATE INDEX IF NOT EXISTS idx_trips_workspace_created_at ON trips (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trips_user_workspace ON trips (user_id, workspace_id);
