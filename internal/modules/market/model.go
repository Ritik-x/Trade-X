package market

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrQuoteNotFound = errors.New("quote not found")

type Instrument struct {
	ID             uuid.UUID `json:"id"`
	Symbol         string    `json:"symbol"`
	CompanyName    string    `json:"company_name"`
	Exchange       string    `json:"exchange"`
	InstrumentType string    `json:"instrument_type"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
