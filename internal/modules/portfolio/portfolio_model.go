package portfolio

import "github.com/google/uuid"

type PortfolioPosition struct {
	InstrumentID    uuid.UUID `json:"instrument_id"`
	Symbol          string    `json:"symbol"`
	CompanyName     string    `json:"company_name"`
	Exchange        string    `json:"exchange"`
	Quantity        int64     `json:"quantity"`
	AverageBuyPrice string    `json:"average_buy_price"`
	CurrentPrice    string    `json:"current_price"`
	MarketValue     string    `json:"market_value"`
	InvestedValue   string    `json:"invested_value"`
	UnrealizedPnL   string    `json:"unrealized_pnl"`
	MarketStatus    string    `json:"market_status"`
}

type PortfolioSummary struct {
	CashBalance    string              `json:"cash_balance"`
	InvestedValue  string              `json:"invested_value"`
	CurrentValue   string              `json:"current_value"`
	TotalPortfolio string              `json:"total_portfolio"`
	UnrealizedPnL  string              `json:"unrealized_pnl"`
	Positions      []PortfolioPosition `json:"positions"`
}
