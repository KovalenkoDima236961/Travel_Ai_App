CREATE TABLE IF NOT EXISTS personalization_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    workspace_id UUID NULL,
    trip_id UUID NULL REFERENCES trips(id) ON DELETE SET NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NULL,
    feedback_type TEXT NOT NULL,
    feedback_value TEXT NULL,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT personalization_feedback_entity_type_check CHECK (entity_type IN ('destination_suggestion', 'route_alternative', 'itinerary_item', 'template', 'budget_suggestion', 'checklist_item', 'general')),
    CONSTRAINT personalization_feedback_type_check CHECK (feedback_type IN ('like', 'dislike', 'too_expensive', 'too_much_walking', 'too_packed', 'not_my_vibe', 'more_nature', 'more_food', 'less_museums', 'prefer_trains', 'avoid_nightlife', 'prefer_relaxed', 'prefer_fast_paced', 'too_far', 'too_many_transfers', 'other'))
);

CREATE INDEX IF NOT EXISTS idx_personalization_feedback_user_created ON personalization_feedback (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_personalization_feedback_user_type ON personalization_feedback (user_id, feedback_type);
CREATE INDEX IF NOT EXISTS idx_personalization_feedback_trip ON personalization_feedback (trip_id);
CREATE INDEX IF NOT EXISTS idx_personalization_feedback_entity ON personalization_feedback (entity_type, entity_id);
