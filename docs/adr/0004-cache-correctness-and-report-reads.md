# ADR 0004: Cache correctness and report reads

- Status: Accepted
- Date: 2026-08-14
- Target release: v0.4.1
- Depends on: [ADR 0001](0001-provider-observations-and-canonical-records.md), [ADR 0003](0003-cache-backends-and-invalidation.md)

## Implementation status

The decision is implemented on the v0.4.1 branch. `apps/iroha-runtime/cache` is the canonical backend-neutral cache module: Postgres is the default backend, Valkey is supported for the k3s deployment,
and `none` is the explicit disabled mode. A production process-memory backend is deliberately not provided because it would be lost on restart and would create a second, unshared cache behavior.
Report caching, direct expense GET caching, generation-safe population, mutation invalidation, degraded bypass, bounded cleanup, and the release performance gate are wired. The current branch also
adds server-owned activity/sleep overview projections, daily date coverage, compact report-series data, and consistent half-open date boundaries.

## Context

Iroha v0.4 already has a backend-neutral, generation-based response cache for successful JSON reads. The live k3s deployment uses Valkey with a 24-hour safety TTL. Daily, monthly, and yearly domain
reads can become cache hits after their first request.

The earlier cache audit found two gaps: reports were not cached, and expense mutations did not invalidate derived reads. The current implementation closes both gaps and also places direct expense GET
reads in `read_expenses`, so the ledger and its metric projections share the same post-commit freshness boundary.

The existing daily tables are canonical imported health facts. They are not a generic derived rollup table, and v0.4 has no aggregation worker or active aggregation schedule. A clock-only job that
aggregates yesterday would be incorrect because imports, corrections, and expense edits can change historical dates.

The first principle remains unchanged:

> Canonical Postgres records are authoritative. Cache entries and future read models are disposable representations.

## Decision

### 1. Use cache-first optimization for v0.4.1

v0.4.1 will make repeated reads cheap without adding a durable daily/monthly/yearly aggregation table. The release will:

1. keep canonical domain tables as the source of truth;
2. cache complete monthly and 12-month report responses, using the compact `monthly-report-series.v2` representation;
3. repair invalidation for every canonical mutation path;
4. make cache identity include every response-affecting interpretation;
5. make cache population safe across namespace-generation changes;
6. add bounded cleanup for disposable Postgres cache rows;
7. measure cold requests on a larger deterministic fixture before deciding whether a derived daily read model is necessary.

### 2. Add a report response namespace

Add the read_reports namespace to the allowlisted HTTP response cache. It covers:

- GET /api/v1/reports/monthly;
- GET /api/v1/reports/monthly-series.

The complete HTTP response is cached, including empty-month information and the generated report envelope. A report error or non-JSON response is never cached. Direct expense list and detail endpoints
are also successful canonical GET reads and are cached under `read_expenses`; expense create, replace, and delete remain uncached mutations and invalidate `read_expenses`, `read_metrics`, and
`read_reports`.

The report cache is lazy. v0.4.1 does not prewarm yesterday, the current month, or the current year. Prewarming and request coalescing remain measurement-driven follow-ups.

### 3. Preserve the freshness contract

All private response-cache namespaces use generation invalidation plus the existing 24-hour safety TTL:

- canonical writes invalidate the affected namespace after the transaction commits;
- the TTL is a recovery boundary, not the normal freshness mechanism;
- stale-while-revalidate is not enabled;
- current and partial periods use the same rule as historical periods;
- generated_at remains the time the response was computed;
- X-Iroha-Cache remains the operational HIT/MISS diagnostic.

The cache is not allowed to become a second canonical store. Losing or bypassing it must fall back to the canonical loader.

### 4. Centralize dependency invalidation

The application layer owns a small dependency-aware invalidation coordinator. Domain services do not know Valkey commands or backend-specific key patterns. HTTP and worker wiring uses the same
coordinator after successful canonical commits.

The v0.4.1 dependency map is:

| Canonical mutation                                         | Invalidate                                                     |
| ---------------------------------------------------------- | -------------------------------------------------------------- |
| Activity, sleep, daily-health, or media import completion  | Existing affected read namespaces, metrics, and reports        |
| Expense create, replace, or delete                         | Expenses, metrics, and reports                                 |
| Media resolution that changes canonical media presentation | Media and reports                                              |
| Geocode refresh that changes route enrichment              | Activities                                                     |
| Future canonical mutation                                  | Explicit dependency entry required before the write path ships |

Invalidation is namespace-based. No caller may scan backend keys or delete individual physical keys.

### 5. Handle invalidation failure explicitly

After a successful canonical commit, the coordinator retries namespace invalidation with a bounded backoff. If invalidation still fails:

1. the canonical write remains committed;
2. the affected namespace is marked degraded in the running process;
3. reads bypass that namespace until a successful invalidation clears the degraded state;
4. a structured operational error is logged and counted;
5. the next successful invalidation restores normal cache use.

This keeps a temporary cache outage from turning a known-stale cached representation into a normal response. A durable invalidation outbox is deferred until multi-instance hosting makes the in-memory
degraded state insufficient.

### 6. Make cache identity complete

The logical cache identity must contain:

- HTTP method and normalized path;
- canonical encoded query parameters;
- effective IANA timezone for date-sensitive reads, including the server default when the request omits it;
- response representation version;
- metric aggregation version where a metric-series definition affects the result.

The user-facing period controls do not need to expose timezone. The server normalizes the effective timezone into the cache identity. A cache-key version bump accompanies this contract change so old
entries cannot collide with the new interpretation.

### 7. Make generation changes race-safe

A response loaded under generation N may only be stored under generation N. Cache population must compare the generation captured at lookup with the current generation before writing. An invalidation
that advances the generation must therefore prevent an in-flight old response from becoming a hit in the new generation.

This rule applies equally to Valkey and Postgres implementations and must be covered by deterministic concurrency tests.

### 8. Keep aggregation materialization deferred

v0.4.1 does not add a generic daily aggregation job, report table, or yearly report endpoint.

Before introducing a durable read model, the release candidate will benchmark:

- a 10-year canonical fixture;
- at least 100 expenses per month;
- representative activity, sleep, daily-health, and media records;
- cold monthly report and 12-month report-series requests;
- repeated cache-hit requests.

A future read model is justified only if the cold-request p95 exceeds 500 ms or database work becomes visibly expensive. If that gate fails, the next design must use dirty date ranges triggered by
canonical writes plus periodic repair. It must store metric-specific sufficient statistics rather than blindly averaging daily averages.

### 9. Maintain the Postgres cache backend

Postgres cache entries remain disposable. v0.4.1 adds a bounded maintenance operation for expired entries and entries from old generations. Cleanup is opportunistic or explicitly invoked and must not
become an aggregation schedule or an unbounded table scan.

## Consequences

Positive consequences:

- repeated domain and report reads avoid repeated aggregation;
- canonical writes have an explicit freshness path;
- the same invalidation policy applies to HTTP and worker mutations;
- timezone and aggregation interpretation cannot silently collide in cache keys;
- an invalidation outage bypasses known-stale cache data;
- the future materialization decision is based on representative measurements.

Accepted trade-offs:

- the first request after invalidation still computes synchronously;
- the report cache stores a complete response rather than independently caching each month;
- v0.4.1 does not provide background report delivery or a yearly report envelope;
- a single-process degraded namespace state is sufficient for the supported single-server deployment;
- a durable invalidation outbox remains future work for multi-instance operation.

## Rejected alternatives

- **Aggregate only yesterday:** rejected because historical imports and corrections are valid.
- **Make the cache the source of truth:** rejected because cache loss must never lose personal data.
- **Cache only the currently visible frontend rows:** rejected because Iroha owns complete aggregation.
- **Add a report table immediately:** rejected until the larger fixture proves response caching is insufficient.
- **Use stale-while-revalidate by default:** rejected because a known-stale personal report is worse than a slower canonical read.
