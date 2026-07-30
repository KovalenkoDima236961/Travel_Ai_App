DROP TABLE IF EXISTS trip_presence;
DROP TABLE IF EXISTS trip_votes;
DROP TABLE IF EXISTS trip_suggestions;

DROP INDEX IF EXISTS idx_itinerary_comments_resolved_at;
DROP INDEX IF EXISTS idx_itinerary_comments_parent_comment_id;
DROP INDEX IF EXISTS idx_itinerary_comments_target;

ALTER TABLE itinerary_comments
    DROP CONSTRAINT IF EXISTS itinerary_comments_day_number_check,
    DROP CONSTRAINT IF EXISTS itinerary_comments_target_type_check,
    DROP COLUMN IF EXISTS edited_at,
    DROP COLUMN IF EXISTS attachments,
    DROP COLUMN IF EXISTS mentions,
    DROP COLUMN IF EXISTS resolved_by_user_id,
    DROP COLUMN IF EXISTS resolved_at,
    DROP COLUMN IF EXISTS parent_comment_id,
    DROP COLUMN IF EXISTS target_id,
    DROP COLUMN IF EXISTS target_type;

ALTER TABLE itinerary_comments
    ADD CONSTRAINT itinerary_comments_day_number_check CHECK (day_number > 0);

DROP TABLE IF EXISTS trip_invitations;

DROP INDEX IF EXISTS idx_trip_collaborators_last_seen_at;
DROP INDEX IF EXISTS idx_trip_collaborators_expires_at;

ALTER TABLE trip_collaborators
    DROP CONSTRAINT IF EXISTS trip_collaborators_role_check,
    DROP CONSTRAINT IF EXISTS trip_collaborators_status_check,
    DROP COLUMN IF EXISTS permissions,
    DROP COLUMN IF EXISTS last_seen_at,
    DROP COLUMN IF EXISTS revoked_at,
    DROP COLUMN IF EXISTS declined_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS message,
    DROP COLUMN IF EXISTS email;

ALTER TABLE trip_collaborators
    ADD CONSTRAINT trip_collaborators_role_check CHECK (role IN ('viewer', 'editor')),
    ADD CONSTRAINT trip_collaborators_status_check CHECK (status IN ('pending', 'accepted', 'removed'));
