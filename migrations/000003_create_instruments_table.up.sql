CREATE TABLE instruments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    symbol VARCHAR(30) NOT NULL,
    company_name VARCHAR(150) NOT NULL,

    exchange VARCHAR(10) NOT NULL,
    instrument_type VARCHAR(20) NOT NULL DEFAULT 'EQUITY',

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT instruments_exchange_check
        CHECK (exchange IN ('NSE', 'BSE')),

    CONSTRAINT instruments_type_check
        CHECK (instrument_type IN ('EQUITY'))
);

CREATE UNIQUE INDEX idx_instruments_exchange_symbol
    ON instruments (exchange, symbol);