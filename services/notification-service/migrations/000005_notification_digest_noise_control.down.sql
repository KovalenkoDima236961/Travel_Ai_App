DROP INDEX IF EXISTS idx_notifications_digest_batch;
DROP INDEX IF EXISTS idx_notifications_user_digest_recent;
DROP INDEX IF EXISTS idx_notifications_user_dedupe_recent;
DROP INDEX IF EXISTS idx_notifications_user_read_created;
DROP INDEX IF EXISTS idx_notifications_user_trip_category_created;
ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_digest_batch_id_fkey;
DROP INDEX IF EXISTS idx_notification_digest_items_group;
DROP INDEX IF EXISTS idx_notification_digest_items_user_trip_category;
DROP INDEX IF EXISTS idx_notification_digest_items_batch;
DROP INDEX IF EXISTS idx_notification_digest_batches_status_scheduled;
DROP INDEX IF EXISTS idx_notification_digest_batches_user_status_scheduled;
DROP INDEX IF EXISTS idx_notification_digest_pending_window;
DROP TABLE IF EXISTS notification_digest_items;
DROP TABLE IF EXISTS notification_digest_batches;
DROP INDEX IF EXISTS idx_notification_trip_mutes_unique;
DROP INDEX IF EXISTS idx_notification_trip_mutes_user_trip;
DROP TABLE IF EXISTS notification_trip_mutes;
DROP INDEX IF EXISTS idx_notification_event_dedupes_latest;
DROP TABLE IF EXISTS notification_event_dedupes;
DROP TABLE IF EXISTS notification_settings;

ALTER TABLE notification_preferences
    DROP CONSTRAINT IF EXISTS notification_preferences_delivery_mode_check,
    DROP CONSTRAINT IF EXISTS notification_preferences_category_check,
    DROP CONSTRAINT IF EXISTS notification_preferences_channel_check,
    DROP COLUMN IF EXISTS delivery_mode;

DELETE FROM notification_preferences
WHERE category NOT IN ('collaboration', 'comments', 'trip_updates', 'role_changes');

ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
        CHECK (channel IN ('in_app', 'email', 'push')),
    ADD CONSTRAINT notification_preferences_category_check
        CHECK (category IN ('collaboration', 'comments', 'trip_updates', 'role_changes'));

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_delivery_mode_check,
    DROP CONSTRAINT IF EXISTS notifications_grouped_count_check,
    DROP CONSTRAINT IF EXISTS notifications_digest_key_length_check,
    DROP CONSTRAINT IF EXISTS notifications_dedupe_key_length_check,
    DROP CONSTRAINT IF EXISTS notifications_priority_check,
    DROP COLUMN IF EXISTS latest_event_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS delivery_status,
    DROP COLUMN IF EXISTS delivery_mode,
    DROP COLUMN IF EXISTS digest_batch_id,
    DROP COLUMN IF EXISTS grouped_count,
    DROP COLUMN IF EXISTS dedupe_key,
    DROP COLUMN IF EXISTS digest_key,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS priority;
