CREATE TABLE trip_recaps (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NULL,
    status TEXT NOT NULL,
    recap_json JSONB NOT NULL,
    source_summary_json JSONB NULL,
    ai_metadata_json JSONB NULL,
    finalized_at TIMESTAMP NULL,
    archived_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT trip_recaps_status_check CHECK (status IN ('draft', 'generated', 'edited', 'finalized', 'archived'))
);

CREATE UNIQUE INDEX trip_recaps_one_active_per_trip_idx ON trip_recaps(trip_id) WHERE archived_at IS NULL;
CREATE INDEX trip_recaps_trip_id_idx ON trip_recaps(trip_id);
CREATE INDEX trip_recaps_created_by_user_id_idx ON trip_recaps(created_by_user_id);
CREATE INDEX trip_recaps_status_idx ON trip_recaps(status);
CREATE INDEX trip_recaps_updated_at_idx ON trip_recaps(updated_at DESC);

CREATE TABLE trip_recap_feedback (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    recap_id UUID NOT NULL REFERENCES trip_recaps(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    feedback_type TEXT NOT NULL,
    entity_type TEXT NULL,
    entity_id TEXT NULL,
    label TEXT NOT NULL,
    value TEXT NULL,
    approved_for_personalization BOOLEAN NOT NULL DEFAULT false,
    metadata_json JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX trip_recap_feedback_trip_id_idx ON trip_recap_feedback(trip_id);
CREATE INDEX trip_recap_feedback_recap_id_idx ON trip_recap_feedback(recap_id);
CREATE INDEX trip_recap_feedback_user_id_idx ON trip_recap_feedback(user_id);
CREATE INDEX trip_recap_feedback_approved_idx ON trip_recap_feedback(approved_for_personalization);
