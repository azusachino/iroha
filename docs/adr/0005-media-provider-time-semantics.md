# ADR-0005: Media provider time semantics

- Status: Accepted for v0.4.1 implementation
- Date: 2026-08-15
- Scope: AniList, Bangumi, media current state, media history, Today briefing
- Detailed plan: [Media provider canonical-history redesign](../plans/2026-08-15-media-provider-canonical-history.md)

## Context

The existing media importer treats provider list snapshots as `list_state` consumption events. This is not a stable temporal contract:

- AniList list entries contain fuzzy started/completed calendar dates and a last-record-update instant, not a per-session history.
- AniList separately exposes list activities with a creation instant.
- Bangumi's official subject-collection schema warns that `updated_at` does not reliably represent collection changes; episode collection timestamps may be unknown.

The current database also permits null `event_at` and the importer can fall back to import-created time. That produced false activity on the connector sync day.

## Decision

Iroha separates three canonical products:

1. `tb_media_progress`: current provider/user library state;
2. `tb_media_state_history`: append-only provider state changes with explicit observation/effective-time basis;
3. `tb_media_consumption_events`: true dated events only, with `event_at NOT NULL`.

AniList `MediaList` state does not become a consumption event. The optional AniList `ListActivity` connector, when enabled, is imported as a dated, explicitly labeled provider list update. Bangumi
snapshot diffs are recorded at Iroha observation time and never attributed to that day as consumption.

Fuzzy dates use a shared partial-date value (`YYYY`, `YYYY-MM`, or `YYYY-MM-DD`) with database precision metadata. They are never coerced into timestamps. The old progress `started_at` and
`finished_at` columns are retired. `last_update_at` remains as a provider-recorded library ordering/cursor field and is not treated as consumption time. A provider-supplied day fact such as Goodreads
`Date Read` is retained as `effective_on` with `time_basis=source_date`; it may appear in day-level facts, but it is not an exact-time consumption session.

The release includes `POST /api/v1/media/events`, which accepts an existing canonical media item, an RFC3339 event instant, an allowed event type, and an idempotency key. This is the first exact-event
producer; provider snapshots remain state/history inputs only.

The importer-derived event table is rebuilt without backfilling the two legacy synthetic rewatch rows; raw provider evidence remains available.

## Consequences

Positive:

- Today cannot report a full library sync as consumption.
- Current library state remains useful even when a provider lacks history.
- Goodreads, WeRead, Apple Books/iBooks, and Kindle can use the same model for reading status, editions, positions, annotations, and source-provided dates.
- Provider timestamp limitations are visible and testable.
- Exact event queries have a non-null, indexable temporal contract.
- AniList history can improve without weakening Bangumi correctness.

Costs:

- The API and UI need separate session/update labels.
- State history is another typed read model and needs fingerprint-based deduplication. The deduplication key is scoped to the supplying snapshot so a real state reversal is not discarded.
- AniList activity coverage is bounded and may be incomplete when activities are disabled, merged, or unavailable.
- Most reading providers expose snapshots or exports rather than an exact session feed; the product must distinguish status, date facts, and sessions.
- Existing media event queries and fixtures must be rewritten, not merely filtered.

## Rejected alternatives

### Use connector/import time for missing event times

Rejected. It creates false daily activity and makes queue/storage delay part of personal history.

### Use AniList `updatedAt` for consumption time

Rejected. It is a list-entry record update time, not proof of watching or reading.

### Use Bangumi `updated_at` for state history time

Rejected. The provider contract explicitly says not to rely on it.

### Keep null-time rows in the consumption table

Rejected. It makes a single table serve incompatible semantics and lets accidental fallback logic reintroduce false events.
