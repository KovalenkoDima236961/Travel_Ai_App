CREATE TABLE IF NOT EXISTS trip_polls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL,
    title TEXT NOT NULL,
    description TEXT NULL,
    poll_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    allow_multiple_votes BOOLEAN NOT NULL DEFAULT FALSE,
    closes_at TIMESTAMP NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP NULL,
    closed_by_user_id UUID NULL,
    CONSTRAINT trip_polls_title_check CHECK (length(btrim(title)) > 0),
    CONSTRAINT trip_polls_poll_type_check CHECK (
        poll_type IN ('single_choice', 'multiple_choice', 'rating', 'yes_no', 'date_choice')
    ),
    CONSTRAINT trip_polls_status_check CHECK (status IN ('open', 'closed', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_trip_polls_trip_id ON trip_polls (trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_polls_status ON trip_polls (status);
CREATE INDEX IF NOT EXISTS idx_trip_polls_created_at ON trip_polls (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trip_polls_created_by_user_id ON trip_polls (created_by_user_id);

CREATE TABLE IF NOT EXISTS trip_poll_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES trip_polls(id) ON DELETE CASCADE,
    option_key TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_poll_options_label_check CHECK (length(btrim(label)) > 0),
    CONSTRAINT trip_poll_options_poll_option_key_unique UNIQUE (poll_id, option_key)
);

CREATE INDEX IF NOT EXISTS idx_trip_poll_options_poll_id ON trip_poll_options (poll_id);
CREATE INDEX IF NOT EXISTS idx_trip_poll_options_sort_order ON trip_poll_options (sort_order);

CREATE TABLE IF NOT EXISTS trip_poll_votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES trip_polls(id) ON DELETE CASCADE,
    option_id UUID NULL REFERENCES trip_poll_options(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    vote_value TEXT NULL,
    rating_value INT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_poll_votes_rating_check CHECK (
        rating_value IS NULL OR (rating_value >= 1 AND rating_value <= 5)
    ),
    CONSTRAINT trip_poll_votes_user_option_unique UNIQUE (poll_id, option_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_trip_poll_votes_poll_id ON trip_poll_votes (poll_id);
CREATE INDEX IF NOT EXISTS idx_trip_poll_votes_user_id ON trip_poll_votes (user_id);

CREATE TABLE IF NOT EXISTS itinerary_item_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    day_number INT NOT NULL,
    item_index INT NOT NULL,
    item_id TEXT NULL,
    user_id UUID NOT NULL,
    reaction TEXT NOT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT itinerary_item_reactions_position_check CHECK (day_number >= 1 AND item_index >= 0),
    CONSTRAINT itinerary_item_reactions_reaction_check CHECK (
        reaction IN ('want_to_do', 'neutral', 'skip', 'must_have')
    ),
    CONSTRAINT itinerary_item_reactions_user_item_unique UNIQUE (
        trip_id, day_number, item_index, user_id
    )
);

CREATE INDEX IF NOT EXISTS idx_itinerary_item_reactions_trip_id ON itinerary_item_reactions (trip_id);
CREATE INDEX IF NOT EXISTS idx_itinerary_item_reactions_user_id ON itinerary_item_reactions (user_id);
CREATE INDEX IF NOT EXISTS idx_itinerary_item_reactions_reaction ON itinerary_item_reactions (reaction);
CREATE INDEX IF NOT EXISTS idx_itinerary_item_reactions_item
    ON itinerary_item_reactions (trip_id, day_number, item_index);

CREATE TABLE IF NOT EXISTS trip_discovery_suggestion_votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES trip_discovery_sessions(id) ON DELETE CASCADE,
    suggestion_id TEXT NOT NULL,
    trip_id UUID NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    vote TEXT NOT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_discovery_suggestion_votes_vote_check CHECK (
        vote IN ('like', 'dislike', 'favorite', 'not_interested')
    ),
    CONSTRAINT trip_discovery_suggestion_votes_user_unique UNIQUE (
        session_id, suggestion_id, user_id
    )
);

CREATE INDEX IF NOT EXISTS idx_trip_discovery_suggestion_votes_session
    ON trip_discovery_suggestion_votes (session_id);
CREATE INDEX IF NOT EXISTS idx_trip_discovery_suggestion_votes_trip
    ON trip_discovery_suggestion_votes (trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_discovery_suggestion_votes_user
    ON trip_discovery_suggestion_votes (user_id);
