CREATE INDEX IF NOT EXISTS idx_trips_user_id_created_at
    ON trips (user_id, created_at DESC);
