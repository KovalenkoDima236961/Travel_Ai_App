DROP TABLE IF EXISTS push_subscriptions;

DELETE FROM notification_preferences
WHERE channel = 'push';

ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS notification_preferences_channel_check;

ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
        CHECK (channel IN ('in_app', 'email'));
