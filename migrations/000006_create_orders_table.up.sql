CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    instrument_id UUID NOT NULL
        REFERENCES instruments(id),

    side VARCHAR(10) NOT NULL
        CHECK (side IN ('BUY', 'SELL')),

    order_type VARCHAR(10) NOT NULL DEFAULT 'MARKET'
        CHECK (order_type IN ('MARKET')),

    quantity BIGINT NOT NULL
        CHECK (quantity > 0),

    price_at_order NUMERIC(18,4) NOT NULL
        CHECK (price_at_order > 0),

    execution_price NUMERIC(18,4),
    total_amount NUMERIC(20,4),

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (
            status IN (
                'PENDING',
                'EXECUTED',
                'REJECTED',
                'CANCELLED'
            )
        ),

    rejection_reason TEXT,
    idempotency_key VARCHAR(100),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    executed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_orders_user_idempotency
    ON orders(user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX idx_orders_user_created
    ON orders(user_id, created_at DESC);

CREATE INDEX idx_orders_instrument_status
    ON orders(instrument_id, status);
