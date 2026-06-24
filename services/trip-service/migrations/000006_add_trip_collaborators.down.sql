DROP TABLE IF EXISTS trip_collaborators;

ALTER TABLE itinerary_versions
    DROP COLUMN IF EXISTS created_by_user_id;
