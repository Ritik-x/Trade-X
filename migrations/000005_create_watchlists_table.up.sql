
CREATE TABLE watchlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    name VARCHAR(100) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_watchlists_user_name
    ON watchlists (user_id, LOWER(name));

CREATE INDEX idx_watchlists_user_id
    ON watchlists (user_id);


CREATE TABLE watchlist_items (
    watchlist_id UUID NOT NULL
        REFERENCES watchlists(id) ON DELETE CASCADE,

    instrument_id UUID NOT NULL
        REFERENCES instruments(id) ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (watchlist_id, instrument_id)
);

CREATE INDEX idx_watchlist_items_instrument_id
    ON watchlist_items (instrument_id);
