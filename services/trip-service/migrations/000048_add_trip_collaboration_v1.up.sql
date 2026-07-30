ALTER TABLE trip_collaborators
    ADD COLUMN IF NOT EXISTS email TEXT NULL,
    ADD COLUMN IF NOT EXISTS message TEXT NULL,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS declined_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '{}';

ALTER TABLE trip_collaborators
    DROP CONSTRAINT IF EXISTS trip_collaborators_role_check,
    DROP CONSTRAINT IF EXISTS trip_collaborators_status_check;

ALTER TABLE trip_collaborators
    ADD CONSTRAINT trip_collaborators_role_check CHECK (
        role IN ('viewer', 'editor', 'commenter', 'guest', 'admin')
    ),
    ADD CONSTRAINT trip_collaborators_status_check CHECK (
        status IN ('pending', 'accepted', 'declined', 'expired', 'revoked', 'removed')
    );

CREATE INDEX IF NOT EXISTS idx_trip_collaborators_expires_at
    ON trip_collaborators (expires_at);

CREATE INDEX IF NOT EXISTS idx_trip_collaborators_last_seen_at
    ON trip_collaborators (last_seen_at);

CREATE TABLE IF NOT EXISTS trip_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    inviter_user_id UUID NOT NULL,
    invited_user_id UUID NULL,
    email TEXT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    message TEXT NULL,
    expires_at TIMESTAMP NOT NULL,
    accepted_at TIMESTAMP NULL,
    declined_at TIMESTAMP NULL,
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_invitations_recipient_check CHECK (
        invited_user_id IS NOT NULL OR length(btrim(coalesce(email, ''))) > 0
    ),
    CONSTRAINT trip_invitations_role_check CHECK (
        role IN ('viewer', 'editor', 'commenter', 'guest', 'admin')
    ),
    CONSTRAINT trip_invitations_status_check CHECK (
        status IN ('pending', 'accepted', 'declined', 'expired', 'revoked')
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_invitations_trip_id
    ON trip_invitations (trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_invitations_invited_user_id
    ON trip_invitations (invited_user_id);

CREATE INDEX IF NOT EXISTS idx_trip_invitations_email
    ON trip_invitations (lower(email));

CREATE INDEX IF NOT EXISTS idx_trip_invitations_status
    ON trip_invitations (status);

CREATE INDEX IF NOT EXISTS idx_trip_invitations_expires_at
    ON trip_invitations (expires_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_invitations_pending_user_unique
    ON trip_invitations (trip_id, invited_user_id)
    WHERE invited_user_id IS NOT NULL AND status = 'pending';

CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_invitations_pending_email_unique
    ON trip_invitations (trip_id, lower(email))
    WHERE email IS NOT NULL AND status = 'pending';

ALTER TABLE itinerary_comments
    ADD COLUMN IF NOT EXISTS target_type TEXT NOT NULL DEFAULT 'itinerary_item',
    ADD COLUMN IF NOT EXISTS target_id TEXT NULL,
    ADD COLUMN IF NOT EXISTS parent_comment_id UUID NULL REFERENCES itinerary_comments(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS resolved_by_user_id UUID NULL,
    ADD COLUMN IF NOT EXISTS mentions JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS attachments JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS edited_at TIMESTAMP NULL;

ALTER TABLE itinerary_comments
    DROP CONSTRAINT IF EXISTS itinerary_comments_day_number_check,
    DROP CONSTRAINT IF EXISTS itinerary_comments_target_type_check;

ALTER TABLE itinerary_comments
    ADD CONSTRAINT itinerary_comments_day_number_check CHECK (day_number >= 0),
    ADD CONSTRAINT itinerary_comments_target_type_check CHECK (
        target_type IN ('trip', 'day', 'itinerary_item', 'budget_item', 'route', 'attachment')
    );

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_target
    ON itinerary_comments (trip_id, target_type, target_id);

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_parent_comment_id
    ON itinerary_comments (parent_comment_id);

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_resolved_at
    ON itinerary_comments (resolved_at);

CREATE TABLE IF NOT EXISTS trip_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    author_user_id UUID NOT NULL,
    suggestion_type TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    before_json JSONB NULL,
    after_json JSONB NULL,
    comment TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    applied_itinerary_revision INT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP NULL,
    resolved_by_user_id UUID NULL,
    CONSTRAINT trip_suggestions_type_check CHECK (
        suggestion_type IN ('activity_replacement', 'time_change', 'budget_adjustment', 'route_change', 'note')
    ),
    CONSTRAINT trip_suggestions_target_type_check CHECK (
        target_type IN ('trip', 'day', 'itinerary_item', 'budget_item', 'route', 'attachment')
    ),
    CONSTRAINT trip_suggestions_status_check CHECK (
        status IN ('open', 'accepted', 'rejected', 'resolved')
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_suggestions_trip_id
    ON trip_suggestions (trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_suggestions_author_user_id
    ON trip_suggestions (author_user_id);

CREATE INDEX IF NOT EXISTS idx_trip_suggestions_target
    ON trip_suggestions (trip_id, target_type, target_id);

CREATE INDEX IF NOT EXISTS idx_trip_suggestions_status
    ON trip_suggestions (status);

CREATE TABLE IF NOT EXISTS trip_votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    user_id UUID NOT NULL,
    vote_type TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_votes_target_type_check CHECK (
        target_type IN ('activity', 'restaurant', 'hotel', 'destination', 'suggestion')
    ),
    CONSTRAINT trip_votes_vote_type_check CHECK (
        vote_type IN ('thumbs_up', 'thumbs_down', 'heart', 'star')
    ),
    CONSTRAINT trip_votes_target_id_check CHECK (length(btrim(target_id)) > 0),
    CONSTRAINT trip_votes_user_target_unique UNIQUE (trip_id, target_type, target_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_trip_votes_trip_id
    ON trip_votes (trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_votes_target
    ON trip_votes (trip_id, target_type, target_id);

CREATE INDEX IF NOT EXISTS idx_trip_votes_user_id
    ON trip_votes (user_id);

CREATE TABLE IF NOT EXISTS trip_presence (
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    state TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    last_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (trip_id, user_id),
    CONSTRAINT trip_presence_state_check CHECK (
        state IN ('viewing', 'editing_itinerary', 'editing_budget', 'idle')
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_presence_last_seen_at
    ON trip_presence (last_seen_at);
