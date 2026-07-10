ALTER TABLE trips
    ADD COLUMN IF NOT EXISTS route_json JSONB NULL,
    ADD COLUMN IF NOT EXISTS trip_type TEXT NOT NULL DEFAULT 'single_destination';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'trips_trip_type_check'
    ) THEN
        ALTER TABLE trips
            ADD CONSTRAINT trips_trip_type_check
            CHECK (trip_type IN ('single_destination', 'multi_destination'));
    END IF;
END $$;
