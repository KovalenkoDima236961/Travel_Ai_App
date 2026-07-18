DROP INDEX IF EXISTS idx_trips_updated_at_archived_at;
DROP INDEX IF EXISTS idx_trips_start_date_archived_at;
DROP INDEX IF EXISTS idx_trips_workspace_archived_at;
DROP INDEX IF EXISTS idx_trips_user_archived_at;

ALTER TABLE trips DROP COLUMN IF EXISTS archive_reason;
ALTER TABLE trips DROP COLUMN IF EXISTS archived_by_user_id;
ALTER TABLE trips DROP COLUMN IF EXISTS archived_at;
