// Package cache provides a cache-aside client backed by valkey/redis.
//
// The client degrades gracefully: when no URL is configured, or the
// configured URL is unreachable, GetOrLoad falls back to calling the
// loader directly. A cache outage must never turn into a request failure.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a redis client. A nil rdb means caching is disabled and every
// operation falls through to the caller's loader.
type Client struct {
	rdb *redis.Client
}

// New builds a Client for url. An empty url disables caching. A malformed
// url also disables caching (logged once at Warn level) rather than
// panicking, since caching is a non-essential dependency.
func New(url string) *Client {
	if url == "" {
		return &Client{}
	}

	opt, err := redis.ParseURL(url)
	if err != nil {
		slog.Warn("cache disabled: invalid url", "error", err)
		return &Client{}
	}

	return &Client{rdb: redis.NewClient(opt)}
}

// Close releases the underlying connection pool, if any.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// GetOrLoad implements cache-aside lookup: try the cache first, and on any
// miss or error, call loader and best-effort populate the cache with its
// result. Only loader's own error is ever returned; cache failures never
// fail the request.
func GetOrLoad[T any](ctx context.Context, c *Client, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	if c == nil || c.rdb == nil {
		return loader()
	}

	if value, ok := get[T](ctx, c, key); ok {
		return value, nil
	}

	value, err := loader()
	if err != nil {
		var zero T
		return zero, err
	}

	set(ctx, c, key, ttl, value)
	return value, nil
}

// get returns the cached value for key and whether it was found and
// successfully decoded. Any redis error or decode failure is treated as a
// miss.
func get[T any](ctx context.Context, c *Client, key string) (T, bool) {
	var zero T

	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.Warn("cache get failed", "key", key, "error", err)
		}
		return zero, false
	}

	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		slog.Warn("cache decode failed", "key", key, "error", err)
		return zero, false
	}

	return value, true
}

// set best-effort populates the cache; failures are logged, not propagated.
func set[T any](ctx context.Context, c *Client, key string, ttl time.Duration, value T) {
	raw, err := json.Marshal(value)
	if err != nil {
		slog.Warn("cache encode failed", "key", key, "error", err)
		return
	}

	if err := c.rdb.Set(ctx, key, raw, ttl).Err(); err != nil {
		slog.Warn("cache set failed", "key", key, "error", err)
	}
}

// DeletePattern finds all keys matching pattern and deletes them.
func (c *Client) DeletePattern(ctx context.Context, pattern string) error {
	if c == nil || c.rdb == nil {
		return nil
	}

	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.rdb.Scan(ctx, cursor, pattern, 0).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if _, err := c.rdb.Del(ctx, keys...).Result(); err != nil {
				return err
			}
		}
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Get wraps the unexported get function.
func Get[T any](ctx context.Context, c *Client, key string) (T, bool) {
	if c == nil || c.rdb == nil {
		var zero T
		return zero, false
	}
	return get[T](ctx, c, key)
}

// Set wraps the unexported set function.
func Set[T any](ctx context.Context, c *Client, key string, ttl time.Duration, value T) {
	if c == nil || c.rdb == nil {
		return
	}
	set(ctx, c, key, ttl, value)
}
