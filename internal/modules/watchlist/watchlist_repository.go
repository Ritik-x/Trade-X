package watchlist

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WatchlistRepository struct {
	db *pgxpool.Pool
}

func NewWatchlistRepository(db *pgxpool.Pool) *WatchlistRepository {
	return &WatchlistRepository{db: db}
}

type WatchlistStock struct {
	InstrumentID  uuid.UUID  `json:"instrument_id"`
	Symbol        string     `json:"symbol"`
	CompanyName   string     `json:"company_name"`
	Exchange      string     `json:"exchange"`
	LastPrice     *string    `json:"last_price"`
	PriceChange   *string    `json:"price_change"`
	ChangePercent *string    `json:"change_percent"`
	MarketStatus  string     `json:"market_status"`
	LastUpdatedAt *time.Time `json:"last_updated_at"`
}
type Watchlist struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new watchlist for a user.
func (r *WatchlistRepository) Create(
	ctx context.Context,
	userID uuid.UUID,
	name string,
) (*Watchlist, error) {

	query := `
        INSERT INTO watchlists (
            user_id,
            name
        )
        VALUES ($1, $2)
        RETURNING id, user_id, name, created_at, updated_at
    `

	var watchlist Watchlist

	err := r.db.QueryRow(
		ctx,
		query,
		userID,
		name,
	).Scan(
		&watchlist.ID,
		&watchlist.UserID,
		&watchlist.Name,
		&watchlist.CreatedAt,
		&watchlist.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create watchlist: %w", err)
	}

	return &watchlist, nil
}

// Fetch all watchlists belonging to a user.
func (r *WatchlistRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]Watchlist, error) {

	query := `
        SELECT
            id,
            user_id,
            name,
            created_at,
            updated_at
        FROM watchlists
        WHERE user_id = $1
        ORDER BY created_at DESC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user watchlists: %w", err)
	}
	defer rows.Close()

	watchlists := make([]Watchlist, 0)

	for rows.Next() {
		var watchlist Watchlist

		err := rows.Scan(
			&watchlist.ID,
			&watchlist.UserID,
			&watchlist.Name,
			&watchlist.CreatedAt,
			&watchlist.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan watchlist: %w", err)
		}

		watchlists = append(watchlists, watchlist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watchlists: %w", err)
	}

	return watchlists, nil
}

// Add an instrument to a user's watchlist.
func (r *WatchlistRepository) AddStock(
	ctx context.Context,
	userID uuid.UUID,
	watchlistID uuid.UUID,
	instrumentID uuid.UUID,
) error {

	query := `
        INSERT INTO watchlist_items (
            watchlist_id,
            instrument_id
        )
        SELECT $2, $3
        FROM watchlists
        WHERE id = $2
          AND user_id = $1
        ON CONFLICT (watchlist_id, instrument_id)
        DO NOTHING
    `

	result, err := r.db.Exec(
		ctx,
		query,
		userID,
		watchlistID,
		instrumentID,
	)

	if err != nil {
		return fmt.Errorf("add stock to watchlist: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("watchlist not found or stock already added")
	}

	return nil
}

// Remove an instrument from a user's watchlist.
func (r *WatchlistRepository) RemoveStock(
	ctx context.Context,
	userID uuid.UUID,
	watchlistID uuid.UUID,
	instrumentID uuid.UUID,
) error {

	query := `
        DELETE FROM watchlist_items wi
        USING watchlists w
        WHERE wi.watchlist_id = w.id
          AND w.id = $2
          AND w.user_id = $1
          AND wi.instrument_id = $3
    `

	result, err := r.db.Exec(
		ctx,
		query,
		userID,
		watchlistID,
		instrumentID,
	)

	if err != nil {
		return fmt.Errorf("remove stock from watchlist: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("stock or watchlist not found")
	}

	return nil
}

func (r *WatchlistRepository) GetWatchlistStocks(
	ctx context.Context,
	userID uuid.UUID,
	watchlistID uuid.UUID,
) ([]WatchlistStock, error) {

	query := `
        SELECT
            i.id,
            i.symbol,
            i.company_name,
            i.exchange,
            mq.last_price::TEXT,
            mq.price_change::TEXT,
            mq.change_percent::TEXT,
            COALESCE(mq.market_status, 'UNAVAILABLE'),
            mq.last_updated_at
        FROM watchlist_items wi
        JOIN watchlists w
            ON w.id = wi.watchlist_id
        JOIN instruments i
            ON i.id = wi.instrument_id
        LEFT JOIN market_quotes mq
            ON mq.instrument_id = i.id
        WHERE w.id = $1
          AND w.user_id = $2
        ORDER BY wi.created_at DESC
    `

	rows, err := r.db.Query(ctx, query, watchlistID, userID)
	if err != nil {
		return nil, fmt.Errorf("get watchlist stocks: %w", err)
	}
	defer rows.Close()

	stocks := make([]WatchlistStock, 0)

	for rows.Next() {
		var stock WatchlistStock

		err := rows.Scan(
			&stock.InstrumentID,
			&stock.Symbol,
			&stock.CompanyName,
			&stock.Exchange,
			&stock.LastPrice,
			&stock.PriceChange,
			&stock.ChangePercent,
			&stock.MarketStatus,
			&stock.LastUpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan watchlist stock: %w", err)
		}

		stocks = append(stocks, stock)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watchlist stocks: %w", err)
	}

	return stocks, nil
}
