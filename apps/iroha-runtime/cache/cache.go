// Package cache provides a backend-neutral, cache-aside store.
//
// The cache is explicitly best effort: misses and backend failures fall back
// to the caller's loader. The current Valkey implementation can be replaced by
// another Store without changing cache callers.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store is the backend contract for shared cache data. Namespace is a logical
// invalidation boundary, not a backend-specific key pattern.
type Store interface {
	Get(context.Context, string, string) ([]byte, bool, error)
	Set(context.Context, string, string, []byte, time.Duration) error
	InvalidateNamespace(context.Context, string) error
	Close() error
}

// Client is the cache facade used by application packages. It owns encoding
// and fail-open behavior; Store owns backend details.
type Client struct {
	store Store
}

// New builds a Valkey-backed Client for url. An empty or malformed URL
// disables caching rather than making cache availability a request concern.
func New(url string) *Client {
	if url == "" {
		return &Client{}
	}

	opt, err := redis.ParseURL(url)
	if err != nil {
		slog.Warn("cache disabled: invalid url", "error", err)
		return &Client{}
	}

	return NewWithStore(&valkeyStore{client: redis.NewClient(opt)})
}

// NewWithStore builds a Client around any Store implementation.
func NewWithStore(store Store) *Client {
	return &Client{store: store}
}

// Close releases the backend connection, if any.
func (c *Client) Close() error {
	if c == nil || c.store == nil {
		return nil
	}
	return c.store.Close()
}

// GetOrLoad implements cache-aside lookup. Cache misses, decode failures, and
// backend errors call loader; only loader's own error is returned.
func GetOrLoad[T any](ctx context.Context, c *Client, namespace, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	if value, ok := Get[T](ctx, c, namespace, key); ok {
		return value, nil
	}

	value, err := loader()
	if err != nil {
		var zero T
		return zero, err
	}

	Set(ctx, c, namespace, key, ttl, value)
	return value, nil
}

// Get returns a decoded cache value and whether it was found. Backend errors
// and decode failures are treated as misses.
func Get[T any](ctx context.Context, c *Client, namespace, key string) (T, bool) {
	var zero T
	if c == nil || c.store == nil {
		return zero, false
	}

	raw, found, err := c.store.Get(ctx, namespace, key)
	if err != nil {
		slog.Warn("cache get failed", "namespace", namespace, "key", key, "error", err)
		return zero, false
	}
	if !found {
		return zero, false
	}

	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		slog.Warn("cache decode failed", "namespace", namespace, "key", key, "error", err)
		return zero, false
	}
	return value, true
}

// Set best-effort populates the cache. Serialization or backend errors are
// logged and intentionally do not fail the request.
func Set[T any](ctx context.Context, c *Client, namespace, key string, ttl time.Duration, value T) {
	if c == nil || c.store == nil {
		return
	}

	raw, err := json.Marshal(value)
	if err != nil {
		slog.Warn("cache encode failed", "namespace", namespace, "key", key, "error", err)
		return
	}
	if err := c.store.Set(ctx, namespace, key, raw, ttl); err != nil {
		slog.Warn("cache set failed", "namespace", namespace, "key", key, "error", err)
	}
}

// InvalidateNamespace invalidates all entries in one logical namespace.
func (c *Client) InvalidateNamespace(ctx context.Context, namespace string) error {
	if c == nil || c.store == nil {
		return nil
	}
	return c.store.InvalidateNamespace(ctx, namespace)
}

type valkeyStore struct {
	client *redis.Client
}

func (s *valkeyStore) Get(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	raw, err := s.client.Get(ctx, namespacedKey(namespace, key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return raw, true, nil
}

func (s *valkeyStore) Set(ctx context.Context, namespace, key string, raw []byte, ttl time.Duration) error {
	return s.client.Set(ctx, namespacedKey(namespace, key), raw, ttl).Err()
}

func (s *valkeyStore) InvalidateNamespace(ctx context.Context, namespace string) error {
	var cursor uint64
	pattern := namespacedKey(namespace, "*")
	for {
		keys, next, err := s.client.Scan(ctx, cursor, pattern, 0).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if _, err := s.client.Del(ctx, keys...).Result(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func (s *valkeyStore) Close() error {
	return s.client.Close()
}

func namespacedKey(namespace, key string) string {
	return namespace + ":" + key
}
