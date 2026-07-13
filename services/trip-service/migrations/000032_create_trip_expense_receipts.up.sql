CREATE TABLE IF NOT EXISTS trip_expense_receipts (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    expense_id UUID NULL REFERENCES trip_expenses(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'uploaded',
    original_filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    file_sha256 TEXT NULL,
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NULL,
    deleted_at TIMESTAMP NULL,
    deleted_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_expense_receipts_status_check CHECK (status IN (
        'uploaded',
        'processing',
        'extracted',
        'extraction_failed',
        'attached',
        'deleted'
    )),
    CONSTRAINT trip_expense_receipts_original_filename_not_empty CHECK (length(btrim(original_filename)) > 0),
    CONSTRAINT trip_expense_receipts_content_type_check CHECK (content_type IN (
        'image/jpeg',
        'image/png',
        'image/webp',
        'application/pdf'
    )),
    CONSTRAINT trip_expense_receipts_size_positive CHECK (size_bytes > 0)
);

CREATE TABLE IF NOT EXISTS receipt_ocr_results (
    id UUID PRIMARY KEY,
    receipt_id UUID NOT NULL REFERENCES trip_expense_receipts(id) ON DELETE CASCADE,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT 'mock',
    status TEXT NOT NULL DEFAULT 'extracted',
    merchant TEXT NULL,
    expense_date DATE NULL,
    amount NUMERIC(12,2) NULL,
    currency TEXT NULL,
    tax_amount NUMERIC(12,2) NULL,
    category TEXT NULL,
    suggested_title TEXT NULL,
    confidence TEXT NOT NULL DEFAULT 'low',
    field_confidence_json JSONB NULL,
    warnings_json JSONB NOT NULL DEFAULT '[]',
    raw_text TEXT NULL,
    normalized_json JSONB NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT receipt_ocr_results_provider_check CHECK (provider IN ('mock', 'local', 'manual')),
    CONSTRAINT receipt_ocr_results_status_check CHECK (status IN ('extracted', 'extraction_failed')),
    CONSTRAINT receipt_ocr_results_confidence_check CHECK (confidence IN ('low', 'medium', 'high')),
    CONSTRAINT receipt_ocr_results_amount_non_negative CHECK (amount IS NULL OR amount >= 0),
    CONSTRAINT receipt_ocr_results_tax_amount_non_negative CHECK (tax_amount IS NULL OR tax_amount >= 0),
    CONSTRAINT receipt_ocr_results_currency_check CHECK (currency IS NULL OR currency ~ '^[A-Z]{3}$'),
    CONSTRAINT receipt_ocr_results_category_check CHECK (category IS NULL OR category IN (
        'transport',
        'accommodation',
        'food',
        'tickets',
        'activities',
        'shopping',
        'fuel',
        'parking',
        'tolls',
        'camping',
        'groceries',
        'health_safety',
        'other'
    ))
);

CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_trip_id ON trip_expense_receipts(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_expense_id ON trip_expense_receipts(expense_id);
CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_status ON trip_expense_receipts(status);
CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_created_by_user_id ON trip_expense_receipts(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_deleted_at ON trip_expense_receipts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_trip_expense_receipts_file_sha256 ON trip_expense_receipts(file_sha256);

CREATE INDEX IF NOT EXISTS idx_receipt_ocr_results_receipt_id ON receipt_ocr_results(receipt_id);
CREATE INDEX IF NOT EXISTS idx_receipt_ocr_results_trip_id ON receipt_ocr_results(trip_id);
CREATE INDEX IF NOT EXISTS idx_receipt_ocr_results_provider ON receipt_ocr_results(provider);
CREATE INDEX IF NOT EXISTS idx_receipt_ocr_results_status ON receipt_ocr_results(status);
