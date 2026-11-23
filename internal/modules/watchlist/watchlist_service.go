package watchlist

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type WatchlistService struct {
	repo *WatchlistRepository
}

func NewWatchlistService(repo *WatchlistRepository) *WatchlistService {
	return &WatchlistService{
		repo: repo,
	}
}

// Create a new watchlist.
func (s *WatchlistService) Create(
	ctx context.Context,
	userID uuid.UUID,
	name string,
) (*Watchlist, error) {

	name = strings.TrimSpace(name)

	if userID == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	if name == "" {
		return nil, fmt.Errorf("watchlist name is required")
	}

	if utf8.RuneCountInString(name) > 100 {
		return nil, fmt.Errorf("watchlist name cannot exceed 100 characters")
	}

	return s.repo.Create(ctx, userID, name)
}

// Get all watchlists belonging to a user.
func (s *WatchlistService) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]Watchlist, error) {

	if userID == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	return s.repo.GetByUserID(ctx, userID)
}

// Add a stock to a watchlist.
func (s *WatchlistService) AddStock(
	ctx context.Context,
	userID uuid.UUID,
	watchlistID uuid.UUID,
	instrumentID uuid.UUID,
) error {

	if userID == uuid.Nil || watchlistID == uuid.Nil || instrumentID == uuid.Nil {
		return fmt.Errorf("invalid user, watchlist, or instrument ID")
	}

	return s.repo.AddStock(
		ctx,
		userID,
		watchlistID,
		instrumentID,
	)
}

// Remove a stock from a watchlist.
func (s *WatchlistService) RemoveStock(
	ctx context.Context,
	userID uuid.UUID,
	watchlistID uuid.UUID,
	instrumentID uuid.UUID,
) error {

	if userID == uuid.Nil || watchlistID == uuid.Nil || instrumentID == uuid.Nil {
		return fmt.Errorf("invalid user, watchlist, or instrument ID")
	}

	return s.repo.RemoveStock(
		ctx,
		userID,
		watchlistID,
		instrumentID,
	)
}

// Get stocks belonging to a watchlist.
func (s *WatchlistService) GetWatchlistStocks(
	ctx context.Context,
	userID uuid.UUID,
	watchlistID uuid.UUID,
) ([]WatchlistStock, error) {

	if userID == uuid.Nil || watchlistID == uuid.Nil {
		return nil, fmt.Errorf("invalid user or watchlist ID")
	}

	return s.repo.GetWatchlistStocks(
		ctx,
		userID,
		watchlistID,
	)
}
