CREATE TABLE IF NOT EXISTS trip_calendar_syncs (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    provider TEXT NOT NULL,
    external_calendar_id TEXT NOT NULL,
    external_event_id TEXT NOT NULL,
    external_event_link TEXT NULL,
    day_number INT NOT NULL,
    item_index INT NOT NULL,
    itinerary_revision INT NOT NULL,
    sync_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    last_synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT trip_calendar_syncs_provider_check CHECK (provider IN ('google')),
    CONSTRAINT trip_calendar_syncs_status_check CHECK (status IN ('active', 'deleted')),
    CONSTRAINT trip_calendar_syncs_day_number_check CHECK (day_number > 0),
    CONSTRAINT trip_calendar_syncs_item_index_check CHECK (item_index >= 0),
    CONSTRAINT trip_calendar_syncs_unique UNIQUE (trip_id, user_id, provider, sync_key)
);

CREATE INDEX IF NOT EXISTS trip_calendar_syncs_trip_id_idx ON trip_calendar_syncs(trip_id);
CREATE INDEX IF NOT EXISTS trip_calendar_syncs_user_id_idx ON trip_calendar_syncs(user_id);
CREATE INDEX IF NOT EXISTS trip_calendar_syncs_provider_idx ON trip_calendar_syncs(provider);
CREATE INDEX IF NOT EXISTS trip_calendar_syncs_trip_user_provider_idx ON trip_calendar_syncs(trip_id, user_id, provider);
CREATE INDEX IF NOT EXISTS trip_calendar_syncs_status_idx ON trip_calendar_syncs(status);
