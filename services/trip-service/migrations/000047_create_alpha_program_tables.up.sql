CREATE TABLE IF NOT EXISTS alpha_invites (
    id UUID PRIMARY KEY,
    code_hash TEXT NOT NULL UNIQUE,
    code_prefix TEXT NOT NULL,
    code_suffix TEXT NOT NULL,
    expires_at TIMESTAMPTZ NULL,
    max_activations INTEGER NOT NULL,
    current_activations INTEGER NOT NULL DEFAULT 0,
    creator_user_id UUID NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    tester_group TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT alpha_invites_max_activations_check CHECK (max_activations >= 1),
    CONSTRAINT alpha_invites_current_activations_check CHECK (current_activations >= 0),
    CONSTRAINT alpha_invites_tester_group_check CHECK (
        tester_group IN ('internal', 'external', 'qa', 'design_reviewer')
    )
);

CREATE TABLE IF NOT EXISTS alpha_invite_activations (
    id UUID PRIMARY KEY,
    invite_id UUID NOT NULL REFERENCES alpha_invites(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    activated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id TEXT NULL,
    UNIQUE (invite_id, user_id),
    UNIQUE (user_id)
);

CREATE TABLE IF NOT EXISTS alpha_waitlist (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    email_hash TEXT NOT NULL UNIQUE,
    email_domain TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'registered',
    invited_invite_id UUID NULL REFERENCES alpha_invites(id) ON DELETE SET NULL,
    source TEXT NOT NULL DEFAULT 'web',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    invited_at TIMESTAMPTZ NULL,
    accepted_at TIMESTAMPTZ NULL,
    declined_at TIMESTAMPTZ NULL,
    removed_at TIMESTAMPTZ NULL,
    CONSTRAINT alpha_waitlist_status_check CHECK (
        status IN ('registered', 'invited', 'accepted', 'declined', 'removed')
    )
);

CREATE TABLE IF NOT EXISTS alpha_participants (
    user_id UUID PRIMARY KEY,
    invite_id UUID NULL REFERENCES alpha_invites(id) ON DELETE SET NULL,
    alpha_participant BOOLEAN NOT NULL DEFAULT TRUE,
    tester_group TEXT NOT NULL DEFAULT 'external',
    invitation_date TIMESTAMPTZ NULL,
    first_login_at TIMESTAMPTZ NULL,
    first_trip_at TIMESTAMPTZ NULL,
    first_ai_generation_at TIMESTAMPTZ NULL,
    last_activity_at TIMESTAMPTZ NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT alpha_participants_tester_group_check CHECK (
        tester_group IN ('internal', 'external', 'qa', 'design_reviewer')
    )
);

CREATE TABLE IF NOT EXISTS product_analytics_events (
    id UUID PRIMARY KEY,
    user_id UUID NULL,
    session_id_hash TEXT NULL,
    event_name TEXT NOT NULL,
    feature TEXT NOT NULL,
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id_hash TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id TEXT NULL,
    correlation_id TEXT NULL,
    app_version TEXT NULL,
    browser_family TEXT NULL,
    os_family TEXT NULL,
    device_type TEXT NULL,
    source TEXT NOT NULL DEFAULT 'web',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_analytics_events_event_name_length_check CHECK (length(event_name) BETWEEN 1 AND 80),
    CONSTRAINT product_analytics_events_feature_length_check CHECK (length(feature) BETWEEN 1 AND 60)
);

CREATE TABLE IF NOT EXISTS alpha_feedback (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    category TEXT NOT NULL,
    title TEXT NOT NULL,
    description_sanitized TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    priority TEXT NOT NULL DEFAULT 'normal',
    owner_user_id UUID NULL,
    internal_notes TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    app_version TEXT NULL,
    browser_family TEXT NULL,
    os_family TEXT NULL,
    device_type TEXT NULL,
    request_id TEXT NULL,
    correlation_id TEXT NULL,
    provider TEXT NULL,
    model_alias TEXT NULL,
    prompt_version TEXT NULL,
    feature_flags JSONB NOT NULL DEFAULT '{}'::jsonb,
    attachment_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT alpha_feedback_category_check CHECK (
        category IN ('ai', 'ui', 'performance', 'bug', 'security', 'accessibility', 'feature_request', 'other')
    ),
    CONSTRAINT alpha_feedback_status_check CHECK (
        status IN ('open', 'triaged', 'in_progress', 'resolved', 'closed', 'duplicate')
    ),
    CONSTRAINT alpha_feedback_priority_check CHECK (
        priority IN ('low', 'normal', 'high', 'urgent')
    ),
    CONSTRAINT alpha_feedback_attachment_count_check CHECK (attachment_count >= 0)
);

CREATE TABLE IF NOT EXISTS alpha_feedback_attachments (
    id UUID PRIMARY KEY,
    feedback_id UUID NOT NULL REFERENCES alpha_feedback(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_sha256 TEXT NOT NULL,
    scan_status TEXT NOT NULL DEFAULT 'clean',
    storage_status TEXT NOT NULL DEFAULT 'metadata_only',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT alpha_feedback_attachments_size_check CHECK (size_bytes BETWEEN 1 AND 5242880),
    CONSTRAINT alpha_feedback_attachments_mime_check CHECK (
        mime_type IN ('image/png', 'image/jpeg', 'image/webp')
    ),
    CONSTRAINT alpha_feedback_attachments_scan_check CHECK (
        scan_status IN ('clean', 'rejected', 'not_configured')
    )
);

CREATE TABLE IF NOT EXISTS weekly_alpha_reports (
    id UUID PRIMARY KEY,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    summary_markdown TEXT NOT NULL,
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    generated_by_user_id UUID NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (week_start, week_end)
);

CREATE INDEX IF NOT EXISTS idx_alpha_invites_created_at ON alpha_invites(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alpha_invites_enabled ON alpha_invites(enabled, expires_at);
CREATE INDEX IF NOT EXISTS idx_alpha_waitlist_status_created_at ON alpha_waitlist(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alpha_participants_active_activity ON alpha_participants(active, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_product_analytics_events_user_time ON product_analytics_events(user_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_product_analytics_events_name_time ON product_analytics_events(event_name, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_product_analytics_events_feature_time ON product_analytics_events(feature, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_alpha_feedback_status_created_at ON alpha_feedback(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alpha_feedback_category_created_at ON alpha_feedback(category, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_weekly_alpha_reports_generated_at ON weekly_alpha_reports(generated_at DESC);
