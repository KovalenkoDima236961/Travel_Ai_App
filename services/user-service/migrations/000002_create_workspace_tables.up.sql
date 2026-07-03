CREATE TABLE IF NOT EXISTS workspaces (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL UNIQUE,
    description         TEXT,
    created_by_user_id  UUID NOT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    archived_at         TIMESTAMP,
    CONSTRAINT workspaces_name_check CHECK (char_length(name) BETWEEN 2 AND 80),
    CONSTRAINT workspaces_description_check CHECK (
        description IS NULL OR char_length(description) <= 500
    )
);

CREATE TABLE IF NOT EXISTS workspace_members (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL,
    role                TEXT NOT NULL,
    status              TEXT NOT NULL,
    invited_by_user_id  UUID,
    invited_at          TIMESTAMP,
    joined_at           TIMESTAMP,
    removed_at          TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT workspace_members_role_check CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    CONSTRAINT workspace_members_status_check CHECK (status IN ('active', 'invited', 'removed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_members_active_invited_unique
    ON workspace_members (workspace_id, user_id)
    WHERE status IN ('active', 'invited');

CREATE INDEX IF NOT EXISTS idx_workspace_members_user_active
    ON workspace_members (user_id, status);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace_status
    ON workspace_members (workspace_id, status);

CREATE TABLE IF NOT EXISTS workspace_invitations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email               TEXT NOT NULL,
    invited_user_id     UUID,
    role                TEXT NOT NULL,
    status              TEXT NOT NULL,
    invited_by_user_id  UUID NOT NULL,
    token_hash          TEXT,
    expires_at          TIMESTAMP,
    accepted_at         TIMESTAMP,
    declined_at         TIMESTAMP,
    revoked_at          TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT workspace_invitations_role_check CHECK (role IN ('admin', 'member', 'viewer')),
    CONSTRAINT workspace_invitations_status_check CHECK (
        status IN ('pending', 'accepted', 'declined', 'revoked', 'expired')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_invitations_pending_email_unique
    ON workspace_invitations (workspace_id, lower(email))
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_workspace_invitations_invited_user_pending
    ON workspace_invitations (invited_user_id, status);

CREATE INDEX IF NOT EXISTS idx_workspace_invitations_email_pending
    ON workspace_invitations (lower(email), status);
