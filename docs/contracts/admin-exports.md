# Private admin exports

Status: implementation contract for issue #85; CLI-only, not public publishing or a restore guarantee.

## Commands

- `iroha-admin export ndjson <domain>` streams one domain to stdout. Domains: `activities`, `sleep`, `daily`, `media`, `expenses`, `tasks`.
- `iroha-admin export gpx <act_id>` streams one activity's untrimmed canonical route as GPX 1.1. IDs use the existing `act_` encoding.

Use the same database configuration and operator access as other admin commands. No HTTP endpoint or browser permission is added. Outputs contain sensitive personal data, exact coordinates and timestamps. Store them privately (`umask 077` before redirecting stdout), never in the public-site directory. Diagnostics go to stderr; interrupted or failed output is incomplete and must be discarded. A successful process exit is required before accepting an artifact.

## NDJSON v1

Each line is `{"version":1,"domain":"activities","table":"tb_activities","record":{...}}`. Records preserve database column names, UUID references, nulls, timestamps, integer currency amounts, source identifiers and JSON values. Child tables follow their parent tables; row ordering within a table is unspecified. Geography's duplicated `geom` column is omitted; latitude/longitude remain lossless. An empty domain is an empty stream.

Domain contents:

- Activities: activities, external references, route points, samplings, laps.
- Sleep: sessions and segments.
- Daily: summaries and metrics.
- Media: works, items, titles, relations, external references, creators/roles, events, progress, state history, lists/items and matching decisions.
- Expenses: expenses (including tombstones), statements and statement rows.
- Tasks: personal tasks.

A domain export uses one read-only repeatable-read snapshot. Rows are written incrementally, not collected into a whole-domain array. Domains exported in separate commands can have different snapshots. Only the fixed domain/table allowlist is accessible: credentials, sessions, passkeys, configuration, job payloads, caches and raw evidence bytes are excluded. This is a canonical-record export, not a database backup or a replacement for retained evidence/replay. Column changes follow SQL migrations; the envelope version must change for an incompatible envelope change. No automatic import or destructive restore is supplied.

## GPX 1.1

GPX includes the activity title, sport and canonical route points in sequence order; each point retains coordinates, optional elevation and optional UTC timestamp. No privacy trimming, rounding or fabricated samples. Points without a complete coordinate pair are omitted; an out-of-range/non-finite coordinate or elevation fails the export. An existing activity without route points yields an empty track segment; a missing or malformed ID fails. GPX does not encode heart rate, laps or other measurements; those remain in NDJSON.

## Acceptance and verification

1. Synthetic integration fixtures decode NDJSON with the standard JSON decoder, preserve nulls/UUID references/child rows, and exclude `geom` and credential tables.
2. A standard XML decoder reads generated GPX with escaped titles, coordinates, elevation and timestamps; exported route endpoints equal the private canonical fixture.
3. Invalid domains/IDs cannot select arbitrary SQL objects; nonexistent activities fail. Cancellation and writer errors propagate without success.
4. `make test`, `make test-integration-ci` against disposable PostGIS, and `make validate` pass. No live-data export is required or authorized.
