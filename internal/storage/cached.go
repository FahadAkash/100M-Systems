package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/FahadAkash/100M-Systems/pkg/events"
	"github.com/FahadAkash/100M-Systems/pkg/metrics"
	"github.com/redis/go-redis/v9"
)

// CachedStore implements Store with a primary persistent store (e.g., PostgreSQL)
// and a Redis cache for fast lookups (cache-aside pattern).
type CachedStore struct {
	primary Store
	redis   *redis.Client
	ttl     time.Duration
}

func NewCachedStore(primary Store, redisClient *redis.Client, ttl time.Duration) *CachedStore {
	return &CachedStore{
		primary: primary,
		redis:   redisClient,
		ttl:     ttl,
	}
}

func (c *CachedStore) Save(ctx context.Context, url URL) error {
	// Persist to primary store first
	if err := c.primary.Save(ctx, url); err != nil {
		return err
	}

	// Warm cache in Redis
	if c.redis != nil {
		data, err := json.Marshal(url)
		if err == nil {
			_ = c.redis.Set(ctx, "url:"+url.Code, data, c.ttl).Err()
		}
	}

	return nil
}

func (c *CachedStore) GetByCode(ctx context.Context, code string) (URL, error) {
	// 1. Try cache hit from Redis
	if c.redis != nil {
		data, err := c.redis.Get(ctx, "url:"+code).Bytes()
		if err == nil {
			var url URL
			if err := json.Unmarshal(data, &url); err == nil {
				metrics.CacheHitsTotal.Inc()
				return url, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			// Redis error: fall through to primary store seamlessly
		}
	}

	// 2. Cache miss: fetch from primary persistent store
	metrics.CacheMissesTotal.Inc()
	url, err := c.primary.GetByCode(ctx, code)
	if err != nil {
		return URL{}, err
	}

	// 3. Populate Redis cache for subsequent requests
	if c.redis != nil {
		data, err := json.Marshal(url)
		if err == nil {
			_ = c.redis.Set(ctx, "url:"+code, data, c.ttl).Err()
		}
	}

	return url, nil
}

// SaveClickBatch delegates batch persistence to the primary store if it implements AnalyticsStore.
func (c *CachedStore) SaveClickBatch(ctx context.Context, batch []events.ClickEvent) error {
	if a, ok := c.primary.(AnalyticsStore); ok {
		return a.SaveClickBatch(ctx, batch)
	}
	return nil
}

// GetAnalytics queries analytics through the underlying primary store.
func (c *CachedStore) GetAnalytics(ctx context.Context, code string) (events.ClickAnalyticsSummary, error) {
	if a, ok := c.primary.(AnalyticsStore); ok {
		return a.GetAnalytics(ctx, code)
	}
	return events.ClickAnalyticsSummary{}, ErrNotFound
}

