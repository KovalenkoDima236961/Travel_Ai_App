-- Provider-backed knowledge extends the curated store from 000042 rather than
-- replacing it. Observations stay separate from travel_places so that raw
-- provider evidence is auditable and a low-quality record can never silently
-- become grounding context.

-- Source metadata gains provider identity, refresh capability, and the
-- rate-limit bucket used by External Integrations quota management.
ALTER TABLE travel_knowledge_sources
    ADD COLUMN IF NOT EXISTS provider_name TEXT NULL,
    ADD COLUMN IF NOT EXISTS terms_url TEXT NULL,
    ADD COLUMN IF NOT EXISTS refresh_supported BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS rate_limit_category TEXT NULL;

-- Review and scoring state for knowledge records. quality_score already exists
-- from 000042 and is reused, not duplicated.
ALTER TABLE travel_places
    ADD COLUMN IF NOT EXISTS review_status TEXT NOT NULL DEFAULT 'auto',
    ADD COLUMN IF NOT EXISTS freshness_score NUMERIC NULL,
    ADD COLUMN IF NOT EXISTS source_trust_score NUMERIC NULL,
    ADD COLUMN IF NOT EXISTS duplicate_group_id UUID NULL,
    ADD COLUMN IF NOT EXISTS canonical_place_id UUID NULL,
    ADD COLUMN IF NOT EXISTS last_quality_checked_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS last_provider_refresh_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS rejected_reason TEXT NULL,
    ADD COLUMN IF NOT EXISTS approved_by_user_id UUID NULL,
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS merged_into_place_id UUID NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'travel_places_review_status_check'
    ) THEN
        ALTER TABLE travel_places
            ADD CONSTRAINT travel_places_review_status_check
            CHECK (review_status IN ('auto', 'approved', 'rejected', 'needs_review', 'merged'));
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'travel_places_quality_score_check'
    ) THEN
        ALTER TABLE travel_places
            ADD CONSTRAINT travel_places_quality_score_check
            CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1));
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'travel_places_freshness_score_check'
    ) THEN
        ALTER TABLE travel_places
            ADD CONSTRAINT travel_places_freshness_score_check
            CHECK (freshness_score IS NULL OR (freshness_score >= 0 AND freshness_score <= 1));
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'travel_places_source_trust_score_check'
    ) THEN
        ALTER TABLE travel_places
            ADD CONSTRAINT travel_places_source_trust_score_check
            CHECK (source_trust_score IS NULL OR (source_trust_score >= 0 AND source_trust_score <= 1));
    END IF;
END
$$;

-- Raw provider evidence. raw_payload is optional by policy: adapters whose
-- terms discourage storage leave it NULL and keep only normalized fields.
-- Provider credentials and private user data must never reach this table.
CREATE TABLE IF NOT EXISTS travel_provider_place_observations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    provider_place_id TEXT NOT NULL,
    destination_id UUID NULL REFERENCES travel_destinations(id) ON DELETE CASCADE,
    raw_name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    category TEXT NULL,
    latitude NUMERIC NULL,
    longitude NUMERIC NULL,
    address TEXT NULL,
    website TEXT NULL,
    opening_hours JSONB NULL,
    rating NUMERIC NULL,
    rating_count INT NULL,
    price_level TEXT NULL,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    raw_payload JSONB NULL,
    source_url TEXT NULL,
    license_name TEXT NULL,
    attribution TEXT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NULL,
    quality_score NUMERIC NULL,
    confidence NUMERIC NOT NULL DEFAULT 0.5,
    matched_place_id UUID NULL REFERENCES travel_places(id) ON DELETE SET NULL,
    match_status TEXT NOT NULL DEFAULT 'unmatched',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_provider_place_observations_provider_unique UNIQUE (provider, provider_place_id),
    CONSTRAINT travel_provider_place_observations_provider_check CHECK (length(trim(provider)) > 0),
    CONSTRAINT travel_provider_place_observations_name_check CHECK (length(trim(raw_name)) > 0),
    CONSTRAINT travel_provider_place_observations_match_status_check CHECK (match_status IN ('unmatched', 'matched', 'duplicate', 'rejected', 'needs_review')),
    CONSTRAINT travel_provider_place_observations_confidence_check CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT travel_provider_place_observations_quality_check CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1)),
    CONSTRAINT travel_provider_place_observations_latitude_check CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT travel_provider_place_observations_longitude_check CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE TABLE IF NOT EXISTS travel_place_duplicate_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    destination_id UUID NOT NULL REFERENCES travel_destinations(id) ON DELETE CASCADE,
    canonical_place_id UUID NULL REFERENCES travel_places(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'open',
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_by_user_id UUID NULL,
    resolved_at TIMESTAMPTZ NULL,
    CONSTRAINT travel_place_duplicate_groups_status_check CHECK (status IN ('open', 'merged', 'rejected', 'split'))
);

CREATE TABLE IF NOT EXISTS travel_place_duplicate_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES travel_place_duplicate_groups(id) ON DELETE CASCADE,
    place_id UUID NOT NULL REFERENCES travel_places(id) ON DELETE CASCADE,
    confidence NUMERIC NOT NULL,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_place_duplicate_group_members_unique UNIQUE (group_id, place_id),
    CONSTRAINT travel_place_duplicate_group_members_confidence_check CHECK (confidence >= 0 AND confidence <= 1)
);

-- Ops review/merge audit. Summaries only: no secrets, no raw provider payloads,
-- no private user content.
CREATE TABLE IF NOT EXISTS travel_knowledge_review_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    place_id UUID NULL REFERENCES travel_places(id) ON DELETE SET NULL,
    duplicate_group_id UUID NULL REFERENCES travel_place_duplicate_groups(id) ON DELETE SET NULL,
    actor_user_id UUID NULL,
    action TEXT NOT NULL,
    old_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    new_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_knowledge_review_events_action_check CHECK (action IN ('approved', 'rejected', 'needs_review', 'refreshed', 'merged', 'duplicate_rejected', 'group_split', 'quality_recomputed', 'ingested'))
);

CREATE INDEX IF NOT EXISTS idx_travel_provider_place_observations_provider ON travel_provider_place_observations (provider, provider_place_id);
CREATE INDEX IF NOT EXISTS idx_travel_provider_place_observations_destination_observed ON travel_provider_place_observations (destination_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_travel_provider_place_observations_match_status_observed ON travel_provider_place_observations (match_status, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_travel_places_destination_review_quality ON travel_places (destination_id, review_status, quality_score);
CREATE INDEX IF NOT EXISTS idx_travel_places_destination_provider_refresh ON travel_places (destination_id, last_provider_refresh_at);
CREATE INDEX IF NOT EXISTS idx_travel_places_duplicate_group ON travel_places (duplicate_group_id);
CREATE INDEX IF NOT EXISTS idx_travel_place_duplicate_groups_destination_status ON travel_place_duplicate_groups (destination_id, status);
CREATE INDEX IF NOT EXISTS idx_travel_knowledge_review_events_place_created ON travel_knowledge_review_events (place_id, created_at DESC);
