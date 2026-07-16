# Cockpit API redesign

Status: implemented, runtime verification pending, 2026-07-14

## Why the current root route is unacceptable

`harus-macmini.har` captures the root cockpit loading against the local stack. The capture contains 61 API requests, 2.32 MB of response bodies, and 34.1 seconds of cumulative request time.

| Endpoint                   | Requests | Response bytes | Cumulative time |
| -------------------------- | -------: | -------------: | --------------: |
| `/api/v1/daily`            |       11 |         555 KB |           9.3 s |
| `/api/v1/sleep`            |       19 |         850 KB |          11.2 s |
| `/api/v1/activities`       |        5 |         315 KB |           2.1 s |
| `/api/v1/media/events`     |       24 |         565 KB |          11.1 s |
| `/api/v1/media`            |        1 |          37 KB |           0.3 s |
| `/api/v1/media/aggregates` |        1 |         0.7 KB |           0.1 s |

The root page is the cause: it starts four unbounded history sweeps and paginates each resource at 100 rows. The selected day only needs a handful of rows, so the initial request shape is inverted.
Domain pages may explore history; the cockpit should read one bounded day.

## Implemented contract

Add:

```text
GET /api/v1/briefing?date=YYYY-MM-DD
```

The date is a UTC calendar day. The response is an ordered registry of versioned sections so future modules do not require a growing fixed top-level struct:

```json
{
  "date": "2026-07-14",
  "previous_date": "2026-07-13",
  "next_date": "2026-07-15",
  "sections": [
    { "key": "daily", "schema": "daily.day.v1", "state": "ready", "data": { "items": [] } },
    { "key": "media", "schema": "media.day.v1", "state": "empty", "data": { "items": [], "has_more": false } }
  ]
}
```

Each contributor owns its section key/schema and returns `ready` or `empty`; query failures become `unavailable` for that section so one domain does not take down the cockpit. The current contributors
are `daily`, `sleep`, `activities`, and `media`, each capped at 20 rows. A domain page remains responsible for full history and pagination. `previous_date` and `next_date` are calendar navigation, not
a historical availability index, so empty days remain selectable without a history sweep.

The endpoint queries each domain with `[date 00:00 UTC, date + 1 day)` predicates and does not call HTTP list endpoints or perform cursor pagination internally. Empty domains are successful empty
sections, not errors. Go keeps typed contributors; the wire envelope is extensible and the web ignores unknown section keys.

## Media query semantics

Media list reads should accept explicit filters:

```text
/api/v1/media?family=anime&status=in_progress&completed_year=2025
```

`family` is the coarse grouping (`anime`, `manga_book`, `game`); `media_type` remains the granular value. `status` is a stable backend enum (`in_progress`, `completed`, `planned`, `abandoned`,
`unknown`). `completed_year` filters completion/progress history, not release metadata. Add `release_year` only when the UI needs that distinct meaning.

The API returns facts, not presentation colors. The web layer owns status tokens and maps them to accessible color-plus-label treatments. `reading`/`watching` is a UI verb derived from `media_type`
and `unit`, not a second backend status.

## Non-goals

- Do not make the root route a general-purpose aggregate endpoint.
- Do not return provider payloads or provider-specific color names.
- Do not solve key-visual hosting in this redesign; `cover_image_url` remains an optional derived field.
- Do not remove pagination from `/media`, `/activities`, `/sleep`, or `/daily` domain pages.

## Acceptance checks

1. Root initial load uses one briefing request plus static assets, not history pagination.
2. A day with no data returns 200 with empty/null sections.
3. The response is bounded by date and documented caps.
4. Media status/family/completion-year filters have contract tests.
5. Source-level verification shows the root uses one `getBriefing` call and no history sweep; Go tests and `bun run check` pass.
6. A replacement HAR is still required against the running local stack to measure real request count and bytes; the original 61-request/2.32 MB capture remains the baseline.
