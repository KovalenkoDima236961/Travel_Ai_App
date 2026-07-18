ALTER TABLE trips ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP NULL;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS archived_by_user_id UUID NULL;
ALTER TABLE trips ADD COLUMN IF NOT EXISTS archive_reason TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_trips_user_archived_at
    ON trips (user_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_trips_workspace_archived_at
    ON trips (workspace_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_trips_start_date_archived_at
    ON trips (start_date, archived_at);
CREATE INDEX IF NOT EXISTS idx_trips_updated_at_archived_at
    ON trips (updated_at, archived_at);
