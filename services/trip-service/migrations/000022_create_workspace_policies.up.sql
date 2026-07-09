CREATE TABLE workspace_policies (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT NULL,
    rules_json JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMP NULL,
    archived_by_user_id UUID NULL,
    CONSTRAINT workspace_policies_status_check CHECK (status IN ('active', 'archived'))
);

CREATE INDEX workspace_policies_workspace_id_idx
    ON workspace_policies (workspace_id);

CREATE INDEX workspace_policies_workspace_status_idx
    ON workspace_policies (workspace_id, status);

CREATE UNIQUE INDEX workspace_policies_one_active_idx
    ON workspace_policies (workspace_id)
    WHERE status = 'active';
