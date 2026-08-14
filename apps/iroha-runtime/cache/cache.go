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
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	BackendNone     = "none"
	BackendPostgres = "postgres"
	BackendValkey   = "valkey"
	keyPrefix       = "iroha:cache:v2:"

	NamespaceBriefing         = "read_briefing"
	NamespaceActivities       = "read_activities"
	NamespaceSleep            = "read_sleep"
	NamespaceDaily            = "read_daily"
	NamespaceMedia            = "read_media"
	NamespaceMetrics          = "read_metrics"
	NamespaceReports          = "read_reports"
	NamespacePublicSummary    = "public_summary"
	NamespacePublicActivities = "public_activities"
	NamespacePublicRoutes     = "public_routes"

	ChangeImport          ChangeKind = "import"
	ChangeExpense         ChangeKind = "expense"
	ChangeMediaResolution ChangeKind = "media_resolution"
	ChangeGeocode         ChangeKind = "geocode"
)

// ChangeKind identifies a canonical write whose dependent read namespaces
// must be invalidated after the write commits.
type ChangeKind string

var changeNamespaces = map[ChangeKind][]string{
	ChangeImport: {
		NamespaceBriefing,
		NamespaceActivities,
		NamespaceSleep,
		NamespaceDaily,
		NamespaceMedia,
		NamespaceMetrics,
		NamespaceReports,
		NamespacePublicSummary,
		NamespacePublicActivities,
		NamespacePublicRoutes,
	},
	ChangeExpense: {
		NamespaceMetrics,
		NamespaceReports,
	},
	ChangeMediaResolution: {
		NamespaceMedia,
		NamespaceReports,
	},
	ChangeGeocode: {
		NamespaceActivities,
		NamespacePublicRoutes,
	},
}

const (
	invalidationAttempts   = 3
	invalidationRetryDelay = 25 * time.Millisecond
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
	store                Store
	flightMu             sync.Mutex
	flights              map[string]*cacheFlight
	degradedMu           sync.RWMutex
	degraded             map[string]bool
	invalidationFailures atomic.Uint64
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
	return &Client{store: store, flights: make(map[string]*cacheFlight), degraded: make(map[string]bool)}
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

// InvalidateNamespace invalidates all entries in one logical namespace with a
// bounded retry policy. A persistent failure marks that namespace degraded so
// callers can bypass potentially stale cache data until a later invalidation
// succeeds.
func (c *Client) InvalidateNamespace(ctx context.Context, namespace string) error {
	if c == nil || c.store == nil {
		return nil
	}
	var err error
	attempts := 0
retry:
	for attempt := 1; attempt <= invalidationAttempts; attempt++ {
		attempts = attempt
		err = c.store.InvalidateNamespace(ctx, namespace)
		if err == nil {
			if c.setDegraded(namespace, false) {
				slog.Info("cache invalidation recovered", "namespace", namespace)
			}
			return nil
		}
		if attempt == invalidationAttempts {
			break
		}
		slog.Warn("cache invalidation failed; retrying", "namespace", namespace, "attempt", attempt, "max_attempts", invalidationAttempts, "error", err)
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break retry
		case <-time.After(invalidationRetryDelay):
		}
	}

	c.invalidationFailures.Add(1)
	c.setDegraded(namespace, true)
	slog.Error("cache namespace degraded after invalidation failure", "event", "cache_invalidation_degraded", "namespace", namespace, "attempts", attempts, "failure_count", c.invalidationFailures.Load(), "error", err)
	return err
}

// InvalidateNamespaces invalidates a set of logical namespaces in order and
// returns all failures. Duplicate namespaces are processed once.
func (c *Client) InvalidateNamespaces(ctx context.Context, namespaces ...string) error {
	seen := make(map[string]struct{}, len(namespaces))
	var failures []error
	for _, namespace := range namespaces {
		if namespace == "" {
			continue
		}
		if _, ok := seen[namespace]; ok {
			continue
		}
		seen[namespace] = struct{}{}
		if err := c.InvalidateNamespace(ctx, namespace); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", namespace, err))
		}
	}
	return errors.Join(failures...)
}

// InvalidateChange applies the canonical write-to-read dependency matrix.
func (c *Client) InvalidateChange(ctx context.Context, change ChangeKind) error {
	namespaces, ok := changeNamespaces[change]
	if !ok {
		return fmt.Errorf("unsupported cache change kind %q", change)
	}
	return c.InvalidateNamespaces(ctx, namespaces...)
}

// IsDegraded reports whether a namespace must bypass cache reads.
func (c *Client) IsDegraded(namespace string) bool {
	if c == nil {
		return false
	}
	c.degradedMu.RLock()
	degraded := c.degraded[namespace]
	c.degradedMu.RUnlock()
	return degraded
}

// InvalidationFailureCount returns the number of namespace invalidation
// operations that exhausted their bounded retry budget for this client.
func (c *Client) InvalidationFailureCount() uint64 {
	if c == nil {
		return 0
	}
	return c.invalidationFailures.Load()
}

func (c *Client) setDegraded(namespace string, degraded bool) bool {
	c.degradedMu.Lock()
	if c.degraded == nil {
		c.degraded = make(map[string]bool)
	}
	degradedBefore := c.degraded[namespace]
	if degraded {
		c.degraded[namespace] = true
	} else {
		delete(c.degraded, namespace)
	}
	c.degradedMu.Unlock()
	return degradedBefore && !degraded
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
