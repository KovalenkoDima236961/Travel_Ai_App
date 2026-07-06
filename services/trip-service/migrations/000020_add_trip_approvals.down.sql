DROP TABLE IF EXISTS trip_approval_events;

DROP INDEX IF EXISTS idx_trips_approval_approved_at;
DROP INDEX IF EXISTS idx_trips_approval_submitted_at;
DROP INDEX IF EXISTS idx_trips_workspace_approval_status;

ALTER TABLE trips
    DROP CONSTRAINT IF EXISTS trips_approval_status_check;

ALTER TABLE trips
    DROP COLUMN IF EXISTS approval_last_status_changed_by_user_id,
    DROP COLUMN IF EXISTS approval_last_status_changed_at,
    DROP COLUMN IF EXISTS approval_decision_note,
    DROP COLUMN IF EXISTS approval_note,
    DROP COLUMN IF EXISTS approval_cancelled_by_user_id,
    DROP COLUMN IF EXISTS approval_cancelled_at,
    DROP COLUMN IF EXISTS approval_changes_requested_by_user_id,
    DROP COLUMN IF EXISTS approval_changes_requested_at,
    DROP COLUMN IF EXISTS approval_approved_by_user_id,
    DROP COLUMN IF EXISTS approval_approved_at,
    DROP COLUMN IF EXISTS approval_submitted_by_user_id,
    DROP COLUMN IF EXISTS approval_submitted_at,
    DROP COLUMN IF EXISTS approval_status;
