
-- One virtual cash account per user.
CREATE TABLE paper_trading_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    cash_balance NUMERIC(20,4) NOT NULL DEFAULT 1000000.0000
        CHECK (cash_balance >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Current stock holdings for each paper-trading account.
CREATE TABLE paper_positions (
    account_id UUID NOT NULL
        REFERENCES paper_trading_accounts(id) ON DELETE CASCADE,
    instrument_id UUID NOT NULL
        REFERENCES instruments(id),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    average_buy_price NUMERIC(18,4) NOT NULL
        CHECK (average_buy_price > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, instrument_id)
);

-- Audit trail for every successfully executed order.
CREATE TABLE paper_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL
        REFERENCES paper_trading_accounts(id),
    order_id UUID NOT NULL UNIQUE
        REFERENCES orders(id),
    instrument_id UUID NOT NULL
        REFERENCES instruments(id),
    transaction_type VARCHAR(10) NOT NULL
        CHECK (transaction_type IN ('BUY', 'SELL')),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    execution_price NUMERIC(18,4) NOT NULL
        CHECK (execution_price > 0),
    total_amount NUMERIC(20,4) NOT NULL
        CHECK (total_amount > 0),
    cash_balance_after NUMERIC(20,4) NOT NULL
        CHECK (cash_balance_after >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_paper_transactions_account_created
    ON paper_transactions(account_id, created_at DESC);

CREATE INDEX idx_paper_positions_instrument
    ON paper_positions(instrument_id);
