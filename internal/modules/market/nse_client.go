package market

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const nseMCPURL = "https://mcp.nseindia.in/cmmkt/mcp"

type NSEClient struct {
	httpClient *http.Client
}

func NewNSEClient() *NSEClient {
	return &NSEClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type nseRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

type nseToolResponse struct {
	Result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type NSEStockQuote struct {
	UpdatedAt string `json:"updatedAt"`

	Stock struct {
		Type             string  `json:"type"`
		Symbol           string  `json:"symbol"`
		Series           string  `json:"series"`
		OpenPrice        float64 `json:"openPrice"`
		HighPrice        float64 `json:"highPrice"`
		LowPrice         float64 `json:"lowPrice"`
		PreClosePrice    float64 `json:"preClosePrice"`
		LastTradedPrice  float64 `json:"lastTradedPrice"`
		Change           float64 `json:"change"`
		PerChange        float64 `json:"perChange"`
		Volume           int64   `json:"volume"`
		Value            float64 `json:"value"`
		FiftyTwoWeekHigh float64 `json:"fiftyTwoWeekHigh"`
		FiftyTwoWeekLow  float64 `json:"fiftyTwoWeekLow"`
		PerChange30d     float64 `json:"perChange30d"`
		LatestTimestamp  string  `json:"latestTimestamp"`
	} `json:"stock"`
}

func (c *NSEClient) GetStockQuote(
	ctx context.Context,
	symbol string,
) (*NSEStockQuote, error) {

	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("stock symbol is required")
	}

	// Step 1: Initialize an MCP session.
	initPayload := nseRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]string{
				"name":    "tradex",
				"version": "1.0.0",
			},
		},
	}

	var initResponse map[string]any

	sessionID, err := c.sendJSON(ctx, initPayload, &initResponse, "")
	if err != nil {
		return nil, fmt.Errorf("initialize NSE session: %w", err)
	}

	if sessionID == "" {
		return nil, fmt.Errorf("NSE did not return a session ID")
	}

	// Step 2: Complete MCP initialization.
	notification := nseRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	if _, err := c.sendJSON(ctx, notification, nil, sessionID); err != nil {
		return nil, fmt.Errorf("complete NSE initialization: %w", err)
	}

	// Step 3: Request the stock quote.
	quoteRequest := nseRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "cm_get_stock_quote",
			"arguments": map[string]string{
				"symbol": symbol,
			},
		},
	}

	var rawResponse nseToolResponse

	if _, err := c.sendSSE(ctx, quoteRequest, sessionID, &rawResponse); err != nil {
		return nil, fmt.Errorf("fetch NSE stock quote: %w", err)
	}

	if rawResponse.Error != nil {
		return nil, fmt.Errorf("NSE error: %s", rawResponse.Error.Message)
	}

	if rawResponse.Result.IsError {
		return nil, fmt.Errorf("NSE returned an unsuccessful tool result")
	}

	if len(rawResponse.Result.Content) == 0 {
		return nil, fmt.Errorf("NSE returned empty quote data")
	}

	var quote NSEStockQuote

	if err := json.Unmarshal(
		[]byte(rawResponse.Result.Content[0].Text),
		&quote,
	); err != nil {
		return nil, fmt.Errorf("decode NSE quote: %w", err)
	}

	if quote.Stock.Symbol == "" {
		return nil, fmt.Errorf("NSE quote does not contain a stock symbol")
	}

	return &quote, nil
}

func (c *NSEClient) sendJSON(
	ctx context.Context,
	payload nseRequest,
	result any,
	sessionID string,
) (string, error) {

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		nseMCPURL,
		bytes.NewReader(data),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Attach the existing MCP session ID to subsequent requests.
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("NSE HTTP status %d: %s",
			resp.StatusCode, string(body))
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return "", err
		}
	}

	return resp.Header.Get("Mcp-Session-Id"), nil
}

func (c *NSEClient) sendSSE(
	ctx context.Context,
	payload nseRequest,
	sessionID string,
	result any,
) (string, error) {

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		nseMCPURL,
		bytes.NewReader(data),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Session-Id", sessionID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("NSE HTTP status %d: %s",
			resp.StatusCode, string(body))
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

		if err := json.Unmarshal([]byte(jsonData), result); err != nil {
			return "", err
		}

		return sessionID, nil
	}

	return "", fmt.Errorf("NSE returned no SSE data event")
}
