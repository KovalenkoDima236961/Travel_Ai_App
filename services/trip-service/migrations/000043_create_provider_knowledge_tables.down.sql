DROP INDEX IF EXISTS idx_travel_knowledge_review_events_place_created;
DROP INDEX IF EXISTS idx_travel_place_duplicate_groups_destination_status;
DROP INDEX IF EXISTS idx_travel_places_duplicate_group;
DROP INDEX IF EXISTS idx_travel_places_destination_provider_refresh;
DROP INDEX IF EXISTS idx_travel_places_destination_review_quality;
DROP INDEX IF EXISTS idx_travel_provider_place_observations_match_status_observed;
DROP INDEX IF EXISTS idx_travel_provider_place_observations_destination_observed;
DROP INDEX IF EXISTS idx_travel_provider_place_observations_provider;

DROP TABLE IF EXISTS travel_knowledge_review_events;
DROP TABLE IF EXISTS travel_place_duplicate_group_members;
DROP TABLE IF EXISTS travel_place_duplicate_groups;
DROP TABLE IF EXISTS travel_provider_place_observations;

ALTER TABLE travel_places
    DROP CONSTRAINT IF EXISTS travel_places_source_trust_score_check,
    DROP CONSTRAINT IF EXISTS travel_places_freshness_score_check,
    DROP CONSTRAINT IF EXISTS travel_places_quality_score_check,
    DROP CONSTRAINT IF EXISTS travel_places_review_status_check;

ALTER TABLE travel_places
    DROP COLUMN IF EXISTS merged_into_place_id,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS approved_by_user_id,
    DROP COLUMN IF EXISTS rejected_reason,
    DROP COLUMN IF EXISTS last_provider_refresh_at,
    DROP COLUMN IF EXISTS last_quality_checked_at,
    DROP COLUMN IF EXISTS canonical_place_id,
    DROP COLUMN IF EXISTS duplicate_group_id,
    DROP COLUMN IF EXISTS source_trust_score,
    DROP COLUMN IF EXISTS freshness_score,
    DROP COLUMN IF EXISTS review_status;

ALTER TABLE travel_knowledge_sources
    DROP COLUMN IF EXISTS rate_limit_category,
    DROP COLUMN IF EXISTS refresh_supported,
    DROP COLUMN IF EXISTS terms_url,
    DROP COLUMN IF EXISTS provider_name;
