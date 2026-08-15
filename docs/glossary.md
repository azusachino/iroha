# Iroha glossary

This glossary defines terms used by the canonical-data, cache, and report contracts.

## Canonical record

An authoritative personal-data record owned and validated by Iroha, such as an activity, sleep session, daily health fact, media event, or expense. Canonical records remain in Postgres and are not
reconstructed from frontend rows or disposable cache entries.

## True media event

An append-only media record with a real, exact event instant. Its database `event_at` is required and is never filled with importer time, a provider record-update time, or a fabricated midnight.

## Media state observation

A canonical record of a provider's current library state as observed by Iroha. It may include a provider-recorded timestamp or an exact provider activity timestamp, but it is not automatically a
consumption session.

## Partial canonical date

A calendar value with explicit precision: `YYYY`, `YYYY-MM`, or `YYYY-MM-DD`. It preserves what a provider actually supplied. A database representative date is never displayed without its precision.

## Time basis

The declared meaning of a timestamp or calendar date: `manual_exact`, `provider_activity`, `provider_recorded`, `iroha_observed`, `source_date`, or `source_fuzzy_date`. Date-scoped consumption
aggregates accept only bases that prove a real dated event; a `source_date` is a day-level fact, not an exact-time session.

## Response cache

A disposable cache of a complete successful HTTP representation. It may reduce repeated work but cannot become the source of truth. A cache miss, expiry, backend outage, or invalidation bypass must
fall back to canonical data.

## Canonical cache module

The backend-neutral runtime cache facade used by Iroha read paths. It supports Postgres, Valkey, and an explicit disabled mode through the same generation-aware contract; it does not make a
process-memory map a production source of responses. Every backend stores disposable representations only.

## Cache cleanup

A bounded maintenance operation that removes expired or superseded-generation cache entries. It is housekeeping for disposable data, not a canonical-data aggregation job.

## Cache namespace

A logical invalidation boundary such as read_metrics or read_reports. Namespace invalidation advances a generation; callers do not delete backend-specific physical keys.

## Namespace generation

The version of a cache namespace that identifies which entries were created before or after the latest canonical change. A response from an old generation must not be served or written into a new
generation.

## Cache identity

Every input that can change a response: method, path, normalized query, effective timezone, representation version, and applicable aggregation version. Omitting one can make two different
representations collide.

## Effective timezone

The IANA timezone actually used by the server to resolve calendar periods. It may come from the request or the configured server default. It belongs in date-sensitive cache identity even when the
client does not place it in the URL.

## Invalidation

A post-commit operation that advances one or more cache namespace generations after canonical data changes. Invalidation is a freshness contract, not a deletion of canonical records.

## Degraded namespace

A namespace whose invalidation failed after a canonical write. Reads bypass its disposable cache until a later invalidation succeeds, preventing known-stale entries from being served normally.

## Read model

A derived representation optimized for reading. It is not canonical data and needs its own source generation, aggregation version, coverage, rebuild, and freshness contract.

## Dirty date range

A calendar range that may have changed because of an import, correction, or backdated mutation. A future read-model job must rebuild dirty ranges rather than only processing the wall-clock previous
day.

## Sufficient statistics

The stored values required to recompute a metric correctly at a larger grain. For example, an average may require a sum and observation count; averaging daily averages alone may be mathematically
wrong.

## Report envelope

The stable cross-domain response containing the resolved period, generated_at, section schemas, available/empty state, and domain-specific data. Caching the envelope preserves empty months and
completeness metadata.
