CREATE TABLE IF NOT EXISTS trip_expenses (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency TEXT NOT NULL,
    category TEXT NOT NULL,
    expense_date DATE NOT NULL,
    paid_by_user_id UUID NOT NULL,
    split_type TEXT NOT NULL,
    linked_day_number INT NULL,
    linked_item_index INT NULL,
    linked_item_id TEXT NULL,
    linked_route_leg_id TEXT NULL,
    linked_accommodation BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NULL,
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NULL,
    deleted_at TIMESTAMP NULL,
    deleted_by_user_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_expenses_title_not_empty CHECK (length(btrim(title)) > 0),
    CONSTRAINT trip_expenses_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT trip_expenses_currency_check CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT trip_expenses_category_check CHECK (category IN (
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
    )),
    CONSTRAINT trip_expenses_split_type_check CHECK (split_type IN (
        'equal',
        'selected_equal',
        'custom_amounts',
        'custom_percentages',
        'payer_only'
    )),
    CONSTRAINT trip_expenses_status_check CHECK (status IN ('active', 'deleted'))
);

CREATE TABLE IF NOT EXISTS trip_expense_participants (
    id UUID PRIMARY KEY,
    expense_id UUID NOT NULL REFERENCES trip_expenses(id) ON DELETE CASCADE,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    share_amount NUMERIC(12,2) NULL,
    share_currency TEXT NULL,
    share_percentage NUMERIC(7,4) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_expense_participants_unique_user UNIQUE (expense_id, user_id),
    CONSTRAINT trip_expense_participants_share_amount_non_negative CHECK (share_amount IS NULL OR share_amount >= 0),
    CONSTRAINT trip_expense_participants_share_currency_check CHECK (share_currency IS NULL OR share_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT trip_expense_participants_share_percentage_non_negative CHECK (share_percentage IS NULL OR share_percentage >= 0)
);

CREATE TABLE IF NOT EXISTS trip_settlements (
    id UUID PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    from_user_id UUID NOT NULL,
    to_user_id UUID NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    source TEXT NOT NULL DEFAULT 'calculated',
    calculation_hash TEXT NULL,
    paid_at TIMESTAMP NULL,
    paid_by_user_id UUID NULL,
    cancelled_at TIMESTAMP NULL,
    cancelled_by_user_id UUID NULL,
    notes TEXT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_settlements_amount_positive CHECK (amount > 0),
    CONSTRAINT trip_settlements_currency_check CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT trip_settlements_status_check CHECK (status IN ('pending', 'paid', 'cancelled')),
    CONSTRAINT trip_settlements_source_check CHECK (source IN ('calculated', 'manual')),
    CONSTRAINT trip_settlements_distinct_users CHECK (from_user_id <> to_user_id)
);

CREATE INDEX IF NOT EXISTS idx_trip_expenses_trip_id ON trip_expenses(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_expenses_paid_by_user_id ON trip_expenses(paid_by_user_id);
CREATE INDEX IF NOT EXISTS idx_trip_expenses_category ON trip_expenses(category);
CREATE INDEX IF NOT EXISTS idx_trip_expenses_expense_date ON trip_expenses(expense_date);
CREATE INDEX IF NOT EXISTS idx_trip_expenses_status ON trip_expenses(status);
CREATE INDEX IF NOT EXISTS idx_trip_expenses_deleted_at ON trip_expenses(deleted_at);

CREATE INDEX IF NOT EXISTS idx_trip_expense_participants_expense_id ON trip_expense_participants(expense_id);
CREATE INDEX IF NOT EXISTS idx_trip_expense_participants_trip_id ON trip_expense_participants(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_expense_participants_user_id ON trip_expense_participants(user_id);

CREATE INDEX IF NOT EXISTS idx_trip_settlements_trip_id ON trip_settlements(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_settlements_from_user_id ON trip_settlements(from_user_id);
CREATE INDEX IF NOT EXISTS idx_trip_settlements_to_user_id ON trip_settlements(to_user_id);
CREATE INDEX IF NOT EXISTS idx_trip_settlements_status ON trip_settlements(status);
CREATE INDEX IF NOT EXISTS idx_trip_settlements_calculation_hash ON trip_settlements(calculation_hash);
