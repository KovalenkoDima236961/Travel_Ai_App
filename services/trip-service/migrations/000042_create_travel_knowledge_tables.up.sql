-- Grounding knowledge is owned by Trip Service: it is planning input and
-- evidence for itinerary validation. Source provenance is mandatory for data
-- that is not manually curated.
CREATE TABLE IF NOT EXISTS travel_knowledge_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_key TEXT NOT NULL UNIQUE,
    source_type TEXT NOT NULL,
    display_name TEXT NOT NULL,
    license_name TEXT NULL,
    license_url TEXT NULL,
    attribution TEXT NULL,
    trust_level TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_knowledge_sources_key_check CHECK (length(trim(source_key)) > 0),
    CONSTRAINT travel_knowledge_sources_type_check CHECK (source_type IN ('manual_curated', 'provider_place', 'open_data', 'user_approved_match', 'user_feedback', 'mock_test_data')),
    CONSTRAINT travel_knowledge_sources_trust_check CHECK (trust_level IN ('trusted_curated', 'trusted_provider', 'public_open_data', 'app_observed', 'user_feedback', 'mock', 'unknown'))
);

CREATE TABLE IF NOT EXISTS travel_destinations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name TEXT NOT NULL,
    country_code TEXT NULL,
    country_name TEXT NULL,
    region_name TEXT NULL,
    latitude NUMERIC NULL,
    longitude NUMERIC NULL,
    aliases JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_id UUID NULL REFERENCES travel_knowledge_sources(id) ON DELETE SET NULL,
    confidence NUMERIC NOT NULL DEFAULT 0.8,
    last_verified_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_destinations_name_check CHECK (length(trim(canonical_name)) > 0),
    CONSTRAINT travel_destinations_canonical_country_unique UNIQUE (canonical_name, country_code),
    CONSTRAINT travel_destinations_confidence_check CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT travel_destinations_latitude_check CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT travel_destinations_longitude_check CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE TABLE IF NOT EXISTS travel_places (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    destination_id UUID NOT NULL REFERENCES travel_destinations(id) ON DELETE CASCADE,
    canonical_name TEXT NOT NULL,
    category TEXT NOT NULL,
    subcategory TEXT NULL,
    latitude NUMERIC NULL,
    longitude NUMERIC NULL,
    address TEXT NULL,
    neighborhood TEXT NULL,
    aliases JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    typical_duration_minutes INT NULL,
    price_level TEXT NULL,
    estimated_cost JSONB NULL,
    opening_hours JSONB NULL,
    website TEXT NULL,
    provider_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_id UUID NULL REFERENCES travel_knowledge_sources(id) ON DELETE SET NULL,
    source_url TEXT NULL,
    license_name TEXT NULL,
    confidence NUMERIC NOT NULL DEFAULT 0.7,
    popularity_score NUMERIC NULL,
    quality_score NUMERIC NULL,
    family_friendly BOOLEAN NULL,
    rain_friendly BOOLEAN NULL,
    outdoor BOOLEAN NULL,
    avoid_if JSONB NOT NULL DEFAULT '[]'::jsonb,
    best_time_of_day JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_verified_at TIMESTAMPTZ NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_places_name_check CHECK (length(trim(canonical_name)) > 0),
    CONSTRAINT travel_places_destination_name_unique UNIQUE (destination_id, canonical_name),
    CONSTRAINT travel_places_category_check CHECK (category IN ('landmark', 'museum', 'park', 'neighborhood', 'viewpoint', 'market', 'restaurant', 'cafe', 'activity', 'nature', 'transport', 'other')),
    CONSTRAINT travel_places_confidence_check CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT travel_places_latitude_check CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT travel_places_longitude_check CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180)),
    CONSTRAINT travel_places_duration_check CHECK (typical_duration_minutes IS NULL OR typical_duration_minutes BETWEEN 5 AND 720),
    CONSTRAINT travel_places_status_check CHECK (status IN ('active', 'archived', 'rejected'))
);

CREATE TABLE IF NOT EXISTS travel_knowledge_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NULL REFERENCES travel_knowledge_sources(id) ON DELETE SET NULL,
    destination_id UUID NULL REFERENCES travel_destinations(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    content_type TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'en',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_url TEXT NULL,
    license_name TEXT NULL,
    attribution TEXT NULL,
    checksum TEXT NOT NULL,
    confidence NUMERIC NOT NULL DEFAULT 0.7,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_knowledge_documents_title_check CHECK (length(trim(title)) > 0),
    CONSTRAINT travel_knowledge_documents_destination_checksum_unique UNIQUE (destination_id, checksum),
    CONSTRAINT travel_knowledge_documents_destination_title_source_unique UNIQUE (destination_id, title, source_id),
    CONSTRAINT travel_knowledge_documents_content_check CHECK (length(trim(content)) > 0),
    CONSTRAINT travel_knowledge_documents_confidence_check CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT travel_knowledge_documents_status_check CHECK (status IN ('active', 'archived', 'rejected'))
);

CREATE TABLE IF NOT EXISTS travel_knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES travel_knowledge_documents(id) ON DELETE CASCADE,
    destination_id UUID NULL REFERENCES travel_destinations(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding_id TEXT NULL,
    checksum TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_knowledge_chunks_index_check CHECK (chunk_index >= 0),
    CONSTRAINT travel_knowledge_chunks_content_check CHECK (length(trim(content)) > 0),
    CONSTRAINT travel_knowledge_chunks_document_index_unique UNIQUE (document_id, chunk_index)
);

CREATE TABLE IF NOT EXISTS travel_ai_feedback_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NULL,
    trip_id UUID NULL REFERENCES trips(id) ON DELETE SET NULL,
    itinerary_revision INT NULL,
    day_number INT NULL,
    item_index INT NULL,
    signal_type TEXT NOT NULL,
    signal_value TEXT NOT NULL,
    place_id UUID NULL REFERENCES travel_places(id) ON DELETE SET NULL,
    provider_place_id TEXT NULL,
    destination TEXT NULL,
    item_snapshot JSONB NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    consent_for_training BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT travel_ai_feedback_signals_type_check CHECK (signal_type IN ('place_wrong', 'place_good', 'too_touristy', 'too_expensive', 'too_far', 'not_my_vibe', 'closed_or_unavailable', 'better_alternative_chosen', 'place_removed', 'place_match_changed', 'place_match_accepted', 'item_regenerated', 'repair_applied', 'item_skipped', 'item_completed', 'cost_edited')),
    CONSTRAINT travel_ai_feedback_signals_value_check CHECK (length(trim(signal_value)) > 0),
    CONSTRAINT travel_ai_feedback_signals_day_check CHECK (day_number IS NULL OR day_number >= 1),
    CONSTRAINT travel_ai_feedback_signals_item_check CHECK (item_index IS NULL OR item_index >= 0)
);

CREATE INDEX IF NOT EXISTS idx_travel_destinations_canonical_name ON travel_destinations (canonical_name);
CREATE INDEX IF NOT EXISTS idx_travel_destinations_country_code ON travel_destinations (country_code);
CREATE INDEX IF NOT EXISTS idx_travel_places_destination_category ON travel_places (destination_id, category);
CREATE INDEX IF NOT EXISTS idx_travel_places_destination_canonical_name ON travel_places (destination_id, canonical_name);
CREATE INDEX IF NOT EXISTS idx_travel_places_status_confidence ON travel_places (status, confidence);
CREATE INDEX IF NOT EXISTS idx_travel_places_last_verified_at ON travel_places (last_verified_at);
CREATE INDEX IF NOT EXISTS idx_travel_knowledge_documents_destination_status ON travel_knowledge_documents (destination_id, status);
CREATE INDEX IF NOT EXISTS idx_travel_knowledge_documents_checksum ON travel_knowledge_documents (checksum);
CREATE INDEX IF NOT EXISTS idx_travel_knowledge_chunks_document_index ON travel_knowledge_chunks (document_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_travel_ai_feedback_signals_trip_created_at ON travel_ai_feedback_signals (trip_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_travel_ai_feedback_signals_type_created_at ON travel_ai_feedback_signals (signal_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_travel_ai_feedback_signals_consent ON travel_ai_feedback_signals (consent_for_training);
