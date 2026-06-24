ALTER TABLE itinerary_versions
    ADD COLUMN IF NOT EXISTS created_by_user_id UUID NULL;

CREATE TABLE IF NOT EXISTS trip_collaborators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    invited_by_user_id UUID NOT NULL,
    invited_at TIMESTAMP NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMP NULL,
    removed_at TIMESTAMP NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_collaborators_role_check CHECK (role IN ('viewer', 'editor')),
    CONSTRAINT trip_collaborators_status_check CHECK (status IN ('pending', 'accepted', 'removed')),
    CONSTRAINT trip_collaborators_trip_user_unique UNIQUE (trip_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_trip_collaborators_trip_id
    ON trip_collaborators (trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_collaborators_user_id
    ON trip_collaborators (user_id);

CREATE INDEX IF NOT EXISTS idx_trip_collaborators_status
    ON trip_collaborators (status);

CREATE INDEX IF NOT EXISTS idx_trip_collaborators_invited_by_user_id
    ON trip_collaborators (invited_by_user_id);
