package portfolio

import (
	"context"
	"fmt"
	"math/big"

	"tradex/internal/modules/market"

	"github.com/google/uuid"
)

type Service struct {
	repo          *Repository
	marketService *market.Service
}

func NewService(repo *Repository, marketService *market.Service) *Service {
	return &Service{
		repo:          repo,
		marketService: marketService,
	}
}

func parseAmount(value string) (*big.Rat, error) {
	amount, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("invalid monetary value: %s", value)
	}
	return amount, nil
}

func formatAmount(value *big.Rat) string {
	return value.FloatString(4)
}

func (s *Service) GetSummary(
	ctx context.Context,
	userID uuid.UUID,
) (*PortfolioSummary, error) {
	cashBalance, err := s.repo.GetCashBalance(ctx, userID)
	if err != nil {
		return nil, err
	}

	cash, err := parseAmount(cashBalance)
	if err != nil {
		return nil, err
	}

	positions, err := s.repo.GetPositions(ctx, userID)
	if err != nil {
		return nil, err
	}

	totalInvested := new(big.Rat)
	totalCurrent := new(big.Rat)

	result := &PortfolioSummary{
		CashBalance: formatAmount(cash),
		Positions:   make([]PortfolioPosition, 0, len(positions)),
	}

	for _, position := range positions {
		quote, err := s.marketService.GetByInstrumentID(
			ctx,
			position.InstrumentID,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"get quote for %s: %w",
				position.Symbol,
				err,
			)
		}

		if quote == nil || quote.LastPrice == nil {
			return nil, fmt.Errorf(
				"market price unavailable for %s",
				position.Symbol,
			)
		}

		currentPrice, err := parseAmount(*quote.LastPrice)
		if err != nil {
			return nil, err
		}

		averageBuyPrice, err := parseAmount(position.AverageBuyPrice)
		if err != nil {
			return nil, err
		}

		quantity := big.NewRat(position.Quantity, 1)

		marketValue := new(big.Rat).Mul(currentPrice, quantity)
		investedValue := new(big.Rat).Mul(averageBuyPrice, quantity)
		unrealizedPnL := new(big.Rat).Sub(marketValue, investedValue)

		totalCurrent.Add(totalCurrent, marketValue)
		totalInvested.Add(totalInvested, investedValue)

		result.Positions = append(result.Positions, PortfolioPosition{
			InstrumentID:    position.InstrumentID,
			Symbol:          position.Symbol,
			CompanyName:     position.CompanyName,
			Exchange:        position.Exchange,
			Quantity:        position.Quantity,
			AverageBuyPrice: position.AverageBuyPrice,
			CurrentPrice:    *quote.LastPrice,
			MarketValue:     formatAmount(marketValue),
			InvestedValue:   formatAmount(investedValue),
			UnrealizedPnL:   formatAmount(unrealizedPnL),
			MarketStatus:    quote.MarketStatus,
		})
	}

	totalPortfolio := new(big.Rat).Add(cash, totalCurrent)
	totalPnL := new(big.Rat).Sub(totalCurrent, totalInvested)

	result.InvestedValue = formatAmount(totalInvested)
	result.CurrentValue = formatAmount(totalCurrent)
	result.TotalPortfolio = formatAmount(totalPortfolio)
	result.UnrealizedPnL = formatAmount(totalPnL)

	return result, nil
}
