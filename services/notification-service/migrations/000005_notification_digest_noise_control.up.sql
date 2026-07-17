ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS digest_key TEXT NULL,
    ADD COLUMN IF NOT EXISTS dedupe_key TEXT NULL,
    ADD COLUMN IF NOT EXISTS grouped_count INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS digest_batch_id UUID NULL,
    ADD COLUMN IF NOT EXISTS delivery_mode TEXT NULL,
    ADD COLUMN IF NOT EXISTS delivery_status TEXT NULL,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS latest_event_at TIMESTAMP NOT NULL DEFAULT NOW();

-- Backfill existing rows with the same deterministic rules used by the
-- application. This keeps the pre-v2 notification center useful after the
-- migration instead of classifying every historical row as "system".
UPDATE notifications
SET priority = CASE
        WHEN type IN ('generation_job_failed', 'budget_optimization_failed',
            'offline_sync_conflict', 'calendar_sync_failed',
            'share_security_changed', 'settlement_overdue') THEN 'urgent'
        WHEN type IN ('collaboration_invited', 'trip_submitted_for_approval',
            'trip_changes_requested', 'checklist_item_overdue',
            'pre_trip_reminder_due', 'settlement_pending', 'route_changed') THEN 'high'
        WHEN type = 'checklist_item_completed' THEN 'low'
        ELSE 'normal'
    END,
    category = CASE
        WHEN type IN ('collaboration_invited', 'collaboration_accepted',
            'availability_requested', 'group_readiness_nudge',
            'availability_nudge', 'poll_vote_nudge', 'trip_poll_created',
            'trip_poll_closed') THEN 'collaboration'
        WHEN type IN ('collaborator_role_changed', 'collaborator_removed') THEN 'role_changes'
        WHEN type = 'comment_created' THEN 'comments'
        WHEN type IN ('checklist_item_assigned', 'checklist_item_completed',
            'checklist_item_overdue', 'checklist_generated',
            'checklist_assignment_nudge', 'reminder_task_nudge',
            'reminder_assigned') THEN 'checklist'
        WHEN type = 'pre_trip_reminder_due' THEN 'reminders'
        WHEN type = 'expense_added' THEN 'expenses'
        WHEN type IN ('settlement_paid', 'settlement_pending',
            'settlement_overdue', 'settlement_nudge') THEN 'settlements'
        WHEN type IN ('trip_submitted_for_approval', 'trip_approved',
            'trip_changes_requested', 'trip_approval_cancelled',
            'trip_approval_reset_to_draft') THEN 'approval'
        WHEN type IN ('budget_optimization_ready', 'budget_optimization_failed',
            'workspace_budget_created', 'workspace_budget_updated',
            'workspace_budget_archived', 'workspace_budget_exceeded',
            'workspace_budget_nearing_limit', 'budget_confidence_changed') THEN 'budget'
        WHEN type = 'trip_health_issue' THEN 'health'
        WHEN type = 'offline_sync_conflict' THEN 'offline_sync'
        WHEN type = 'calendar_sync_failed' THEN 'calendar'
        WHEN type IN ('generation_job_failed', 'itinerary_generated') THEN 'ai_generation'
        WHEN type = 'share_security_changed' THEN 'security'
        WHEN type = 'notification_digest' THEN 'system'
        ELSE 'trip_updates'
    END,
    latest_event_at = created_at;

UPDATE notifications
SET digest_key = CASE
    WHEN trip_id IS NOT NULL THEN 'trip:' || trip_id::text || ':' || category
    ELSE 'category:' || category || ':' || type
END
WHERE digest_key IS NULL;

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_priority_check,
    DROP CONSTRAINT IF EXISTS notifications_grouped_count_check,
    DROP CONSTRAINT IF EXISTS notifications_delivery_mode_check;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_priority_check
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    ADD CONSTRAINT notifications_grouped_count_check CHECK (grouped_count >= 1),
    ADD CONSTRAINT notifications_digest_key_length_check
        CHECK (digest_key IS NULL OR char_length(digest_key) <= 500),
    ADD CONSTRAINT notifications_dedupe_key_length_check
        CHECK (dedupe_key IS NULL OR char_length(dedupe_key) <= 500),
    ADD CONSTRAINT notifications_delivery_mode_check
        CHECK (delivery_mode IS NULL OR delivery_mode IN
            ('instant', 'hourly_digest', 'daily_digest', 'weekly_digest', 'muted'));

ALTER TABLE notification_preferences
    ADD COLUMN IF NOT EXISTS delivery_mode TEXT;

UPDATE notification_preferences
SET delivery_mode = CASE
    WHEN enabled = FALSE THEN 'muted'
    WHEN channel = 'in_app' THEN 'instant'
    WHEN channel = 'email' AND category IN ('collaboration', 'role_changes', 'pre_trip_reminders') THEN 'instant'
    WHEN channel = 'email' THEN 'daily_digest'
    WHEN channel = 'push' AND category IN (
        'collaboration', 'role_changes', 'pre_trip_reminders', 'checklist_reminders'
    ) THEN 'instant'
    WHEN channel = 'push' THEN 'muted'
    ELSE 'muted'
END
WHERE delivery_mode IS NULL;

ALTER TABLE notification_preferences
    ALTER COLUMN delivery_mode SET NOT NULL,
    ALTER COLUMN delivery_mode SET DEFAULT 'instant',
    DROP CONSTRAINT IF EXISTS notification_preferences_channel_check,
    DROP CONSTRAINT IF EXISTS notification_preferences_category_check,
    DROP CONSTRAINT IF EXISTS notification_preferences_delivery_mode_check;

ALTER TABLE notification_preferences
    ADD CONSTRAINT notification_preferences_channel_check
        CHECK (channel IN ('in_app', 'email', 'push')),
    ADD CONSTRAINT notification_preferences_category_check
        CHECK (category IN (
            'collaboration', 'comments', 'trip_updates', 'role_changes',
            'checklist', 'checklist_reminders', 'reminders', 'pre_trip_reminders',
            'expenses', 'settlements', 'approval', 'budget', 'health',
            'offline_sync', 'calendar', 'ai_generation', 'security', 'system'
        )),
    ADD CONSTRAINT notification_preferences_delivery_mode_check
        CHECK (delivery_mode IN ('instant', 'hourly_digest', 'daily_digest', 'weekly_digest', 'muted'));

CREATE TABLE IF NOT EXISTS notification_settings (
    user_id                       UUID PRIMARY KEY,
    quiet_hours_enabled           BOOLEAN NOT NULL DEFAULT FALSE,
    quiet_hours_start             TIME NOT NULL DEFAULT '22:00',
    quiet_hours_end               TIME NOT NULL DEFAULT '08:00',
    quiet_hours_timezone          TEXT NOT NULL DEFAULT 'UTC',
    urgent_bypasses_quiet_hours   BOOLEAN NOT NULL DEFAULT TRUE,
    daily_digest_time             TIME NOT NULL DEFAULT '08:00',
    weekly_digest_day             SMALLINT NOT NULL DEFAULT 1,
    weekly_digest_time            TIME NOT NULL DEFAULT '08:00',
    created_at                    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT notification_settings_weekly_day_check
        CHECK (weekly_digest_day BETWEEN 0 AND 6),
    CONSTRAINT notification_settings_timezone_not_empty
        CHECK (char_length(quiet_hours_timezone) > 0)
);

CREATE TABLE IF NOT EXISTS notification_trip_mutes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    trip_id     UUID NOT NULL,
    category    TEXT NULL,
    muted_until TIMESTAMP NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_trip_mutes_unique
    ON notification_trip_mutes (user_id, trip_id, COALESCE(category, ''));
CREATE INDEX IF NOT EXISTS idx_notification_trip_mutes_user_trip
    ON notification_trip_mutes (user_id, trip_id);

-- A channel-independent ledger makes exact-event dedupe atomic even when the
-- user muted in-app rows. The notification link is optional because an event
-- may still be delivered by email/push without creating an in-app row.
CREATE TABLE IF NOT EXISTS notification_event_dedupes (
    user_id         UUID NOT NULL,
    dedupe_key      TEXT NOT NULL,
    notification_id UUID NULL REFERENCES notifications(id) ON DELETE SET NULL,
    first_seen_at   TIMESTAMP NOT NULL,
    latest_event_at TIMESTAMP NOT NULL,
    grouped_count   INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, dedupe_key),
    CONSTRAINT notification_event_dedupes_key_not_empty
        CHECK (char_length(dedupe_key) > 0 AND char_length(dedupe_key) <= 500),
    CONSTRAINT notification_event_dedupes_grouped_count_check CHECK (grouped_count >= 1)
);
CREATE INDEX IF NOT EXISTS idx_notification_event_dedupes_latest
    ON notification_event_dedupes (latest_event_at);

INSERT INTO notification_event_dedupes
    (user_id, dedupe_key, notification_id, first_seen_at, latest_event_at, grouped_count)
SELECT DISTINCT ON (user_id, dedupe_key)
    user_id, dedupe_key, id, created_at, latest_event_at, grouped_count
FROM notifications
WHERE dedupe_key IS NOT NULL AND char_length(dedupe_key) BETWEEN 1 AND 500
ORDER BY user_id, dedupe_key, latest_event_at DESC
ON CONFLICT (user_id, dedupe_key) DO NOTHING;

CREATE TABLE IF NOT EXISTS notification_digest_batches (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    channel             TEXT NOT NULL,
    mode                TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    scheduled_for       TIMESTAMP NOT NULL,
    sent_at             TIMESTAMP NULL,
    attempts            INTEGER NOT NULL DEFAULT 0,
    next_attempt_at     TIMESTAMP NULL,
    error_code          TEXT NULL,
    error_message_safe  TEXT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT notification_digest_batches_channel_check
        CHECK (channel IN ('in_app', 'email', 'push')),
    CONSTRAINT notification_digest_batches_mode_check
        CHECK (mode IN ('hourly_digest', 'daily_digest', 'weekly_digest')),
    CONSTRAINT notification_digest_batches_status_check
        CHECK (status IN ('pending', 'processing', 'sent', 'failed', 'cancelled')),
    CONSTRAINT notification_digest_batches_attempts_check CHECK (attempts >= 0)
);

CREATE TABLE IF NOT EXISTS notification_digest_items (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id         UUID NOT NULL REFERENCES notification_digest_batches(id) ON DELETE CASCADE,
    notification_id  UUID NULL REFERENCES notifications(id) ON DELETE SET NULL,
    user_id          UUID NOT NULL,
    trip_id          UUID NULL,
    category         TEXT NOT NULL,
    priority         TEXT NOT NULL,
    digest_key       TEXT NOT NULL,
    title            TEXT NOT NULL,
    message          TEXT NOT NULL,
    metadata_json    JSONB NOT NULL DEFAULT '{}',
    event_count      INTEGER NOT NULL DEFAULT 1,
    latest_event_at  TIMESTAMP NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT notification_digest_items_priority_check
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT notification_digest_items_event_count_check CHECK (event_count >= 1)
);

ALTER TABLE notifications
    ADD CONSTRAINT notifications_digest_batch_id_fkey
    FOREIGN KEY (digest_batch_id) REFERENCES notification_digest_batches(id)
    ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_digest_pending_window
    ON notification_digest_batches (user_id, channel, mode, scheduled_for)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_notification_digest_batches_user_status_scheduled
    ON notification_digest_batches (user_id, status, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_notification_digest_batches_status_scheduled
    ON notification_digest_batches (status, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_notification_digest_items_batch
    ON notification_digest_items (batch_id);
CREATE INDEX IF NOT EXISTS idx_notification_digest_items_user_trip_category
    ON notification_digest_items (user_id, trip_id, category);
CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_digest_items_group
    ON notification_digest_items (batch_id, digest_key);
CREATE INDEX IF NOT EXISTS idx_notifications_user_trip_category_created
    ON notifications (user_id, trip_id, category, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read_created
    ON notifications (user_id, read_at, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_dedupe_recent
    ON notifications (user_id, dedupe_key, latest_event_at DESC)
    WHERE dedupe_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_user_digest_recent
    ON notifications (user_id, digest_key, latest_event_at DESC)
    WHERE digest_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_digest_batch
    ON notifications (digest_batch_id)
    WHERE digest_batch_id IS NOT NULL;
