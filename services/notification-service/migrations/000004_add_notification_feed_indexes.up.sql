CREATE INDEX IF NOT EXISTS idx_notifications_user_unread_created
    ON notifications (user_id, created_at DESC, id DESC)
    WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_user_feed_created
    ON notifications (user_id, created_at DESC, id DESC);

