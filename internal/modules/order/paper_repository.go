package order

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaperRepository struct {
	db *pgxpool.Pool
}

func NewPaperRepository(db *pgxpool.Pool) *PaperRepository {
	return &PaperRepository{db: db}
}

type PaperAccount struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	CashBalance string    `json:"cash_balance"`
	Currency    string    `json:"currency"`
}
type ExecuteOrderParams struct {
	OrderID        uuid.UUID
	UserID         uuid.UUID
	InstrumentID   uuid.UUID
	Side           string
	Quantity       int64
	ExecutionPrice string
}
type PaperPosition struct {
	InstrumentID    uuid.UUID `json:"instrument_id"`
	Symbol          string    `json:"symbol"`
	CompanyName     string    `json:"company_name"`
	Exchange        string    `json:"exchange"`
	Quantity        int64     `json:"quantity"`
	AverageBuyPrice string    `json:"average_buy_price"`
}

func (r *PaperRepository) CreateAccount(
	ctx context.Context,
	userID uuid.UUID,
) (*PaperAccount, error) {
	var account PaperAccount

	err := r.db.QueryRow(ctx, `
		INSERT INTO paper_trading_accounts (user_id)
		VALUES ($1)
		ON CONFLICT (user_id)
		DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id, user_id, cash_balance::TEXT, currency
	`, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.CashBalance,
		&account.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("create paper account: %w", err)
	}

	return &account, nil
}

func (r *PaperRepository) GetAccount(
	ctx context.Context,
	userID uuid.UUID,
) (*PaperAccount, error) {
	var account PaperAccount

	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, cash_balance::TEXT, currency
		FROM paper_trading_accounts
		WHERE user_id = $1
	`, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.CashBalance,
		&account.Currency,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get paper account: %w", err)
	}

	return &account, nil
}

func (r *PaperRepository) ExecuteOrder(
	ctx context.Context,
	params ExecuteOrderParams,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin paper execution: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock the order so it cannot be executed twice concurrently.
	var status string

	err = tx.QueryRow(ctx, `
		SELECT status
		FROM orders
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, params.OrderID, params.UserID).Scan(&status)
	if err != nil {
		return fmt.Errorf("find order: %w", err)
	}

	if status != "PENDING" {
		return fmt.Errorf("order is not pending")
	}

	// Lock the user's paper account.
	var accountID uuid.UUID
	var cashBalance string

	err = tx.QueryRow(ctx, `
		SELECT id, cash_balance::TEXT
		FROM paper_trading_accounts
		WHERE user_id = $1
		FOR UPDATE
	`, params.UserID).Scan(&accountID, &cashBalance)
	if err != nil {
		return fmt.Errorf("paper account not found: %w", err)
	}

	var totalAmount string

	err = tx.QueryRow(ctx, `
		SELECT ($1::NUMERIC * $2::NUMERIC)::TEXT
	`, params.ExecutionPrice, params.Quantity).Scan(&totalAmount)
	if err != nil {
		return fmt.Errorf("calculate order amount: %w", err)
	}

	if params.Side == "BUY" {
		var updatedBalance string

		err = tx.QueryRow(ctx, `
			UPDATE paper_trading_accounts
			SET cash_balance = cash_balance - $1::NUMERIC,
			    updated_at = NOW()
			WHERE id = $2
			  AND cash_balance >= $1::NUMERIC
			RETURNING cash_balance::TEXT
		`, totalAmount, accountID).Scan(&updatedBalance)
		if err != nil {
			return fmt.Errorf("insufficient paper cash: %w", err)
		}

		// Update the position, calculating weighted average buy price.
		_, err = tx.Exec(ctx, `
			INSERT INTO paper_positions (
				account_id, instrument_id, quantity, average_buy_price
			)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (account_id, instrument_id)
			DO UPDATE SET
				average_buy_price = (
					paper_positions.quantity * paper_positions.average_buy_price
					+ EXCLUDED.quantity * EXCLUDED.average_buy_price
				) / (paper_positions.quantity + EXCLUDED.quantity),
				quantity = paper_positions.quantity + EXCLUDED.quantity,
				updated_at = NOW()
		`, accountID, params.InstrumentID, params.Quantity, params.ExecutionPrice)
		if err != nil {
			return fmt.Errorf("update paper position: %w", err)
		}

		cashBalance = updatedBalance

	} else if params.Side == "SELL" {
		var remainingQuantity int64

		err = tx.QueryRow(ctx, `
			UPDATE paper_positions
			SET quantity = quantity - $1,
			    updated_at = NOW()
			WHERE account_id = $2
			  AND instrument_id = $3
			  AND quantity >= $1
			RETURNING quantity
		`, params.Quantity, accountID, params.InstrumentID).Scan(&remainingQuantity)
		if err != nil {
			return fmt.Errorf("insufficient stock holdings: %w", err)
		}

		if remainingQuantity == 0 {
			_, err = tx.Exec(ctx, `
				DELETE FROM paper_positions
				WHERE account_id = $1 AND instrument_id = $2
			`, accountID, params.InstrumentID)
			if err != nil {
				return fmt.Errorf("remove empty position: %w", err)
			}
		}

		err = tx.QueryRow(ctx, `
			UPDATE paper_trading_accounts
			SET cash_balance = cash_balance + $1::NUMERIC,
			    updated_at = NOW()
			WHERE id = $2
			RETURNING cash_balance::TEXT
		`, totalAmount, accountID).Scan(&cashBalance)
		if err != nil {
			return fmt.Errorf("credit paper cash: %w", err)
		}

	} else {
		return fmt.Errorf("invalid order side")
	}

	// Record the successful execution.
	_, err = tx.Exec(ctx, `
		INSERT INTO paper_transactions (
			account_id, order_id, instrument_id, transaction_type,
			quantity, execution_price, total_amount, cash_balance_after
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, accountID, params.OrderID, params.InstrumentID, params.Side,
		params.Quantity, params.ExecutionPrice, totalAmount, cashBalance)
	if err != nil {
		return fmt.Errorf("record paper transaction: %w", err)
	}

	// Mark the order as executed.
	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status = 'EXECUTED',
		    execution_price = $1,
		    total_amount = $2,
		    executed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3
	`, params.ExecutionPrice, totalAmount, params.OrderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PaperRepository) GetPositions(
	ctx context.Context,
	userID uuid.UUID,
) ([]PaperPosition, error) {
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
		return nil, fmt.Errorf("get paper positions: %w", err)
	}
	defer rows.Close()

	positions := make([]PaperPosition, 0)

	for rows.Next() {
		var position PaperPosition

		err := rows.Scan(
			&position.InstrumentID,
			&position.Symbol,
			&position.CompanyName,
			&position.Exchange,
			&position.Quantity,
			&position.AverageBuyPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("scan paper position: %w", err)
		}

		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate paper positions: %w", err)
	}

	return positions, nil
}
