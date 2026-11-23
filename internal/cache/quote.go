package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const quoteTTL = 30 * time.Second

type QuoteCache struct {
	client *redis.Client
}

func NewQuoteCache(client *redis.Client) *QuoteCache {
	return &QuoteCache{
		client: client,
	}
}

func quoteKey(instrumentID uuid.UUID) string {
	return fmt.Sprintf("tradex:quote:%s", instrumentID)
}

func (q *QuoteCache) SetQuote(
	ctx context.Context,
	instrumentID uuid.UUID,
	quote any,
) error {
	data, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("marshal quote: %w", err)
	}

	if err := q.client.Set(
		ctx,
		quoteKey(instrumentID),
		data,
		quoteTTL,
	).Err(); err != nil {
		return fmt.Errorf("cache quote: %w", err)
	}

	return nil
}

func (q *QuoteCache) GetQuote(
	ctx context.Context,
	instrumentID uuid.UUID,
	result any,
) error {
	data, err := q.client.Get(
		ctx,
		quoteKey(instrumentID),
	).Bytes()

	if err != nil {
		if err == redis.Nil {
			return redis.Nil
		}
		return fmt.Errorf("get cached quote: %w", err)
	}

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("unmarshal cached quote: %w", err)
	}

	return nil
}

func (q *QuoteCache) DeleteQuote(
	ctx context.Context,
	instrumentID uuid.UUID,
) error {
	if err := q.client.Del(
		ctx,
		quoteKey(instrumentID),
	).Err(); err != nil {
		return fmt.Errorf("delete cached quote: %w", err)
	}

	return nil
}
