CREATE TABLE IF NOT EXISTS trip_availability_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    available_ranges_json JSONB NOT NULL DEFAULT '[]',
    unavailable_ranges_json JSONB NOT NULL DEFAULT '[]',
    preferred_ranges_json JSONB NOT NULL DEFAULT '[]',
    min_trip_days INT NULL,
    max_trip_days INT NULL,
    timezone TEXT NULL,
    notes TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_availability_responses_trip_user_unique UNIQUE (trip_id, user_id),
    CONSTRAINT trip_availability_responses_min_days_check CHECK (
        min_trip_days IS NULL OR min_trip_days > 0
    ),
    CONSTRAINT trip_availability_responses_max_days_check CHECK (
        max_trip_days IS NULL OR max_trip_days > 0
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_availability_responses_trip_id
    ON trip_availability_responses (trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_availability_responses_user_id
    ON trip_availability_responses (user_id);
CREATE INDEX IF NOT EXISTS idx_trip_availability_responses_updated_at
    ON trip_availability_responses (updated_at DESC);
