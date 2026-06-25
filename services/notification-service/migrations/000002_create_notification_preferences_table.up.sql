CREATE TABLE IF NOT EXISTS notification_preferences (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    channel     TEXT NOT NULL,
    category    TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT notification_preferences_channel_check
        CHECK (channel IN ('in_app', 'email')),
    CONSTRAINT notification_preferences_category_check
        CHECK (category IN ('collaboration', 'comments', 'trip_updates', 'role_changes')),
    CONSTRAINT notification_preferences_user_channel_category_key
        UNIQUE (user_id, channel, category)
);

CREATE INDEX IF NOT EXISTS idx_notification_preferences_user_id
    ON notification_preferences (user_id);

CREATE INDEX IF NOT EXISTS idx_notification_preferences_user_channel
    ON notification_preferences (user_id, channel);

-- The UNIQUE (user_id, channel, category) constraint above is backed by a
-- unique index, which also serves the upsert ON CONFLICT target.
