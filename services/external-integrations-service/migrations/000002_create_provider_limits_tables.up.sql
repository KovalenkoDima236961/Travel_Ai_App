-- Per provider + operation daily usage counters. Rows are per usage day so the
-- Ops Dashboard can show detailed operation-level usage.
CREATE TABLE IF NOT EXISTS provider_daily_usage (
    id UUID PRIMARY KEY,
    provider TEXT NOT NULL,
    operation TEXT NOT NULL,
    usage_date DATE NOT NULL,
    used_count BIGINT NOT NULL DEFAULT 0,
    blocked_count BIGINT NOT NULL DEFAULT 0,
    fallback_count BIGINT NOT NULL DEFAULT 0,
    last_allowed_at TIMESTAMP NULL,
    last_blocked_at TIMESTAMP NULL,
    last_fallback_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT provider_daily_usage_used_count_check CHECK (used_count >= 0),
    CONSTRAINT provider_daily_usage_blocked_count_check CHECK (blocked_count >= 0),
    CONSTRAINT provider_daily_usage_fallback_count_check CHECK (fallback_count >= 0),
    CONSTRAINT provider_daily_usage_unique UNIQUE (provider, operation, usage_date)
);

CREATE INDEX IF NOT EXISTS provider_daily_usage_date_idx ON provider_daily_usage(usage_date);
CREATE INDEX IF NOT EXISTS provider_daily_usage_provider_date_idx ON provider_daily_usage(provider, usage_date);
CREATE INDEX IF NOT EXISTS provider_daily_usage_provider_operation_date_idx ON provider_daily_usage(provider, operation, usage_date);

-- Per provider daily totals. This table is the aggregate across all of a
-- provider's operations and is used to serialize atomic quota reservations
-- (SELECT ... FOR UPDATE) so concurrent reservations cannot exceed the quota.
CREATE TABLE IF NOT EXISTS provider_daily_totals (
    id UUID PRIMARY KEY,
    provider TEXT NOT NULL,
    usage_date DATE NOT NULL,
    used_count BIGINT NOT NULL DEFAULT 0,
    blocked_count BIGINT NOT NULL DEFAULT 0,
    fallback_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT provider_daily_totals_used_count_check CHECK (used_count >= 0),
    CONSTRAINT provider_daily_totals_blocked_count_check CHECK (blocked_count >= 0),
    CONSTRAINT provider_daily_totals_fallback_count_check CHECK (fallback_count >= 0),
    CONSTRAINT provider_daily_totals_unique UNIQUE (provider, usage_date)
);

CREATE INDEX IF NOT EXISTS provider_daily_totals_date_idx ON provider_daily_totals(usage_date);
CREATE INDEX IF NOT EXISTS provider_daily_totals_provider_date_idx ON provider_daily_totals(provider, usage_date);
