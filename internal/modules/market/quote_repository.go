package market

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuoteRepository struct {
	db *pgxpool.Pool
}

func NewQuoteRepository(db *pgxpool.Pool) *QuoteRepository {
	return &QuoteRepository{db: db}
}

func (r *QuoteRepository) GetByInstrumentID(
	ctx context.Context,
	instrumentID uuid.UUID,
) (*Quote, error) {
	query := `
		SELECT
			instrument_id,
			last_price::TEXT,
			open_price::TEXT,
			high_price::TEXT,
			low_price::TEXT,
			previous_close::TEXT,
			volume,
			price_change::TEXT,
			change_percent::TEXT,
			market_status,
			last_updated_at
		FROM market_quotes
		WHERE instrument_id = $1
	`

	var quote Quote

	err := r.db.QueryRow(ctx, query, instrumentID).Scan(
		&quote.InstrumentID,
		&quote.LastPrice,
		&quote.OpenPrice,
		&quote.HighPrice,
		&quote.LowPrice,
		&quote.PreviousClose,
		&quote.Volume,
		&quote.PriceChange,
		&quote.ChangePercent,
		&quote.MarketStatus,
		&quote.LastUpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("quote not found")
	}

	if err != nil {
		return nil, fmt.Errorf("get market quote: %w", err)
	}

	return &quote, nil
}

func (r *QuoteRepository) Upsert(
	ctx context.Context,
	instrumentID uuid.UUID,
	quote Quote,
) error {
	query := `
		INSERT INTO market_quotes (
			instrument_id,
			last_price,
			open_price,
			high_price,
			low_price,
			previous_close,
			volume,
			price_change,
			change_percent,
			market_status,
			last_updated_at,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, NOW()
		)
		ON CONFLICT (instrument_id)
		DO UPDATE SET
			last_price = EXCLUDED.last_price,
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			previous_close = EXCLUDED.previous_close,
			volume = EXCLUDED.volume,
			price_change = EXCLUDED.price_change,
			change_percent = EXCLUDED.change_percent,
			market_status = EXCLUDED.market_status,
			last_updated_at = EXCLUDED.last_updated_at,
			updated_at = NOW()
	`

	_, err := r.db.Exec(
		ctx,
		query,
		instrumentID,
		quote.LastPrice,
		quote.OpenPrice,
		quote.HighPrice,
		quote.LowPrice,
		quote.PreviousClose,
		quote.Volume,
		quote.PriceChange,
		quote.ChangePercent,
		quote.MarketStatus,
		quote.LastUpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("upsert market quote: %w", err)
	}

	return nil
}
