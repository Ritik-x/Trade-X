package order

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	InstrumentID    uuid.UUID  `json:"instrument_id"`
	Side            string     `json:"side"`
	OrderType       string     `json:"order_type"`
	Quantity        int64      `json:"quantity"`
	PriceAtOrder    string     `json:"price_at_order"`
	ExecutionPrice  *string    `json:"execution_price"`
	TotalAmount     *string    `json:"total_amount"`
	Status          string     `json:"status"`
	RejectionReason *string    `json:"rejection_reason"`
	IdempotencyKey  *string    `json:"idempotency_key"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ExecutedAt      *time.Time `json:"executed_at"`
}

type PlaceOrderRequest struct {
	InstrumentID   uuid.UUID `json:"instrument_id" binding:"required"`
	Side           string    `json:"side" binding:"required"`
	Quantity       int64     `json:"quantity" binding:"required"`
	IdempotencyKey string    `json:"-"`
}
