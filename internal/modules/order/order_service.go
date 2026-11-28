package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tradex/internal/modules/market"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OrderService struct {
	repo          *OrderRepository
	paperRepo     *PaperRepository
	marketService *market.Service
}

func NewOrderService(
	repo *OrderRepository,
	paperRepo *PaperRepository,
	marketService *market.Service,
) *OrderService {
	return &OrderService{
		repo:          repo,
		paperRepo:     paperRepo,
		marketService: marketService,
	}
}

func (s *OrderService) PlaceOrder(
	ctx context.Context,
	userID uuid.UUID,
	req PlaceOrderRequest,
) (*Order, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	if req.InstrumentID == uuid.Nil {
		return nil, fmt.Errorf("instrument ID is required")
	}

	req.Side = strings.ToUpper(strings.TrimSpace(req.Side))

	if req.Side != "BUY" && req.Side != "SELL" {
		return nil, fmt.Errorf("side must be BUY or SELL")
	}

	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than zero")
	}

	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)

	if req.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency key is required")
	}

	if len(req.IdempotencyKey) > 100 {
		return nil, fmt.Errorf("idempotency key cannot exceed 100 characters")
	}

	// 1. Check whether this request was already processed.
	existingOrder, err := s.repo.GetByIdempotencyKey(
		ctx,
		userID,
		req.IdempotencyKey,
	)

	if err == nil {
		// A retry returns the original order.
		if existingOrder.InstrumentID != req.InstrumentID ||
			existingOrder.Side != req.Side ||
			existingOrder.Quantity != req.Quantity {
			return nil, fmt.Errorf("idempotency key reused with different order details")
		}

		return existingOrder, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check existing order: %w", err)
	}

	// 2. Find the instrument and its current market quote.
	instrument, err := s.marketService.GetByInstrumentID(
		ctx,
		req.InstrumentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get instrument quote: %w", err)
	}

	if instrument.LastPrice == nil || *instrument.LastPrice == "" {
		return nil, fmt.Errorf("market price is unavailable")
	}

	// 3. Only accept fresh market quotes for simulated execution.
	if instrument.MarketStatus != "LIVE" {
		return nil, fmt.Errorf("market quote is not live")
	}

	// 4. Create a pending order.
	order := &Order{
		UserID:         userID,
		InstrumentID:   req.InstrumentID,
		Side:           req.Side,
		OrderType:      "MARKET",
		Quantity:       req.Quantity,
		PriceAtOrder:   *instrument.LastPrice,
		Status:         "PENDING",
		IdempotencyKey: &req.IdempotencyKey,
	}

	createdOrder, err := s.repo.Create(ctx, order)

	if errors.Is(err, pgx.ErrNoRows) {
		// Another concurrent request inserted this idempotency key.
		existingOrder, getErr := s.repo.GetByIdempotencyKey(
			ctx,
			userID,
			req.IdempotencyKey,
		)
		if getErr != nil {
			return nil, fmt.Errorf("get existing order after conflict: %w", getErr)
		}

		// Make sure the same key wasn't reused for different order details.
		if existingOrder.InstrumentID != req.InstrumentID ||
			existingOrder.Side != req.Side ||
			existingOrder.Quantity != req.Quantity {
			return nil, fmt.Errorf("idempotency key reused with different order details")
		}

		return existingOrder, nil
	}

	if err != nil {
		return nil, fmt.Errorf("place order: %w", err)
	}

	return createdOrder, nil
}

func (s *OrderService) ExecuteOrder(
	ctx context.Context,
	userID uuid.UUID,
	orderID uuid.UUID,
) (*Order, error) {
	if userID == uuid.Nil || orderID == uuid.Nil {
		return nil, fmt.Errorf("invalid user or order ID")
	}

	// Always load the original order from the database.
	order, err := s.repo.GetByID(ctx, userID, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	if order.Status != "PENDING" {
		return nil, fmt.Errorf("order is not pending")
	}

	// Ensure the user has a virtual cash account.
	_, err = s.paperRepo.CreateAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Fetch the latest market quote.
	quote, err := s.marketService.GetByInstrumentID(
		ctx,
		order.InstrumentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get market quote: %w", err)
	}

	if quote.LastPrice == nil || strings.TrimSpace(*quote.LastPrice) == "" {
		return nil, fmt.Errorf("market price unavailable")
	}

	if quote.MarketStatus != "LIVE" {
		return nil, fmt.Errorf("market quote is not live")
	}

	// Execute using trusted values from the saved order and market quote.
	err = s.paperRepo.ExecuteOrder(ctx, ExecuteOrderParams{
		OrderID:        order.ID,
		UserID:         order.UserID,
		InstrumentID:   order.InstrumentID,
		Side:           order.Side,
		Quantity:       order.Quantity,
		ExecutionPrice: *quote.LastPrice,
	})
	if err != nil {
		return nil, err
	}

	// Return the updated order.
	return s.repo.GetByID(ctx, userID, orderID)
}

func (s *OrderService) GetPaperAccount(
	ctx context.Context,
	userID uuid.UUID,
) (*PaperAccount, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	account, err := s.paperRepo.CreateAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *OrderService) GetPortfolio(
	ctx context.Context,
	userID uuid.UUID,
) ([]PaperPosition, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	_, err := s.paperRepo.CreateAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.paperRepo.GetPositions(ctx, userID)
}
func (s *OrderService) GetOrderHistory(
	ctx context.Context,
	userID uuid.UUID,
) ([]Order, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	return s.repo.GetByUserID(ctx, userID)
}

func (s *OrderService) CancelOrder(
	ctx context.Context,
	userID uuid.UUID,
	orderID uuid.UUID,
) (*Order, error) {
	if userID == uuid.Nil || orderID == uuid.Nil {
		return nil, fmt.Errorf("invalid user or order ID")
	}

	return s.repo.CancelOrder(ctx, userID, orderID)
}
