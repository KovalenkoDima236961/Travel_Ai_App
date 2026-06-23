CREATE TABLE IF NOT EXISTS itinerary_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id         UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL,
    version_number  INTEGER NOT NULL,
    source          TEXT NOT NULL,
    itinerary       JSONB NOT NULL,
    metadata        JSONB,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT itinerary_versions_trip_version_unique UNIQUE (trip_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_itinerary_versions_trip_id_version_number
    ON itinerary_versions (trip_id, version_number DESC);

CREATE INDEX IF NOT EXISTS idx_itinerary_versions_user_id_created_at
    ON itinerary_versions (user_id, created_at DESC);
