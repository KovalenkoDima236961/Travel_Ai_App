DROP INDEX IF EXISTS idx_trip_shares_expires_at;

ALTER TABLE trip_shares
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS password_required,
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS expires_at;
