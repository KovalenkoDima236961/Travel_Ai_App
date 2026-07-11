CREATE TABLE IF NOT EXISTS trip_checklists (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    title TEXT NOT NULL,
    summary TEXT NULL,
    generated_from_itinerary_revision INT NULL,
    generated_from_route_revision INT NULL,
    generated_by_user_id UUID NULL,
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMP NULL,
    archived_by_user_id UUID NULL,
    CONSTRAINT trip_checklists_status_check CHECK (status IN ('active', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_trip_checklists_trip_id ON trip_checklists(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_checklists_status ON trip_checklists(status);
CREATE INDEX IF NOT EXISTS idx_trip_checklists_created_at_desc ON trip_checklists(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_checklists_one_active_per_trip
    ON trip_checklists(trip_id)
    WHERE status = 'active';

CREATE TABLE IF NOT EXISTS trip_checklist_items (
    id UUID PRIMARY KEY,
    checklist_id UUID NOT NULL REFERENCES trip_checklists(id) ON DELETE CASCADE,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NULL,
    category TEXT NOT NULL,
    item_type TEXT NOT NULL DEFAULT 'packing',
    priority TEXT NOT NULL DEFAULT 'medium',
    quantity INT NULL,
    assigned_to_user_id UUID NULL,
    due_date DATE NULL,
    checked BOOLEAN NOT NULL DEFAULT false,
    checked_at TIMESTAMP NULL,
    checked_by_user_id UUID NULL,
    source TEXT NOT NULL DEFAULT 'manual',
    reason TEXT NULL,
    related_day_number INT NULL,
    related_item_index INT NULL,
    related_item_id TEXT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    metadata JSONB NULL,
    created_by_user_id UUID NULL,
    updated_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    deleted_by_user_id UUID NULL,
    CONSTRAINT trip_checklist_items_category_check CHECK (category IN (
        'documents',
        'clothing',
        'electronics',
        'health_safety',
        'transport',
        'accommodation',
        'activities',
        'food_water',
        'money',
        'before_departure',
        'group_items',
        'camping_hiking',
        'weather',
        'other'
    )),
    CONSTRAINT trip_checklist_items_item_type_check CHECK (item_type IN (
        'packing',
        'preparation',
        'booking_check',
        'document',
        'shared_group_item',
        'reminder',
        'safety_check',
        'other'
    )),
    CONSTRAINT trip_checklist_items_priority_check CHECK (priority IN (
        'low',
        'medium',
        'high',
        'critical'
    )),
    CONSTRAINT trip_checklist_items_source_check CHECK (source IN (
        'ai',
        'manual',
        'template',
        'regenerated',
        'system'
    )),
    CONSTRAINT trip_checklist_items_quantity_check CHECK (quantity IS NULL OR (quantity >= 1 AND quantity <= 99))
);

CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_checklist_id ON trip_checklist_items(checklist_id);
CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_trip_id ON trip_checklist_items(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_assigned_to_user_id ON trip_checklist_items(assigned_to_user_id);
CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_category ON trip_checklist_items(category);
CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_checked ON trip_checklist_items(checked);
CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_due_date ON trip_checklist_items(due_date);
CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_deleted_at ON trip_checklist_items(deleted_at);
