DROP INDEX IF EXISTS idx_trips_user_workspace;
DROP INDEX IF EXISTS idx_trips_workspace_created_at;
DROP INDEX IF EXISTS idx_trips_workspace_id;

ALTER TABLE trips
    DROP COLUMN IF EXISTS workspace_id;
