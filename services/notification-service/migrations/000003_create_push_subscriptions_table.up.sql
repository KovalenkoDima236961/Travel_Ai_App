ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS notification_preferences_channel_check;

ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
        CHECK (channel IN ('in_app', 'email', 'push'));

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    endpoint        TEXT NOT NULL,
    p256dh          TEXT NOT NULL,
    auth            TEXT NOT NULL,
    user_agent      TEXT NULL,
    browser         TEXT NULL,
    device_label    TEXT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMP NULL,
    disabled_at     TIMESTAMP NULL,
    disable_reason  TEXT NULL,
    CONSTRAINT push_subscriptions_status_check
        CHECK (status IN ('active', 'disabled')),
    CONSTRAINT push_subscriptions_endpoint_not_empty
        CHECK (char_length(endpoint) > 0),
    CONSTRAINT push_subscriptions_p256dh_not_empty
        CHECK (char_length(p256dh) > 0),
    CONSTRAINT push_subscriptions_auth_not_empty
        CHECK (char_length(auth) > 0),
    CONSTRAINT push_subscriptions_endpoint_key
        UNIQUE (endpoint)
);

CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user_id
    ON push_subscriptions (user_id);

CREATE INDEX IF NOT EXISTS idx_push_subscriptions_status
    ON push_subscriptions (status);

CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user_status
    ON push_subscriptions (user_id, status);
