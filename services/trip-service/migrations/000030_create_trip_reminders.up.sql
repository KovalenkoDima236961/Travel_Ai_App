CREATE TABLE IF NOT EXISTS trip_reminders (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NULL,
    category TEXT NOT NULL,
    priority TEXT NOT NULL DEFAULT 'medium',
    source TEXT NOT NULL DEFAULT 'manual',
    status TEXT NOT NULL DEFAULT 'pending',
    trigger_date DATE NOT NULL,
    trigger_time TIME NULL,
    timezone TEXT NULL,
    relative_offset_days INT NULL,
    assigned_to_user_id UUID NULL,
    checklist_item_id UUID NULL REFERENCES trip_checklist_items(id) ON DELETE SET NULL,
    related_day_number INT NULL,
    related_item_index INT NULL,
    related_item_id TEXT NULL,
    sent_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    completed_by_user_id UUID NULL,
    disabled_at TIMESTAMP NULL,
    disabled_by_user_id UUID NULL,
    cancelled_at TIMESTAMP NULL,
    cancelled_by_user_id UUID NULL,
    failed_at TIMESTAMP NULL,
    failure_reason TEXT NULL,
    metadata JSONB NULL,
    created_by_user_id UUID NULL,
    updated_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    deleted_by_user_id UUID NULL,
    CONSTRAINT trip_reminders_title_not_empty CHECK (length(btrim(title)) > 0),
    CONSTRAINT trip_reminders_category_check CHECK (category IN (
        'documents',
        'packing',
        'transport',
        'accommodation',
        'weather',
        'activities',
        'group',
        'checklist',
        'before_departure',
        'route',
        'safety',
        'other'
    )),
    CONSTRAINT trip_reminders_priority_check CHECK (priority IN (
        'low',
        'medium',
        'high',
        'critical'
    )),
    CONSTRAINT trip_reminders_source_check CHECK (source IN (
        'checklist',
        'route',
        'transport',
        'accommodation',
        'weather',
        'manual',
        'system',
        'regenerated'
    )),
    CONSTRAINT trip_reminders_status_check CHECK (status IN (
        'pending',
        'sent',
        'completed',
        'disabled',
        'cancelled',
        'failed'
    ))
);

CREATE INDEX IF NOT EXISTS idx_trip_reminders_trip_id ON trip_reminders(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_reminders_assigned_to_user_id ON trip_reminders(assigned_to_user_id);
CREATE INDEX IF NOT EXISTS idx_trip_reminders_status ON trip_reminders(status);
CREATE INDEX IF NOT EXISTS idx_trip_reminders_trigger_date ON trip_reminders(trigger_date);
CREATE INDEX IF NOT EXISTS idx_trip_reminders_status_trigger_date ON trip_reminders(status, trigger_date);
CREATE INDEX IF NOT EXISTS idx_trip_reminders_checklist_item_id ON trip_reminders(checklist_item_id);
CREATE INDEX IF NOT EXISTS idx_trip_reminders_deleted_at ON trip_reminders(deleted_at);
