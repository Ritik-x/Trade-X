CREATE TABLE market_quotes (
    instrument_id UUID PRIMARY KEY
        REFERENCES instruments(id) ON DELETE CASCADE,

    last_price NUMERIC(18, 4),
    open_price NUMERIC(18, 4),
    high_price NUMERIC(18, 4),
    low_price NUMERIC(18, 4),
    previous_close NUMERIC(18, 4),

    volume BIGINT NOT NULL DEFAULT 0,

    price_change NUMERIC(18, 4),
    change_percent NUMERIC(10, 4),

    market_status VARCHAR(20) NOT NULL DEFAULT 'UNAVAILABLE',

    last_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT market_quotes_status_check
        CHECK (
            market_status IN (
                'LIVE',
                'DELAYED',
                'STALE',
                'CLOSED',
                'UNAVAILABLE'
            )
        ),

    CONSTRAINT market_quotes_volume_check
        CHECK (volume >= 0)
);