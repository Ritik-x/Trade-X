package market

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"tradex/internal/cache"

	"github.com/google/uuid"
)

type Service struct {
	repo       *Repository
	quoteRepo  *QuoteRepository
	quoteCache *cache.QuoteCache
	nseClient  *NSEClient
}

func NewService(
	repo *Repository,
	quoteRepo *QuoteRepository,
	quoteCache *cache.QuoteCache,
	nseClient *NSEClient,

) *Service {
	return &Service{
		repo:       repo,
		quoteRepo:  quoteRepo,
		quoteCache: quoteCache,
		nseClient:  nseClient,
	}
}
func (s *Service) GetAll(ctx context.Context) ([]Instrument, error) {
	instruments, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all instruments: %w", err)
	}

	return instruments, nil
}

func (s *Service) GetBySymbol(
	ctx context.Context,
	exchange string,
	symbol string,
) (*Instrument, error) {
	exchange = strings.ToUpper(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if exchange != "NSE" && exchange != "BSE" {
		return nil, fmt.Errorf("unsupported exchange")
	}

	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	instrument, err := s.repo.GetBySymbol(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}

	return instrument, nil
}

func (s *Service) GetQuote(
	ctx context.Context,
	exchange string,
	symbol string,
) (*Quote, error) {
	exchange = strings.ToUpper(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if exchange != "NSE" && exchange != "BSE" {
		return nil, fmt.Errorf("unsupported exchange")
	}

	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	// 1. Find the instrument in PostgreSQL.
	instrument, err := s.repo.GetBySymbol(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}

	// 2. Check Redis cache.
	var cachedQuote Quote

	err = s.quoteCache.GetQuote(
		ctx,
		instrument.ID,
		&cachedQuote,
	)

	if err == nil {
		return &cachedQuote, nil
	}

	// 3. Cache miss or Redis failure: fetch from PostgreSQL.
	quote, err := s.quoteRepo.GetByInstrumentID(
		ctx,
		instrument.ID,
	)
	if err != nil {
		return nil, err
	}

	// 4. Save quote in Redis for future requests.
	// Cache failure should not fail the API request.
	_ = s.quoteCache.SetQuote(
		ctx,
		instrument.ID,
		quote,
	)

	return quote, nil
}

func (s *Service) FetchAndStoreQuote(
	ctx context.Context,
	exchange string,
	symbol string,
) (*Quote, error) {
	exchange = strings.ToUpper(strings.TrimSpace(exchange))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if exchange != "NSE" {
		return nil, fmt.Errorf("NSE provider supports NSE instruments only")
	}

	instrument, err := s.repo.GetBySymbol(ctx, exchange, symbol)
	if err != nil {
		return nil, err
	}

	// Fetch the latest available quote from NSE.
	nseQuote, err := s.nseClient.GetStockQuote(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("fetch NSE quote: %w", err)
	}

	if nseQuote.Stock.Symbol != symbol {
		return nil, fmt.Errorf("NSE returned a different stock symbol")
	}

	// NSE timestamps are in Indian Standard Time.
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return nil, fmt.Errorf("load IST timezone: %w", err)
	}

	latestTime, err := time.ParseInLocation(
		"2006-01-02 15:04:05",
		nseQuote.Stock.LatestTimestamp,
		location,
	)
	if err != nil {
		return nil, fmt.Errorf("parse NSE quote timestamp: %w", err)
	}

	now := time.Now().In(location)

	// Determine whether the quote is live, stale, or from a closed market.
	marketStatus := "CLOSED"

	if now.Weekday() >= time.Monday &&
		now.Weekday() <= time.Friday &&
		!now.Before(time.Date(
			now.Year(), now.Month(), now.Day(),
			9, 15, 0, 0, location,
		)) &&
		now.Before(time.Date(
			now.Year(), now.Month(), now.Day(),
			15, 30, 0, 0, location,
		)) {

		if now.Sub(latestTime) >= 0 &&
			now.Sub(latestTime) <= 2*time.Minute {
			marketStatus = "LIVE"
		} else {
			marketStatus = "STALE"
		}
	}

	// Convert source prices to decimal strings.
	lastPrice := strconv.FormatFloat(
		nseQuote.Stock.LastTradedPrice, 'f', 4, 64,
	)
	openPrice := strconv.FormatFloat(
		nseQuote.Stock.OpenPrice, 'f', 4, 64,
	)
	highPrice := strconv.FormatFloat(
		nseQuote.Stock.HighPrice, 'f', 4, 64,
	)
	lowPrice := strconv.FormatFloat(
		nseQuote.Stock.LowPrice, 'f', 4, 64,
	)
	previousClose := strconv.FormatFloat(
		nseQuote.Stock.PreClosePrice, 'f', 4, 64,
	)
	priceChange := strconv.FormatFloat(
		nseQuote.Stock.Change, 'f', 4, 64,
	)
	changePercent := strconv.FormatFloat(
		nseQuote.Stock.PerChange, 'f', 4, 64,
	)

	quote := &Quote{
		InstrumentID:  instrument.ID,
		LastPrice:     &lastPrice,
		OpenPrice:     &openPrice,
		HighPrice:     &highPrice,
		LowPrice:      &lowPrice,
		PreviousClose: &previousClose,
		Volume:        nseQuote.Stock.Volume,
		PriceChange:   &priceChange,
		ChangePercent: &changePercent,
		MarketStatus:  marketStatus,
		LastUpdatedAt: &latestTime,
	}

	// Persist the new quote in PostgreSQL.
	if err := s.quoteRepo.Upsert(ctx, instrument.ID, *quote); err != nil {
		return nil, err
	}

	// Refresh Redis after the database write succeeds.
	// Redis failure must not invalidate the successful database update.
	_ = s.quoteCache.SetQuote(ctx, instrument.ID, quote)

	return quote, nil
}

func (s *Service) GetByInstrumentID(
	ctx context.Context,
	instrumentID uuid.UUID,
) (*Quote, error) {
	if instrumentID == uuid.Nil {
		return nil, fmt.Errorf("invalid instrument ID")
	}

	var cachedQuote Quote

	err := s.quoteCache.GetQuote(ctx, instrumentID, &cachedQuote)
	if err == nil {
		return &cachedQuote, nil
	}

	quote, err := s.quoteRepo.GetByInstrumentID(ctx, instrumentID)
	if err != nil {
		return nil, err
	}

	_ = s.quoteCache.SetQuote(ctx, instrumentID, quote)

	return quote, nil
}
