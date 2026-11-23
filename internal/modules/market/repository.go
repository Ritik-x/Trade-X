package market

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetAll(ctx context.Context) ([]Instrument, error) {
	query := `
		SELECT id, symbol, company_name, exchange,
		       instrument_type, is_active, created_at, updated_at
		FROM instruments
		WHERE is_active = TRUE
		ORDER BY symbol ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get instruments: %w", err)
	}
	defer rows.Close()

	instruments := make([]Instrument, 0)

	for rows.Next() {
		var instrument Instrument

		err := rows.Scan(
			&instrument.ID,
			&instrument.Symbol,
			&instrument.CompanyName,
			&instrument.Exchange,
			&instrument.InstrumentType,
			&instrument.IsActive,
			&instrument.CreatedAt,
			&instrument.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan instrument: %w", err)
		}

		instruments = append(instruments, instrument)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate instruments: %w", err)
	}

	return instruments, nil
}

func (r *Repository) GetBySymbol(
	ctx context.Context,
	exchange string,
	symbol string,
) (*Instrument, error) {
	query := `
		SELECT id, symbol, company_name, exchange,
		       instrument_type, is_active, created_at, updated_at
		FROM instruments
		WHERE exchange = $1 AND symbol = $2 AND is_active = TRUE
	`

	var instrument Instrument

	err := r.db.QueryRow(ctx, query, exchange, symbol).Scan(
		&instrument.ID,
		&instrument.Symbol,
		&instrument.CompanyName,
		&instrument.Exchange,
		&instrument.InstrumentType,
		&instrument.IsActive,
		&instrument.CreatedAt,
		&instrument.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("instrument not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get instrument: %w", err)
	}

	return &instrument, nil
}
