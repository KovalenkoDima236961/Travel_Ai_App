CREATE TABLE IF NOT EXISTS trip_shares (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id      UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL,
    share_token  TEXT NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    disabled_at  TIMESTAMP NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_shares_share_token
    ON trip_shares (share_token);

CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_shares_trip_id_unique
    ON trip_shares (trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_shares_user_id
    ON trip_shares (user_id);
