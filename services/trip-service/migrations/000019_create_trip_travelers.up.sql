CREATE TABLE IF NOT EXISTS trip_travelers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    email TEXT NULL,
    linked_user_id UUID NULL,
    role TEXT NOT NULL DEFAULT 'traveler',
    status TEXT NOT NULL DEFAULT 'active',
    created_by_user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    removed_at TIMESTAMP NULL,
    CONSTRAINT trip_travelers_role_check CHECK (role IN ('organizer', 'traveler')),
    CONSTRAINT trip_travelers_status_check CHECK (status IN ('active', 'removed'))
);

CREATE INDEX IF NOT EXISTS idx_trip_travelers_trip_id
    ON trip_travelers (trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_travelers_linked_user_id
    ON trip_travelers (linked_user_id);

CREATE INDEX IF NOT EXISTS idx_trip_travelers_status
    ON trip_travelers (status);

CREATE INDEX IF NOT EXISTS idx_trip_travelers_trip_status
    ON trip_travelers (trip_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_travelers_active_email_unique
    ON trip_travelers (trip_id, lower(email))
    WHERE email IS NOT NULL AND status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_travelers_active_linked_user_unique
    ON trip_travelers (trip_id, linked_user_id)
    WHERE linked_user_id IS NOT NULL AND status = 'active';
