CREATE TABLE IF NOT EXISTS trips (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID,
    destination     TEXT NOT NULL,
    start_date      DATE,
    days            INT NOT NULL,
    budget_amount   NUMERIC(10, 2),
    budget_currency VARCHAR(3) DEFAULT 'EUR',
    travelers       INT DEFAULT 1,
    interests       JSONB NOT NULL DEFAULT '[]',
    pace            TEXT NOT NULL DEFAULT 'balanced',
    status          TEXT NOT NULL DEFAULT 'DRAFT',
    itinerary       JSONB,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trips_user_id ON trips (user_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips (status);
