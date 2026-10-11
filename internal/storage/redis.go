package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStore creates a new RedisStore.
// If ttl is 0, keys will persist without expiration.
func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{
		client: client,
		ttl:    ttl,
	}
}

func (r *RedisStore) Save(ctx context.Context, url URL) error {
	data, err := json.Marshal(url)
	if err != nil {
		return err
	}

	key := "url:" + url.Code
	// SetNX ensures we don't overwrite existing short codes (conflict check)
	ok, err := r.client.SetNX(ctx, key, data, r.ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrConflict
	}
	return nil
}

func (r *RedisStore) GetByCode(ctx context.Context, code string) (URL, error) {
	key := "url:" + code
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return URL{}, ErrNotFound
		}
		return URL{}, err
	}

	var url URL
	if err := json.Unmarshal(data, &url); err != nil {
		return URL{}, err
	}

	return url, nil
}

