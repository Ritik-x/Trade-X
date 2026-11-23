package market

import (
	"time"

	"github.com/google/uuid"
)

type Quote struct {
	InstrumentID  uuid.UUID  `json:"instrument_id"`
	LastPrice     *string    `json:"last_price"`
	OpenPrice     *string    `json:"open_price"`
	HighPrice     *string    `json:"high_price"`
	LowPrice      *string    `json:"low_price"`
	PreviousClose *string    `json:"previous_close"`
	Volume        int64      `json:"volume"`
	PriceChange   *string    `json:"price_change"`
	ChangePercent *string    `json:"change_percent"`
	MarketStatus  string     `json:"market_status"`
	LastUpdatedAt *time.Time `json:"last_updated_at"`
}
