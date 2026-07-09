UPDATE user_profiles
SET preferred_language = 'en'
WHERE preferred_language IS NULL
   OR preferred_language NOT IN ('en', 'es', 'uk', 'fr');

ALTER TABLE user_profiles
    ALTER COLUMN preferred_language SET DEFAULT 'en',
    ALTER COLUMN preferred_language SET NOT NULL;

ALTER TABLE user_profiles
    DROP CONSTRAINT IF EXISTS user_profiles_preferred_language_check;

ALTER TABLE user_profiles
    ADD CONSTRAINT user_profiles_preferred_language_check
    CHECK (preferred_language IN ('en', 'es', 'uk', 'fr'));
