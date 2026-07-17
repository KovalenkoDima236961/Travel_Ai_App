-- Composite/partial indexes for the hot trip-detail, receipt, and ops paths.
-- These match actual WHERE + ORDER BY shapes and avoid indexing deleted rows.
CREATE INDEX IF NOT EXISTS idx_trip_expenses_trip_active_sort
    ON trip_expenses (trip_id, expense_date DESC, created_at DESC, id DESC)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_trip_active_sort
    ON trip_expense_receipts (trip_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND status <> 'deleted';

CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_expense_active_sort
    ON trip_expense_receipts (expense_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND status <> 'deleted';

CREATE INDEX IF NOT EXISTS idx_receipt_ocr_latest
    ON receipt_ocr_results (trip_id, receipt_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_trip_checklist_items_trip_open_assignee
    ON trip_checklist_items (trip_id, assigned_to_user_id, due_date, sort_order)
    WHERE deleted_at IS NULL AND checked = FALSE;

CREATE INDEX IF NOT EXISTS idx_trip_reminders_trip_status_trigger
    ON trip_reminders (trip_id, status, trigger_date, trigger_time)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_status_created
    ON ai_generation_traces (status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_ai_generation_traces_type_created
    ON ai_generation_traces (generation_type, created_at DESC, id DESC);

