CREATE TABLE IF NOT EXISTS trip_activity_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id         UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    actor_user_id   UUID NULL,
    event_type      TEXT NOT NULL,
    entity_type     TEXT NULL,
    entity_id       UUID NULL,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_activity_events_event_type_not_empty CHECK (char_length(event_type) > 0)
);

CREATE INDEX IF NOT EXISTS idx_trip_activity_events_trip_id
    ON trip_activity_events (trip_id);

-- Primary feed query: newest-first within a trip, with id as a stable tiebreaker
-- for keyset pagination.
CREATE INDEX IF NOT EXISTS idx_trip_activity_events_trip_created
    ON trip_activity_events (trip_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_trip_activity_events_actor_user_id
    ON trip_activity_events (actor_user_id);

CREATE INDEX IF NOT EXISTS idx_trip_activity_events_event_type
    ON trip_activity_events (event_type);

CREATE INDEX IF NOT EXISTS idx_trip_activity_events_entity
    ON trip_activity_events (entity_type, entity_id);
