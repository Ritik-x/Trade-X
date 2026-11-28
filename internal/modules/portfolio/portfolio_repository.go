package portfolio

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type Position struct {
	InstrumentID    uuid.UUID
	Symbol          string
	CompanyName     string
	Exchange        string
	Quantity        int64
	AverageBuyPrice string
}

func (r *Repository) GetCashBalance(
	ctx context.Context,
	userID uuid.UUID,
) (string, error) {
	var balance string

	err := r.db.QueryRow(ctx, `
        SELECT cash_balance::TEXT
        FROM paper_trading_accounts
        WHERE user_id = $1
    `, userID).Scan(&balance)

	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("paper trading account not found")
		}
		return "", fmt.Errorf("get cash balance: %w", err)
	}

	return balance, nil
}

func (r *Repository) GetPositions(
	ctx context.Context,
	userID uuid.UUID,
) ([]Position, error) {
	rows, err := r.db.Query(ctx, `
        SELECT
            p.instrument_id,
            i.symbol,
            i.company_name,
            i.exchange,
            p.quantity,
            p.average_buy_price::TEXT
        FROM paper_positions p
        JOIN paper_trading_accounts a
            ON a.id = p.account_id
        JOIN instruments i
            ON i.id = p.instrument_id
        WHERE a.user_id = $1
        ORDER BY i.symbol
    `, userID)
	if err != nil {
		return nil, fmt.Errorf("get portfolio positions: %w", err)
	}
	defer rows.Close()

	positions := make([]Position, 0)

	for rows.Next() {
		var position Position

		err := rows.Scan(
			&position.InstrumentID,
			&position.Symbol,
			&position.CompanyName,
			&position.Exchange,
			&position.Quantity,
			&position.AverageBuyPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("scan portfolio position: %w", err)
		}

		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolio positions: %w", err)
	}

	return positions, nil
}
