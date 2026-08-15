# Media provider canonical-history redesign

- Status: Implemented in v0.4.1 — Claude Code review blockers incorporated
- Date: 2026-08-15
- Target: v0.4.1
- Depends on: [ADR-0001](../adr/0001-provider-observations-and-canonical-records.md), [ADR-0002](../adr/0002-provider-adapter-contracts.md)
- Supersedes: the earlier assumption that every AniList/Bangumi list snapshot can become a media consumption event

## 1. Problem

The current media importer collapses three different facts into one table:

1. the provider's current library state;
2. a provider record changing;
3. an actual dated consumption event.

That is why a full AniList/Bangumi sync can look like activity on the sync day. The current adapter emits `list_state` rows without an event time, and the persistence fallback can attach the
import-created time. The current AniList conversion is also incorrect: a missing month becomes January and day zero is normalized by `time.Date`, so a year-month value can land in the wrong month or
year, not merely at a fabricated UTC midnight. Bangumi's own OpenAPI description says its collection `updated_at` does not reliably represent collection changes, so it cannot be used as a canonical
event timestamp.

Iroha needs a canonical model that is honest about what each provider proves, across anime, manga, books, ebooks, and audiobooks:

```text
provider snapshot/activity
  -> canonical media identity
  -> current library projection
  -> state-history observations
  -> true dated media events only when an exact event time exists
```

The source remains the provider snapshot/activity response and its raw-file record. Iroha owns the normalized records, temporal precision, source basis, deduplication, and read models.

## 2. User stories

- As a user, I can open Today for a day with no dated media event and see an honest empty media-sessions section; a library sync must not create fake watching or reading activity.
- As a user, I can see that a provider library record changed without the UI calling that change a watched session.
- As a user, I can see AniList list updates on their provider activity time when AniList supplies one, with a visible “AniList update” basis.
- As a user, I can see Bangumi progress and collection state in the library even when Bangumi cannot prove when the change happened.
- As a user, I can see AniList's partial started/completed dates as `YYYY`, `YYYY-MM`, or `YYYY-MM-DD` without Iroha inventing a missing month, day, or time.
- As a user, repeated full-library syncs do not create duplicate history rows or hundreds of Today entries.
- As a maintainer, a raw provider response can be reprocessed after adapter fixes without treating the re-import time as the media event time.
- As a maintainer, a provider timestamp with a documented limitation is retained as evidence but cannot silently drive a daily/monthly event aggregate.
- As a local agent or playback client, I can submit a minimal exact media event with an RFC3339 instant and idempotency key, so the exact-session surface has a real producer in this release.
- As a user, Goodreads, WeRead, Apple Books/iBooks, and Kindle records can join the same canonical book/edition library without becoming four competing book databases.
- As a user, a source-provided reading day such as Goodreads `Date Read` can appear as a day-precise completion fact, while a current reading position remains a state observation.

## 3. Provider evidence contract

| Source field                                                                            | What it proves                                                                                                     | Canonical destination                                                      | Eligible for a dated consumption event?                            |
| --------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| AniList `MediaList.progress`, `status`, `score`, `notes`, `repeat`                      | Current list state                                                                                                 | `tb_media_progress` and state history                                      | No                                                                 |
| AniList `MediaList.startedAt`, `completedAt`                                            | User-entered fuzzy calendar dates                                                                                  | Progress partial-date fields; day precision may also be a source date fact | No exact session; it has no exact time and is not a history feed   |
| AniList `MediaList.updatedAt`                                                           | When the list-entry record was last updated                                                                        | State history `provider_recorded_at`                                       | No; it is a record-update time, not proof of consumption           |
| AniList `ListActivity.createdAt`, `status`, `progress`                                  | A dated provider list update, when the activity is available                                                       | Dated provider-update history                                              | No for a “session”; yes for a separately labeled dated list update |
| Bangumi subject collection `ep_status`, `vol_status`, `type`, `rate`, `comment`, `tags` | Current collection state                                                                                           | `tb_media_progress` and state history                                      | No                                                                 |
| Bangumi subject collection `updated_at`                                                 | A provider field whose contract explicitly warns that it is unreliable                                             | Raw evidence only and optional untrusted metadata                          | No                                                                 |
| Bangumi episode collection `type`, `updated_at`                                         | Current per-episode collection state; timestamp may be `0`/unknown                                                 | Future optional episode-state history                                      | No in this release                                                 |
| Goodreads CSV shelf/status, ISBN, rating, `Date Read`                                   | Exported current library state and, when populated, a user-entered completion date                                 | Current state plus day-precise source date                                 | No exact session; a valid `Date Read` is a day fact                |
| WeRead export/API snapshot                                                              | Provider-specific current reading state; contract availability must be verified per export                         | Current state and Iroha-observed history                                   | No unless the source supplies an explicit dated activity           |
| Apple Books/iBooks library/progress/finished state                                      | Current library, position, collections, bookmarks, notes, and finished state where the local snapshot exposes them | Current state and optional annotations                                     | No stable public session feed assumed                              |
| Kindle library/progress/annotations export                                              | Current position, library identity, notes/highlights, and provider state where exported                            | Current state and optional annotations                                     | No stable session feed assumed                                     |
| Iroha `POST /api/v1/media/events` manual/Telegram/playback event with exact time        | A user/client event at a known instant and an idempotency key                                                      | `tb_media_consumption_events`                                              | Yes                                                                |

AniList's official `MediaList` reference distinguishes progress, repeat, fuzzy started/completed dates, `updatedAt`, and `createdAt`; the official `FuzzyDate` type allows incomplete dates. AniList
also exposes `ListActivity` with `status`, `progress`, and `createdAt`. Bangumi's official subject-collection schema explicitly warns not to rely on `updated_at`, and its episode-collection schema
allows `updated_at: 0` for unknown/unrecorded values.

## 4. Canonical temporal model

### Exact instants

`tb_media_consumption_events.event_at` becomes `NOT NULL`. Every row in this table represents a true dated event and has a real instant supplied by the exact-event intake endpoint, an explicit
playback source, or a future provider source that documents actual consumption timing. AniList list activity remains a dated provider update, not a session. No import time, Unix epoch sentinel, or
fabricated midnight is allowed. The release includes the minimal intake endpoint; otherwise this table would be a designed-empty sink with no producer.

The server uses the configured `Asia/Tokyo` location to derive a calendar day for day-scoped reads. The exact stored instant remains UTC in Postgres and is rendered as an RFC3339 instant only where
the API needs an instant.

### Partial calendar dates

Provider fuzzy dates are not timestamps. Iroha introduces one shared canonical partial-date value for media metadata and progress:

```json
"started_on": "2026-08"
```

The allowed wire values are `YYYY`, `YYYY-MM`, and `YYYY-MM-DD`; precision is inferred from the value and is never silently upgraded. The database stores a typed representative date plus an explicit
precision (`year`, `month`, or `day`) so range queries remain correct:

```text
started_on_value      date          -- 2026-01-01 for a year value; 2026-08-01 for a month value
started_on_precision  text          -- year | month | day
```

The representative date is an implementation value, not a claim that AniList supplied January or the first day. API serializers must use the precision when emitting and displaying it.

The same pair is used for `completed_on` and provider release dates that are fuzzy. Optional source dates may be absent; an exact media event may not be absent from `event_at`.

### Explicit time basis

Every state-history or dated-update row carries a non-empty basis:

- `manual_exact`: user supplied the event time;
- `provider_activity`: AniList `ListActivity.createdAt`;
- `provider_recorded`: a provider record timestamp such as AniList `MediaList.updatedAt`, retained for state provenance only;
- `iroha_observed`: Iroha observed a changed snapshot at fetch time;
- `source_fuzzy_date`: a provider supplied only a partial calendar date.
- `source_date`: a provider explicitly supplied a calendar date for a fact such as “Date Read”; it can participate in day-level fact views but is not an exact-time session.

Only `manual_exact` and sources that explicitly prove a consumption instant may enter the exact event aggregate. `provider_activity` may enter the separately labeled dated-update timeline, and
`source_date` may enter day-level fact views. `provider_recorded` and `iroha_observed` never masquerade as consumption time.

## 5. Canonical storage

### Current library projection

Keep `tb_media_progress` as the fast current projection. It stores status, position, unit, score-related state, repeat/play count, and partial started/completed dates in `*_on_value` plus
`*_on_precision` columns. The old `started_at` and `finished_at` timestamp columns are retired; they were capable of turning a fuzzy provider date into a false instant. `last_update_at` remains a
deliberate provider-recorded ordering field for library pagination and cursor stability. Its ordering role is not a consumption claim; the same timestamp is retained as `provider_recorded_at` in state
history. `updated_at` means when Iroha refreshed the projection. The projection remains directly readable without a provenance join.

`repeat` is retained as a current count. It does not generate one synthetic `rewatch` row per count because the provider does not supply the dates of those rewatches.

### True media events

Keep the public table name `tb_media_consumption_events` for compatibility, but rebuild it from scratch:

```sql
event_at       timestamptz not null
event_type     text        not null
source_kind    text        not null
source_event_id text       not null default ''
raw_file_id    uuid
```

The table contains only actual dated media events. `list_state` is not a valid event type. `event_at` has no nullable compatibility path. A partial date from AniList is stored on the projection, not
coerced into this table.

The v0.4.1 intake accepts these event types: `started`, `progressed`, `completed`, `finished`, `read`, `watched`, `listened`, `reread`, `rewatched`, `abandoned`, `paused`, `reopened`, `rated`,
`noted`, and `bookmarked`.

Use a partial unique index for non-empty `(source_kind, source_event_id, event_type)` values. Events without a provider/client identity are append-only and are deduplicated only when the explicit
intake layer supplies an idempotency key.

### Provider state history

Add `tb_media_state_history` as a canonical, append-only history of provider state observations. It is not a consumption-session table.

Required fields:

```text
id
media_item_id
source_kind
source_event_id
observed_at             timestamptz not null
effective_at            timestamptz null
time_basis              text not null
change_kind             text not null  -- snapshot, changed, removed, provider_activity
state_fingerprint       text not null
status, unit, position, total, progress_percent
rating, rating_scale, note, repeat_count
started_on_value/precision
completed_on_value/precision
effective_on_value/precision
provider_recorded_at    timestamptz null
raw_file_id             uuid null
created_at              timestamptz not null
```

The typed state columns are the canonical read contract; raw JSON remains evidence/debug material, not the only representation. `effective_at` is populated only for an exact provider activity.
`effective_on_value/precision` is populated for a source-provided calendar fact such as Goodreads `Date Read` or an AniList day-precision completion date. Snapshot diffs always have `observed_at`; a
Bangumi sync therefore remains truthful without inventing a user-action time.

Append a row only when the canonical state fingerprint changes. The idempotency index is scoped to the supplying raw snapshot, so a legitimate A -> B -> A transition remains representable. A
successful complete snapshot may append a `removed` row for a provider item that disappeared, but partial/failed connector runs must never reconcile missing rows.

### Raw evidence and observation time

The connector runner records one `observed_at` for each fetched snapshot before storing the raw file and passes it through the import snapshot. `tb_raw_files.created_at` remains file-storage time; it
is not substituted for source observation time. Manual uploads without a source observation timestamp use their ingestion time only as the Iroha-observed time basis. This avoids making later queue
delay part of connector provider history.

## 6. Connector redesign

### AniList current-list connector

Keep the current `MediaListCollection` connector for the library projection. Change its adapter behavior:

- map `startedAt`, `completedAt`, and `media.startDate` to the shared partial-date type;
- retain `updatedAt` as provider-recorded metadata only;
- emit a typed current-state observation, not a `list_state` consumption event;
- retain `repeat` only on current progress;
- never synthesize `rewatch` events;
- include the source entry ID and state fingerprint for idempotent history.

### AniList activity connector

Implementation status (2026-08-15): enabled in the AniList sync path. The worker resolves the configured username to an AniList user ID, fetches a bounded 365-day `ListActivity` window, preserves each
response as raw evidence, and writes dated `provider_activity` history rows. The current-list sync remains independent and still supplies the library projection.

The separate connector/source kind for `Page.activities` is filtered to the user's `ListActivity` records. It resolves the user ID once, fetches by a bounded created-at window, and retains a 24-hour
overlap cursor for incremental runs. It deduplicates by AniList activity ID and preserves each raw GraphQL response.

The adapter may map an activity's `status` and `progress` when the progress text is unambiguous. It must preserve the original progress string and refrain from inventing a numeric position for formats
it cannot parse. These rows are `provider_activity` dated list updates, not watched/read sessions. If the activity feed is unavailable, private, disabled, merged, or incomplete, the current-list
connector still works and only provides state history.

First synchronization has an explicit bounded backfill window (default: the previous 365 days, configurable). It reports its coverage; it does not claim that older history exists. Incremental
synchronization overlaps the previous 24 hours to handle activity ordering/merging and remains idempotent by activity ID.

### Bangumi connector

Keep subject collection sync for current state and state-history diffs. Do not use subject `updated_at` as an event time or as the ordering key. Store the raw value only as untrusted provider
metadata. Do not fetch every episode collection in the normal sync: it adds request fan-out but still cannot reconstruct history because the endpoint is current state and allows unknown timestamps. An
explicit future detail sync may add episode-level state, but it will use `iroha_observed` unless the provider contract changes.

### Future reading providers

The media contract is intentionally provider-neutral so reading sources do not require a second domain model:

| Source                       | Initial adapter shape                         | Canonical facts                                                                                                             | Deliberate limitation                                                                              |
| ---------------------------- | --------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Goodreads                    | Versioned CSV import                          | ISBN/title identity, exclusive shelf, rating, tags, notes, position if supplied, `Date Read` as a day-precision source fact | No exact reading-session history; custom shelves remain source labels                              |
| WeRead                       | Export-first snapshot adapter                 | Book identity, shelf/progress/rating/notes when present, observed state changes                                             | Do not make undocumented cookie-backed web endpoints a core connector; no assumed event history    |
| Apple Books (`ibooks` alias) | Local/export snapshot adapter                 | Book/audiobook/PDF identity, collections, current position, finished state, bookmarks/notes when the snapshot exposes them  | Apple documents iCloud synchronization, not a stable public history API; no invented session times |
| Kindle                       | Amazon personal-data or local artifact import | ASIN/ISBN identity, current position, library state, highlights/notes, completion state where present                       | Provider sync backs up position/annotations but does not establish a stable per-session feed       |

Source IDs are aliases at the intake boundary only: canonical provider IDs are `goodreads`, `weread`, `apple_books`, and `kindle`. `ibooks` normalizes to `apple_books`.

Book identity follows the existing work/item split. An ISBN-10/ISBN-13, ASIN, provider book ID, or Apple Books asset ID becomes an external reference. A known ISBN/ASIN attaches to an edition/item; a
title-only row creates an unresolved resolution task rather than silently merging two editions. A reading status belongs to the concrete item that the provider reports, with an optional later rollup
to the parent work.

The shared status/unit vocabulary is:

```text
status: planned | in_progress | completed | paused | abandoned | unknown
unit:   pages | chapters | volumes | seconds | percent | locations | position
```

Provider shelves and custom labels remain in source state for lossless display. Only documented exclusive shelves map to the shared status enum. A current position is not a progress event; a valid
source completion date is a day fact; an exact manual capture is a true event.

## 7. Read/API contract

### Media library

`GET /api/v1/media` and detail responses expose current state and partial dates:

```json
{
  "status": "completed",
  "position": 12,
  "unit": "episodes",
  "started_on": "2026-08",
  "completed_on": "2026-08-13",
  "repeat_count": 1,
  "source": "anilist"
}
```

No API response emits a fabricated `yyyy-MM-dd HH:mm:ss` for a fuzzy source date.

### Media events

`GET /api/v1/media/events` returns only `tb_media_consumption_events` rows and therefore only rows with a non-null exact `event_at` (the database constraint makes the filter structural). It is the
source for dated consumption charts and Today’s “media sessions”. `POST /api/v1/media/events` is the minimal exact-event producer. It requires an existing canonical `media_id`, an allowed event type,
an RFC3339 `event_at`, and a non-empty `idempotency_key`; the key is stored as `source_event_id`. Repeated requests with the same `(source_kind, idempotency_key, event_type)` are idempotent, while a
different payload for an existing key is rejected.

### Provider changes

Add a separate read path, either `GET /api/v1/media/changes` or a clearly named state-history subresource. It returns:

```json
{
  "change_kind": "provider_activity",
  "time_basis": "provider_activity",
  "effective_at": "2026-08-14T13:21:00Z",
  "effective_on": null,
  "date_precision": null,
  "observed_at": "2026-08-14T13:21:04Z",
  "source_kind": "anilist",
  "title": "…",
  "position": 5,
  "progress_percent": null
}
```

For a Goodreads completion fact, `effective_at` is absent and the response instead contains `effective_on: "2026-08-13"`, `date_precision: "day"`, and `time_basis: "source_date"`. For a Bangumi
snapshot diff, both effective fields are absent and `time_basis` is `iroha_observed`; it is available in library/source history but is not placed in a day-scoped consumption aggregate.

### Today briefing

The fixed `media` section has distinct fields:

```json
{
  "sessions": { "state": "empty", "items": [], "count": 0 },
  "dated_updates": { "state": "ready", "items": [], "count": 0 },
  "coverage": { "timezone": "Asia/Tokyo", "date": "2026-08-15" }
}
```

`sessions` contains only true events. `dated_updates` contains exact AniList activity updates and day-precise source facts such as Goodreads `Date Read` when available, each with an explicit basis and
label. Snapshot-only Bangumi changes do not appear as if they happened on the sync day; they are visible through library/source history. Expenses remain a separate fixed briefing section and are
filtered by their canonical `occurred_on` date.

The UI labels must use “Media sessions” and “Media updates”; never “watched today” for a state observation.

### Existing read-model mapping

The redesign does not leave old queries semantically ambiguous:

| Existing read model                                      | Canonical source after v0.4.1                                                                                                                                                 |
| -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Library item status/position/rating/order                | `tb_media_progress` plus its retained `last_update_at` cursor field                                                                                                           |
| Library completion year/facet                            | `tb_media_progress.completed_on_value` only when `completed_on_precision = 'day'`, plus exact `finished`/`completed` events                                                   |
| Period report `event_count`, rated count, average rating | exact `tb_media_consumption_events` only; provider state ratings are exposed through state history, not session totals                                                        |
| Period report completed items and media metric series    | exact completion events plus day-precision source completion facts when the report explicitly opts into facts; month/year fuzzy dates are excluded from day-scoped completion |
| `/media/events` and Today media sessions                 | exact `tb_media_consumption_events` only                                                                                                                                      |
| `/media/changes` and provider-update timeline            | `tb_media_state_history`, including `provider_activity`, `source_date`, and `iroha_observed` labels                                                                           |
| Canonical date coverage                                  | exact event dates and explicitly day-precise source facts; provider observation time does not add a media date                                                                |

This makes the post-rebuild change explicit: provider library state no longer contributes a media date to the canonical daily-date union. Activity, sleep, daily health, and expense contributors remain
unchanged.

### Cache contract

The server read-cache key version is bumped with the wire-contract change. Media events, media changes, library/detail, aggregates, period reports, metric series, briefing, and canonical date coverage
share the media write invalidation boundary. A successful exact-event write or completed media import invalidates that boundary after commit; the state-history changes endpoint is under the same
`/api/v1/media` namespace, so stale `X-Iroha-Cache: HIT` responses cannot survive a media write.

## 8. Migration and data handling

The user approved rebuilding the importer-derived event table. The migration will:

1. drop `tb_media_consumption_events`;
2. create it again with `event_at NOT NULL` and the exact-event constraints;
3. create `tb_media_state_history` with raw-snapshot-scoped fingerprint idempotency;
4. retire `tb_media_progress.started_at` and `finished_at`; add partial-date value/precision columns and constraints;
5. add `tb_raw_files.observed_at` and carry connector observation time into `tb_import_snapshots.taken_at`;
6. update GORM models, adapters, import persistence, media read queries, briefing contributors, contracts, and the exact-event intake;
7. bump the read-cache key version and invalidate the media change namespace on import/event writes;
8. re-run AniList/Bangumi current-state imports;
9. run the bounded AniList activity backfill only after the activity connector is enabled;
10. verify that repeated syncs produce no new state-history rows when the fingerprint is unchanged, while a state reversal remains visible.

The old two synthetic `rewatch` rows are intentionally discarded. The current provider snapshots do not contain their exact event times, and the raw connector responses remain available for future
parser work. No guessed backfill is allowed.

This migration is intentionally destructive and cannot restore the discarded derived rows on rollback. Before applying it, retain the raw provider files and verify the database backup; rollback
restores the old table shape only, not the old synthetic event data. Deployment deletes the completed migration Job, recreates it with the new image, waits for successful completion, and only then
rolls the server and worker Deployments.

## 9. Verification and acceptance criteria

### Adapter tests

- AniList year-only, year-month, and full dates preserve precision and serialize to `YYYY`, `YYYY-MM`, and `YYYY-MM-DD`.
- An AniList missing month/day never becomes January/day zero or a UTC timestamp.
- AniList current-list parsing emits projection/state observations but no `list_state` or synthetic `rewatch` consumption events.
- If the optional AniList activity connector is enabled, its fixtures map exact `createdAt` and preserve unparseable progress text; the base release does not claim activity coverage until that
  connector is enabled.
- Bangumi subject `updated_at` is never used as `event_at` or the state-history effective time.
- Bangumi episode `updated_at=0` stays unknown.

### Persistence/integration tests

- a full provider snapshot creates current progress and state history, but zero consumption events;
- a changed snapshot creates exactly one state-history row;
- an unchanged snapshot creates no additional state-history row;
- a complete snapshot removal creates a removal observation; a failed/partial run does not;
- true exact events persist with non-null `event_at` and appear in event reads;
- state observations with unknown effective time never appear in day-scoped consumption reads;
- dropping/recreating the derived event table leaves no legacy null-time or synthetic rewatch row;
- reprocessing the same raw file is idempotent.
- the exact-event endpoint creates one event, accepts a retry idempotently, and rejects a conflicting retry;

### API/browser checks

- `/today` for 2026-08-15 has zero media sessions when the database has no exact event for that Tokyo day;
- a dated AniList activity appears under “Media updates”, not “Media sessions”;
- a Goodreads day-precise completion appears under “Media updates” as a source date, not as a timestamped session;
- a Bangumi sync-only change does not appear as consumption on the sync day;
- all date strings in library, detail, event, changes, and briefing responses follow the canonical contract;
- current library cards still show progress and fuzzy dates after the event-table rebuild;
- year/month fuzzy completions do not enter day- or month-scoped exact completion aggregates; day-precision facts are labeled as facts;
- library pagination remains stable because `last_update_at` remains an ordering projection even though it is not consumption time;
- a media write changes the read-cache generation and the next media/briefing/report read is a miss;
- migration deployment deletes the completed migration Job before recreating it, waits for success, and only then rolls the affected server/job Deployments;
- six theme compositions render the same semantic sections while each theme owns its visual treatment.

## 10. Explicit non-goals

- Reconstructing per-episode/per-chapter consumption sessions from a current provider list.
- Treating Bangumi's unreliable `updated_at` as a historical clock.
- Treating AniList `repeat` as dated rewatch history.
- Fetching Bangumi episode collections on every normal sync.
- Treating Goodreads `Date Read`, Apple Books finished state, or Kindle position as an exact reading session.
- Adding frontend editing of provider state to this redesign.
- Claiming complete AniList activity history beyond the configured backfill coverage.

## 11. Approved decisions

Claude Code reviewed these decisions and approved them; they are implementation constraints, not open questions:

1. AniList `ListActivity` becomes an optional source of dated **provider list updates**, but not of consumption sessions.
2. Bangumi snapshot diffs are canonical state history with `iroha_observed` timing and are excluded from day-scoped consumption aggregates.
3. Goodreads/WeRead/Apple Books/Kindle use the same state-history model; a source-provided reading day is a day fact, never an invented instant.

## 12. Research references

- [AniList MediaList reference](https://docs.anilist.co/reference/object/medialist)
- [AniList FuzzyDate reference](https://docs.anilist.co/reference/object/fuzzydate)
- [AniList ListActivity reference](https://docs.anilist.co/reference/object/listactivity)
- [AniList Query reference](https://docs.anilist.co/reference/query)
- [Bangumi user subject collection schema](https://raw.githubusercontent.com/bangumi/server/master/openapi/components/user_subject_collection.yaml)
- [Bangumi user episode collection schema](https://raw.githubusercontent.com/bangumi/server/master/openapi/components/get-user-episodes-collection.yaml)
- [Apple Books support: read books and collections](https://support.apple.com/guide/ipad/read-books-ipadc8494b6b/26/ipados/26)
- [Apple Books support: iCloud reading position and annotations](https://support.apple.com/en-gb/guide/books/ibks86dab303/mac)
- [Amazon Kindle support: sync position, notes, and highlights](https://digprjsurvey.amazon.co.uk/csad/help/node/GGFEXXS8Z7DPJSTN?theme=light)

Goodreads export column behavior and WeRead artifact formats are treated as source-format research, not as stable API contracts. Their adapters must begin with fixture-backed imports and versioned
parsers.

The rest of this document follows from those three decisions and the already-approved clean rebuild of the derived event table.
