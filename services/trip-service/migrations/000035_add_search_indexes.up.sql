CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_trips_destination_trgm
    ON trips USING GIN (destination gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_trip_expenses_title_trgm
    ON trip_expenses USING GIN (title gin_trgm_ops)
    WHERE status = 'active' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_trip_expenses_description_trgm
    ON trip_expenses USING GIN (description gin_trgm_ops)
    WHERE status = 'active' AND deleted_at IS NULL AND description IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_filename_trgm
    ON trip_expense_receipts USING GIN (original_filename gin_trgm_ops)
    WHERE deleted_at IS NULL AND status <> 'deleted';

CREATE INDEX IF NOT EXISTS idx_receipt_ocr_results_merchant_trgm
    ON receipt_ocr_results USING GIN (merchant gin_trgm_ops)
    WHERE merchant IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_receipt_ocr_results_suggested_title_trgm
    ON receipt_ocr_results USING GIN (suggested_title gin_trgm_ops)
    WHERE suggested_title IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_title_trgm
    ON trip_checklist_items USING GIN (title gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_description_trgm
    ON trip_checklist_items USING GIN (description gin_trgm_ops)
    WHERE deleted_at IS NULL AND description IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trip_reminders_title_trgm
    ON trip_reminders USING GIN (title gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_trip_reminders_description_trgm
    ON trip_reminders USING GIN (description gin_trgm_ops)
    WHERE deleted_at IS NULL AND description IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trip_polls_title_trgm
    ON trip_polls USING GIN (title gin_trgm_ops)
    WHERE status <> 'archived';

CREATE INDEX IF NOT EXISTS idx_trip_polls_description_trgm
    ON trip_polls USING GIN (description gin_trgm_ops)
    WHERE status <> 'archived' AND description IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trip_templates_title_trgm
    ON trip_templates USING GIN (title gin_trgm_ops)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_trip_templates_description_trgm
    ON trip_templates USING GIN (description gin_trgm_ops)
    WHERE status = 'active' AND description IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trip_templates_destination_hint_trgm
    ON trip_templates USING GIN (destination_hint gin_trgm_ops)
    WHERE status = 'active' AND destination_hint IS NOT NULL;
