CREATE TABLE IF NOT EXISTS calendar_connections (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    provider TEXT NOT NULL,
    provider_account_email TEXT NULL,
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT NULL,
    token_expires_at TIMESTAMP NULL,
    scopes TEXT NULL,
    connected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMP NULL,
    status TEXT NOT NULL DEFAULT 'active',
    CONSTRAINT calendar_connections_provider_check CHECK (provider IN ('google')),
    CONSTRAINT calendar_connections_status_check CHECK (status IN ('active', 'disconnected')),
    CONSTRAINT calendar_connections_user_provider_unique UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS calendar_connections_user_id_idx ON calendar_connections(user_id);
CREATE INDEX IF NOT EXISTS calendar_connections_provider_idx ON calendar_connections(provider);
CREATE INDEX IF NOT EXISTS calendar_connections_status_idx ON calendar_connections(status);

CREATE TABLE IF NOT EXISTS calendar_oauth_states (
    state TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    provider TEXT NOT NULL,
    return_url TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    CONSTRAINT calendar_oauth_states_provider_check CHECK (provider IN ('google'))
);

CREATE INDEX IF NOT EXISTS calendar_oauth_states_user_id_idx ON calendar_oauth_states(user_id);
CREATE INDEX IF NOT EXISTS calendar_oauth_states_expires_at_idx ON calendar_oauth_states(expires_at);
