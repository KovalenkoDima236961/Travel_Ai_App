CREATE TABLE IF NOT EXISTS itinerary_comments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id         UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    day_number      INT NOT NULL,
    item_index      INT NOT NULL,
    author_user_id  UUID NOT NULL,
    body            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP NULL,
    CONSTRAINT itinerary_comments_day_number_check CHECK (day_number > 0),
    CONSTRAINT itinerary_comments_item_index_check CHECK (item_index >= 0),
    CONSTRAINT itinerary_comments_status_check CHECK (status IN ('active', 'deleted')),
    CONSTRAINT itinerary_comments_body_length_check CHECK (char_length(body) <= 2000)
);

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_trip_id
    ON itinerary_comments (trip_id);

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_trip_day_item
    ON itinerary_comments (trip_id, day_number, item_index);

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_author_user_id
    ON itinerary_comments (author_user_id);

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_status
    ON itinerary_comments (status);

CREATE INDEX IF NOT EXISTS idx_itinerary_comments_created_at
    ON itinerary_comments (created_at);
