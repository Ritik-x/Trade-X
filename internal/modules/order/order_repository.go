package order

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

func scanOrder(row pgx.Row) (*Order, error) {
	var order Order

	err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.InstrumentID,
		&order.Side,
		&order.OrderType,
		&order.Quantity,
		&order.PriceAtOrder,
		&order.ExecutionPrice,
		&order.TotalAmount,
		&order.Status,
		&order.RejectionReason,
		&order.IdempotencyKey,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ExecutedAt,
	)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

const orderColumns = `
	id,
	user_id,
	instrument_id,
	side,
	order_type,
	quantity,
	price_at_order::TEXT,
	execution_price::TEXT,
	total_amount::TEXT,
	status,
	rejection_reason,
	idempotency_key,
	created_at,
	updated_at,
	executed_at
`

// Create inserts a new order.
func (r *OrderRepository) Create(
	ctx context.Context,
	order *Order,
) (*Order, error) {
	query := `
		INSERT INTO orders (
			user_id,
			instrument_id,
			side,
			order_type,
			quantity,
			price_at_order,
			status,
			idempotency_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'PENDING', $7)
		RETURNING ` + orderColumns

	createdOrder, err := scanOrder(
		r.db.QueryRow(
			ctx,
			query,
			order.UserID,
			order.InstrumentID,
			order.Side,
			order.OrderType,
			order.Quantity,
			order.PriceAtOrder,
			order.IdempotencyKey,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	return createdOrder, nil
}

// GetByID retrieves an order belonging to a specific user.
func (r *OrderRepository) GetByID(
	ctx context.Context,
	userID uuid.UUID,
	orderID uuid.UUID,
) (*Order, error) {
	query := `
		SELECT ` + orderColumns + `
		FROM orders
		WHERE id = $1 AND user_id = $2
	`

	order, err := scanOrder(
		r.db.QueryRow(ctx, query, orderID, userID),
	)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	return order, nil
}

// GetByIdempotencyKey retrieves an existing order for a retry.
func (r *OrderRepository) GetByIdempotencyKey(
	ctx context.Context,
	userID uuid.UUID,
	key string,
) (*Order, error) {
	query := `
		SELECT ` + orderColumns + `
		FROM orders
		WHERE user_id = $1 AND idempotency_key = $2
	`

	order, err := scanOrder(
		r.db.QueryRow(ctx, query, userID, key),
	)
	if err != nil {
		return nil, fmt.Errorf("get order by idempotency key: %w", err)
	}

	return order, nil
}

// GetByUserID retrieves a user's order history.
func (r *OrderRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]Order, error) {
	query := `
		SELECT ` + orderColumns + `
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user orders: %w", err)
	}
	defer rows.Close()

	orders := make([]Order, 0)

	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}

		orders = append(orders, *order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) CancelOrder(
	ctx context.Context,
	userID uuid.UUID,
	orderID uuid.UUID,
) (*Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin cancel order transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string

	err = tx.QueryRow(ctx, `
		SELECT status
		FROM orders
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, orderID, userID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("get order status: %w", err)
	}

	if status != "PENDING" {
		return nil, fmt.Errorf("only pending orders can be cancelled")
	}

	cancelledOrder, err := scanOrder(tx.QueryRow(ctx, `
		UPDATE orders
		SET status = 'CANCELLED',
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING `+orderColumns,
		orderID, userID,
	))
	if err != nil {
		return nil, fmt.Errorf("cancel order: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cancellation: %w", err)
	}

	return cancelledOrder, nil
}
