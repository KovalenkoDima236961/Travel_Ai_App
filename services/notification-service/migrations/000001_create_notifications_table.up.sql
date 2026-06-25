CREATE TABLE IF NOT EXISTS notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    trip_id         UUID NULL,
    actor_user_id   UUID NULL,
    type            TEXT NOT NULL,
    title           TEXT NOT NULL,
    message         TEXT NOT NULL,
    entity_type     TEXT NULL,
    entity_id       UUID NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    read_at         TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT notifications_type_not_empty CHECK (char_length(type) > 0),
    CONSTRAINT notifications_title_not_empty CHECK (char_length(title) > 0),
    CONSTRAINT notifications_title_length_check CHECK (char_length(title) <= 200),
    CONSTRAINT notifications_message_not_empty CHECK (char_length(message) > 0),
    CONSTRAINT notifications_message_length_check CHECK (char_length(message) <= 1000)
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id
    ON notifications (user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_user_read_at
    ON notifications (user_id, read_at);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created_at
    ON notifications (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_trip_id
    ON notifications (trip_id);

CREATE INDEX IF NOT EXISTS idx_notifications_type
    ON notifications (type);

CREATE INDEX IF NOT EXISTS idx_notifications_entity
    ON notifications (entity_type, entity_id);
