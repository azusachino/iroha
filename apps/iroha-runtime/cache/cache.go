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
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	BackendNone     = "none"
	BackendPostgres = "postgres"
	BackendValkey   = "valkey"
	keyPrefix       = "iroha:cache:v2:"

	NamespaceBriefing   = "read_briefing"
	NamespaceActivities = "read_activities"
	NamespaceSleep      = "read_sleep"
	NamespaceDaily      = "read_daily"
	NamespaceMedia      = "read_media"
	NamespaceMetrics    = "read_metrics"
	NamespaceReports    = "read_reports"
)

// Store is the backend contract for shared cache data. Namespace is a logical
// invalidation boundary, not a backend-specific key pattern.
type Store interface {
	Get(context.Context, string, string) ([]byte, bool, error)
	Set(context.Context, string, string, []byte, time.Duration) error
	InvalidateNamespace(context.Context, string) error
	Close() error
}

// GenerationStore extends Store with a compare-and-set boundary for cache
// population. A loader records the generation observed on its miss; the
// backend must skip the write if invalidation has advanced that generation
// before the loader finishes.
type GenerationStore interface {
	Store
	GetWithGeneration(context.Context, string, string) ([]byte, int64, bool, error)
	SetAtGeneration(context.Context, string, string, int64, []byte, time.Duration) (bool, error)
}

// Client is the cache facade used by application packages. It owns encoding
// and fail-open behavior; Store owns backend details.
type Client struct {
	store    Store
	flightMu sync.Mutex
	flights  map[string]*cacheFlight
}

type cacheFlight struct {
	done  chan struct{}
	value any
	err   error
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
	return &Client{store: store, flights: make(map[string]*cacheFlight)}
}

// NewBackend selects a configured cache backend. Cache data is disposable, so
// switching between backends does not require data migration.
func NewBackend(backend, valkeyURL string, db *gorm.DB) (*Client, error) {
	switch backend {
	case "", BackendPostgres:
		if db == nil {
			return nil, errors.New("postgres cache backend requires database")
		}
		return NewWithStore(NewPostgresStore(db)), nil
	case BackendValkey:
		return New(valkeyURL), nil
	case BackendNone:
		return New(""), nil
	default:
		return nil, fmt.Errorf("unsupported cache backend %q", backend)
	}
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
	value, generation, ok := GetWithGeneration[T](ctx, c, namespace, key)
	if ok {
		return value, nil
	}
	if c == nil || c.store == nil {
		return loader()
	}

	flightKey := namespace + "\x00" + key + "\x00" + reflect.TypeOf((*T)(nil)).Elem().String()
	c.flightMu.Lock()
	if flight, ok := c.flights[flightKey]; ok {
		c.flightMu.Unlock()
		select {
		case <-flight.done:
			value, ok := flight.value.(T)
			if !ok {
				var zero T
				return zero, flight.err
			}
			return value, flight.err
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		}
	}
	flight := &cacheFlight{done: make(chan struct{})}
	c.flights[flightKey] = flight
	c.flightMu.Unlock()

	value, err := loader()
	if err == nil {
		SetAtGeneration(ctx, c, namespace, key, generation, ttl, value)
	}
	c.flightMu.Lock()
	flight.value = value
	flight.err = err
	delete(c.flights, flightKey)
	close(flight.done)
	c.flightMu.Unlock()
	if err != nil {
		var zero T
		return zero, err
	}
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

// GetWithGeneration returns a decoded cache value, its observed namespace
// generation, and whether it was found. Stores without generation support use
// the ordinary Store contract and return generation zero.
func GetWithGeneration[T any](ctx context.Context, c *Client, namespace, key string) (T, int64, bool) {
	var zero T
	if c == nil || c.store == nil {
		return zero, 0, false
	}

	store, ok := c.store.(GenerationStore)
	if !ok {
		value, found := Get[T](ctx, c, namespace, key)
		return value, 0, found
	}

	raw, generation, found, err := store.GetWithGeneration(ctx, namespace, key)
	if err != nil {
		slog.Warn("cache get failed", "namespace", namespace, "key", key, "error", err)
		return zero, 0, false
	}
	if !found {
		return zero, generation, false
	}

	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		slog.Warn("cache decode failed", "namespace", namespace, "key", key, "error", err)
		return zero, generation, false
	}
	return value, generation, true
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

// SetAtGeneration best-effort populates the cache only when the observed
// namespace generation is still current. Stores without generation support
// fall back to the ordinary Set contract.
func SetAtGeneration[T any](ctx context.Context, c *Client, namespace, key string, generation int64, ttl time.Duration, value T) bool {
	if c == nil || c.store == nil {
		return false
	}

	raw, err := json.Marshal(value)
	if err != nil {
		slog.Warn("cache encode failed", "namespace", namespace, "key", key, "error", err)
		return false
	}
	if store, ok := c.store.(GenerationStore); ok {
		stored, err := store.SetAtGeneration(ctx, namespace, key, generation, raw, ttl)
		if err != nil {
			slog.Warn("cache conditional set failed", "namespace", namespace, "key", key, "generation", generation, "error", err)
		}
		return stored
	}
	if err := c.store.Set(ctx, namespace, key, raw, ttl); err != nil {
		slog.Warn("cache set failed", "namespace", namespace, "key", key, "error", err)
		return false
	}
	return true
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

var setAtGenerationScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
	return 0
end
redis.call("SET", KEYS[2], ARGV[3], "PX", ARGV[2])
return 1
`)

func (s *valkeyStore) Get(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	raw, _, found, err := s.GetWithGeneration(ctx, namespace, key)
	return raw, found, err
}

func (s *valkeyStore) GetWithGeneration(ctx context.Context, namespace, key string) ([]byte, int64, bool, error) {
	generation, err := s.generation(ctx, namespace)
	if err != nil {
		return nil, 0, false, err
	}
	raw, err := s.client.Get(ctx, namespacedKey(namespace, generation, key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, generation, false, nil
		}
		return nil, generation, false, err
	}
	return raw, generation, true, nil
}

func (s *valkeyStore) Set(ctx context.Context, namespace, key string, raw []byte, ttl time.Duration) error {
	generation, err := s.generation(ctx, namespace)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, namespacedKey(namespace, generation, key), raw, ttl).Err()
}

func (s *valkeyStore) SetAtGeneration(ctx context.Context, namespace, key string, generation int64, raw []byte, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, nil
	}
	result, err := setAtGenerationScript.Run(ctx, s.client,
		[]string{generationKey(namespace), namespacedKey(namespace, generation, key)},
		strconv.FormatInt(generation, 10), strconv.FormatInt(ttl.Milliseconds(), 10), raw,
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (s *valkeyStore) InvalidateNamespace(ctx context.Context, namespace string) error {
	_, err := s.client.Incr(ctx, generationKey(namespace)).Result()
	return err
}

func (s *valkeyStore) Close() error {
	return s.client.Close()
}

func (s *valkeyStore) generation(ctx context.Context, namespace string) (int64, error) {
	key := generationKey(namespace)
	generation, err := s.client.Get(ctx, key).Int64()
	if err == nil {
		return generation, nil
	}
	if !errors.Is(err, redis.Nil) {
		return 0, err
	}
	if _, err := s.client.SetNX(ctx, key, 1, 0).Result(); err != nil {
		return 0, err
	}
	return 1, nil
}

func generationKey(namespace string) string {
	return keyPrefix + namespace + ":__generation"
}

func namespacedKey(namespace string, generation int64, key string) string {
	return keyPrefix + namespace + ":g" + fmt.Sprint(generation) + ":" + key
}
