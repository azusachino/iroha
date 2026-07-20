# ADR 0003: Cache backends and invalidation

- Status: Proposed
- Date: 2026-07-18
- Depends on: [ADR 0001](0001-provider-observations-and-canonical-records.md)

## Context

Iroha currently uses Valkey for a small cache-aside layer. The cache stores public
summary responses, public activity pages, public route GeoJSON, and background
reverse-geocoding results. It is not the source of truth for canonical data,
jobs, sync cursors, authentication, or authorization.

The current cache package is coupled to the Redis protocol and exposes a
Redis-specific pattern deletion operation. This makes a cache backend change a
caller change, and it treats historical public data as a short-lived TTL cache
even though imports are the real source of invalidation.

Iroha is a self-hosted application with Postgres already required for canonical
data and durable jobs. The cache must remain best-effort: losing a cache entry
must cause a reload, not a request failure.

## Decision

Define a backend-neutral cache store in the runtime package. Valkey and Postgres
are interchangeable implementations of the same contract; callers do not
select keys, commands, or invalidation mechanics for a particular backend.

The first Postgres implementation uses a logged table. Cache data is
rebuildable, but logged storage keeps stable historical responses warm across
ordinary restarts and makes the default behavior easier to operate and inspect.
An `UNLOGGED` implementation may be added for high-churn disposable namespaces
after measuring WAL and table pressure. It must never be used for canonical
data, jobs, sync state, or authorization state.

### Store contract

The public contract is intentionally small:

```go
type Store interface {
	Get(ctx context.Context, namespace, key string, dst any) (hit bool, err error)
	Set(ctx context.Context, namespace, key string, value any, ttl time.Duration) error
	Invalidate(ctx context.Context, namespace string) error
}
```

The runtime cache facade owns JSON encoding, cache-key validation, and
fail-open behavior. A backend may return an operational error for metrics and
diagnostics, but callers treat backend errors as cache misses and continue to
the loader. Cache writes and invalidation are best effort unless the caller is
explicitly running a cache maintenance operation.

`Invalidate` is namespace-based. Callers must not use backend-specific pattern
scans such as `public:*`. A Valkey backend can map namespaces to key prefixes;
the Postgres backend can advance a namespace generation and leave old rows for
bounded cleanup.

### Cache entry semantics

Each entry has a namespace, logical key, generation, encoded value, creation
time, and expiry time. A hit is valid only when:

1. the entry belongs to the current namespace generation; and
2. it has not passed its expiry time.

The cache does not serve stale values by default. Stale-while-revalidate may
be added only as an explicit namespace policy; it is not part of the base
contract.

The logical key must contain all response-affecting filters and representation
versions. User identity, authorization scope, or private data must not be
omitted from a key if a private namespace is introduced later.

### Namespace policies

| Namespace | Data | Policy |
| --- | --- | --- |
| `public_summary` | aggregate public summary | short TTL for current periods; generation invalidation after import |
| `public_activities` | sanitized activity pages | generation invalidation after import; historical pages may use a long TTL |
| `public_routes` | sanitized route GeoJSON | generation invalidation after route/geocode changes |
| `geocode` | temporary provider lookup result | separate durable geocode table; not a generic response-cache entry |

Historical public responses are stable until an import or projection refresh
changes their source data. Their cache identity therefore includes the current
public projection generation rather than relying only on a fixed 24-hour TTL.
Current-period responses retain short TTLs because the active period changes
frequently.

### Durable versus disposable data

The following remain logged, canonical or durable records:

- raw files and connector snapshots;
- import jobs and import snapshots;
- activities, sleep, daily, and media projections;
- task queue state and sync cursors;
- geocode results when they are retained as enrichment evidence;
- future authentication revocations.

The response cache is disposable. A database restart, cache cleanup, backend
switch, or expired generation may remove entries without data loss.

### Geocoding

Reverse geocoding is not treated as a pure response-cache concern. The rounded
coordinate/provider result is stored durably so repeated route rendering does
not depend on Valkey or a process-local map. Refreshes are scheduled through
the durable job dispatcher, with a unique coordinate key preventing duplicate
work. The public geocode endpoint reads the durable result before calling the
external provider.

### Authentication and rate limiting

JWT validation remains stateless and local. There is no authentication cache.
If token revocation is required later, revocations become logged authorization
state keyed by token ID and expiry, not cache entries.

The current rate limiter is process-local. That is the supported behavior for
the single-server self-hosted deployment. Rate limiting is a separate contract
from response caching because it requires atomic counters and an explicit
failure policy; cache misses and namespace invalidation must never affect
limits.

If multi-instance deployment becomes a supported target, Valkey may be added as
an optional `Limiter` backend for atomic short-lived counters. It is not a
required cache or job-queue dependency, and a distributed limiter must not be
implied by selecting the Postgres response-cache backend.

### Backend selection and migration

Configuration selects `none`, `valkey`, or `postgres`. The application contract
and tests are backend-independent. Switching backends does not require cache
data migration: misses regenerate entries. The cutover sequence is:

1. introduce the `Store` contract and retain Valkey;
2. add and test the Postgres backend;
3. migrate callers to namespaces and generation invalidation;
4. run backend-parity and real-stack checks;
5. switch local and deployment configuration to Postgres;
6. remove Valkey only after rollback to the Valkey backend remains verified.

## Consequences

Positive consequences:

- cache callers no longer depend on Redis commands or key scans;
- Postgres becomes the only required stateful service;
- historical public data can remain warm until its projection generation changes;
- cache loss remains harmless and observable;
- geocoding becomes restart-safe and recoverable.

Accepted trade-offs:

- Postgres handles cache reads and writes in addition to canonical queries;
- namespace generation and cleanup add a small amount of schema and maintenance
  logic;
- a shared rate limiter is intentionally deferred for multi-instance hosting;
- Valkey remains an optional future limiter backend, not a required runtime service;
- an `UNLOGGED` cache backend is not the default because stable historical cache
  behavior is more valuable than premature WAL reduction.
