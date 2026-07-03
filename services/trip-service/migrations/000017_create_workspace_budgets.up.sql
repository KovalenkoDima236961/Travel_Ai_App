CREATE TABLE IF NOT EXISTS workspace_budgets (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    amount NUMERIC(12,2) NOT NULL,
    currency TEXT NOT NULL,
    period_start DATE,
    period_end DATE,
    status TEXT NOT NULL DEFAULT 'active',
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_by_user_id UUID NOT NULL,
    archived_by_user_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMP,
    CONSTRAINT workspace_budgets_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT workspace_budgets_currency_format CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT workspace_budgets_status_valid CHECK (status IN ('active', 'archived')),
    CONSTRAINT workspace_budgets_period_valid CHECK (
        period_start IS NULL OR period_end IS NULL OR period_start <= period_end
    )
);

CREATE INDEX IF NOT EXISTS workspace_budgets_workspace_idx
    ON workspace_budgets (workspace_id);

CREATE INDEX IF NOT EXISTS workspace_budgets_workspace_status_idx
    ON workspace_budgets (workspace_id, status);

CREATE INDEX IF NOT EXISTS workspace_budgets_workspace_primary_idx
    ON workspace_budgets (workspace_id, is_primary);

CREATE INDEX IF NOT EXISTS workspace_budgets_period_idx
    ON workspace_budgets (workspace_id, period_start, period_end);

CREATE UNIQUE INDEX IF NOT EXISTS workspace_budgets_one_primary_active
    ON workspace_budgets (workspace_id)
    WHERE status = 'active' AND is_primary = true;
