CREATE TABLE IF NOT EXISTS user_profiles (
    user_id             UUID PRIMARY KEY,
    display_name        TEXT,
    home_city           TEXT,
    home_country        TEXT,
    preferred_currency  TEXT NOT NULL DEFAULT 'EUR',
    preferred_language  TEXT NOT NULL DEFAULT 'en',
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT user_profiles_currency_check CHECK (preferred_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT user_profiles_language_check CHECK (char_length(preferred_language) BETWEEN 2 AND 10)
);

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id                   UUID PRIMARY KEY,
    travel_styles             JSONB NOT NULL DEFAULT '[]',
    pace                      TEXT NOT NULL DEFAULT 'balanced',
    max_walking_km_per_day    NUMERIC,
    food_preferences          JSONB NOT NULL DEFAULT '[]',
    avoid                     JSONB NOT NULL DEFAULT '[]',
    preferred_transport       JSONB NOT NULL DEFAULT '[]',
    accommodation_style       JSONB NOT NULL DEFAULT '[]',
    dietary_restrictions      JSONB NOT NULL DEFAULT '[]',
    created_at                TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT user_preferences_pace_check CHECK (pace IN ('relaxed', 'balanced', 'intensive')),
    CONSTRAINT user_preferences_max_walking_check CHECK (
        max_walking_km_per_day IS NULL
        OR (max_walking_km_per_day >= 0 AND max_walking_km_per_day <= 50)
    )
);

