-- Workspace Approval Workflow v1: lightweight approval state on workspace trips
-- plus a dedicated approval-event history table.

ALTER TABLE trips
    ADD COLUMN IF NOT EXISTS approval_status TEXT NOT NULL DEFAULT 'not_required',
    ADD COLUMN IF NOT EXISTS approval_submitted_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS approval_submitted_by_user_id UUID NULL,
    ADD COLUMN IF NOT EXISTS approval_approved_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS approval_approved_by_user_id UUID NULL,
    ADD COLUMN IF NOT EXISTS approval_changes_requested_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS approval_changes_requested_by_user_id UUID NULL,
    ADD COLUMN IF NOT EXISTS approval_cancelled_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS approval_cancelled_by_user_id UUID NULL,
    ADD COLUMN IF NOT EXISTS approval_note TEXT NULL,
    ADD COLUMN IF NOT EXISTS approval_decision_note TEXT NULL,
    ADD COLUMN IF NOT EXISTS approval_last_status_changed_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS approval_last_status_changed_by_user_id UUID NULL;

-- Existing workspace trips predate approval and should start life as drafts so
-- they can be submitted; personal trips stay not_required.
UPDATE trips
    SET approval_status = 'draft'
    WHERE workspace_id IS NOT NULL AND approval_status = 'not_required';

ALTER TABLE trips
    DROP CONSTRAINT IF EXISTS trips_approval_status_check;
ALTER TABLE trips
    ADD CONSTRAINT trips_approval_status_check CHECK (
        approval_status IN (
            'not_required',
            'draft',
            'pending_approval',
            'changes_requested',
            'approved',
            'cancelled'
        )
    );

CREATE INDEX IF NOT EXISTS idx_trips_workspace_approval_status
    ON trips (workspace_id, approval_status);
CREATE INDEX IF NOT EXISTS idx_trips_approval_submitted_at
    ON trips (approval_submitted_at);
CREATE INDEX IF NOT EXISTS idx_trips_approval_approved_at
    ON trips (approval_approved_at);

CREATE TABLE IF NOT EXISTS trip_approval_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL,
    actor_user_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    from_status TEXT NULL,
    to_status TEXT NOT NULL,
    note TEXT NULL,
    checklist_snapshot JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_approval_events_event_type_check CHECK (
        event_type IN (
            'submitted',
            'approved',
            'changes_requested',
            'cancelled',
            'reset_to_draft'
        )
    ),
    CONSTRAINT trip_approval_events_to_status_check CHECK (
        to_status IN (
            'not_required',
            'draft',
            'pending_approval',
            'changes_requested',
            'approved',
            'cancelled'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_approval_events_trip_created_at
    ON trip_approval_events (trip_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trip_approval_events_workspace_created_at
    ON trip_approval_events (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trip_approval_events_event_type
    ON trip_approval_events (event_type);
