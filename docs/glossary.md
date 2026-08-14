# Iroha glossary

This glossary defines terms used by the canonical-data, cache, and report contracts.

## Canonical record

An authoritative personal-data record owned and validated by Iroha, such as an activity, sleep session, daily health fact, media event, or expense. Canonical records remain in Postgres and are not
reconstructed from frontend rows or disposable cache entries.

## Response cache

A disposable cache of a complete successful HTTP representation. It may reduce repeated work but cannot become the source of truth. A cache miss, expiry, backend outage, or invalidation bypass must
fall back to canonical data.

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
