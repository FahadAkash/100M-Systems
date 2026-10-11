package storage

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/FahadAkash/100M-Systems/pkg/events"
	"github.com/FahadAkash/100M-Systems/pkg/metrics"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const negativeCacheSentinel = "__NOT_FOUND__"

type l1Item struct {
	url       URL
	notFound  bool
	expiresAt time.Time
}

// CachedStore implements Store with a primary persistent store (e.g., PostgreSQL),
// an L1 in-memory local cache, an L2 Redis distributed cache, singleflight request deduplication,
// and negative caching protection against cache penetration.
type CachedStore struct {
	primary Store
	redis   *redis.Client
	ttl     time.Duration
	sfGroup singleflight.Group

	l1Mu sync.RWMutex
	l1   map[string]l1Item
}

func NewCachedStore(primary Store, redisClient *redis.Client, ttl time.Duration) *CachedStore {
	cs := &CachedStore{
		primary: primary,
		redis:   redisClient,
		ttl:     ttl,
		l1:      make(map[string]l1Item),
	}

	go cs.l1Janitor()
	return cs
}

func (c *CachedStore) l1Janitor() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.l1Mu.Lock()
		for k, v := range c.l1 {
			if now.After(v.expiresAt) {
				delete(c.l1, k)
			}
		}
		c.l1Mu.Unlock()
	}
}

func (c *CachedStore) setL1(code string, url URL, notFound bool, ttl time.Duration) {
	c.l1Mu.Lock()
	defer c.l1Mu.Unlock()
	if len(c.l1) > 20000 {
		for k := range c.l1 {
			delete(c.l1, k)
			break
		}
	}
	c.l1[code] = l1Item{
		url:       url,
		notFound:  notFound,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *CachedStore) getL1(code string) (URL, bool, bool) {
	c.l1Mu.RLock()
	defer c.l1Mu.RUnlock()
	item, ok := c.l1[code]
	if !ok || time.Now().After(item.expiresAt) {
		return URL{}, false, false
	}
	return item.url, item.notFound, true
}

func (c *CachedStore) Save(ctx context.Context, url URL) error {
	// 1. Persist to primary store first
	if err := c.primary.Save(ctx, url); err != nil {
		return err
	}

	// 2. Warm L2 cache in Redis
	if c.redis != nil {
		data, err := json.Marshal(url)
		if err == nil {
			_ = c.redis.Set(ctx, "url:"+url.Code, data, c.ttl).Err()
		}
	}

	// 3. Warm L1 in-memory cache (zero network hops for immediate follow-up requests)
	c.setL1(url.Code, url, false, 3*time.Minute)

	return nil
}

func (c *CachedStore) GetByCode(ctx context.Context, code string) (URL, error) {
	// Tier 1: In-Memory L1 Cache Check (Sub-microsecond latency, 0 network hops)
	if url, notFound, found := c.getL1(code); found {
		if notFound {
			return URL{}, ErrNotFound
		}
		metrics.CacheHitsTotal.Inc()
		return url, nil
	}

	// Tier 2: Distributed L2 Redis Cache Check
	if c.redis != nil {
		raw, err := c.redis.Get(ctx, "url:"+code).Result()
		if err == nil {
			if raw == negativeCacheSentinel {
				// Negative cache hit! Cache in L1 and return ErrNotFound without touching Postgres
				c.setL1(code, URL{}, true, 30*time.Second)
				return URL{}, ErrNotFound
			}

			var url URL
			if err := json.Unmarshal([]byte(raw), &url); err == nil {
				metrics.CacheHitsTotal.Inc()
				// Warm L1 memory cache
				c.setL1(code, url, false, 2*time.Minute)
				return url, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			// Redis network/connection error: fall through smoothly
		}
	}

	// Tier 3: Primary Persistent Store with Singleflight Deduplication & Negative Caching
	metrics.CacheMissesTotal.Inc()

	result, err, _ := c.sfGroup.Do(code, func() (any, error) {
		// Fetch from primary persistent storage (PostgreSQL)
		url, err := c.primary.GetByCode(ctx, code)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// Protect against Cache Penetration (Negative Caching in Redis & L1)
				if c.redis != nil {
					_ = c.redis.Set(ctx, "url:"+code, negativeCacheSentinel, 60*time.Second).Err()
				}
				c.setL1(code, URL{}, true, 30*time.Second)
				return URL{}, ErrNotFound
			}
			return URL{}, err
		}

		// Warm L2 Redis
		if c.redis != nil {
			data, err := json.Marshal(url)
			if err == nil {
				_ = c.redis.Set(ctx, "url:"+code, data, c.ttl).Err()
			}
		}

		// Warm L1 Local Memory
		c.setL1(code, url, false, 2*time.Minute)

		return url, nil
	})

	if err != nil {
		return URL{}, err
	}
	return result.(URL), nil
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
