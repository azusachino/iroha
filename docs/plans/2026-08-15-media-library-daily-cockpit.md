# Media library and daily cockpit correction

Status: implementation complete; live rollout and browser verification pending

This plan follows the live library repro, the supplied Claude review, and the fresh GPT Sol-low review. It is the implementation handoff for `iroha:media-library-daily-cockpit`.

## Findings

1. The six themed media components render controlled status and completion-year selects, but their callbacks discard the selected value. The route therefore reloads with the previous filter.
2. Family changes only the list request. The aggregate request is unfiltered and is not repeated, so charts and summary cards remain static.
3. Every filter request replaces the rendered route with a loading branch. A completed view must remain visible while the next result is fetched.
4. The media Today section still exposes only exact consumption events. The canonical-history plan requires a separate dated-update surface.
5. `/media/changes` is ordered and filtered by Iroha observation time. It must not be reused for a day briefing because a provider snapshot would appear on the sync day.
6. AniList partial dates are source facts, not instants. Bangumi's current collection is a snapshot without a trustworthy change date. Neither may be promoted to a consumption event or an invented
   daily timestamp.

## Decisions

### Library filtering

`ListFilters` is the canonical scope for both `/api/v1/media` and `/api/v1/media/aggregates`. The browser sends one captured filter object and the server owns all row, chart, and summary aggregation.
The supported scope is

- `family`;
- `status`;
- `completed_year`.

Cursor and page size apply only to the list. Aggregate responses are computed for the same semantic scope without cursor pagination.

The route uses stale-while-refresh behavior: initial loading has a full loading surface; later requests retain the previous result, show a small updating indicator, and only commit a response that
matches the latest request generation.

### Daily media semantics

Today exposes two independent collections:

- `sessions`: exact rows from `tb_media_consumption_events`, each with a non-null `event_at`.
- `dated_updates`: canonical provider-state changes whose date is provable: day-precision source facts (`effective_on_precision=day`) and future exact provider activity timestamps
  (`time_basis=provider_activity`).

The daily query never uses `observed_at` as a substitute for a source date. It excludes `iroha_observed` and metadata-only `provider_recorded` rows.

Consequences for uploaded manga history:

- AniList entries with a day-precision started/completed date become dated updates for that source day and remain in the library projection.
- AniList `updatedAt` remains an ordering/provenance field until a real `ListActivity.createdAt` collector is added; it is not a daily event.
- Bangumi state changes update canonical library state/history, but a snapshot-only change does not appear on Today because Bangumi does not supply a trustworthy change date.
- Month/year-only dates remain visible in library/detail views but do not enter a specific day.
- Exact manual/playback events remain sessions and are never inferred from a provider list snapshot.

The media briefing schema becomes `media.day.v2` and includes explicit coverage (`Asia/Tokyo` plus the requested canonical date). The response shape is:

```json
{
  "sessions": { "state": "empty", "items": [], "count": 0 },
  "dated_updates": { "state": "ready", "items": [], "count": 0 },
  "coverage": { "timezone": "Asia/Tokyo", "date": "2026-08-15" }
}
```

## Implementation order

1. Add contract tests for filtered list/aggregate parity, select callback value propagation, stale refresh response ordering, and the v2 daily media shape.
2. Add a dedicated media service query for day-eligible changes. Keep general `/media/changes` observation-time semantics unchanged.
3. Update the briefing contributor, cache/schema version, web API types, Today route, fallback, and all six shared theme compositions.
4. Make all six shared media controls pass selected values. Make the library load list and aggregates together from one filter snapshot and preserve old data during refresh.
5. Run repository checks and deploy the migration/server/job/web images. The media history migration has already been applied in the local stack, so no new destructive migration is expected for this
   correction; still roll the migration job before deployments when the image contains a new migration.
6. Run AniList and Bangumi syncs, then verify with the live browser:
   - status, year, and family alter both list and charts;
   - rapid filter changes cannot commit an older response;
   - the existing surface does not disappear during refresh;
   - AniList day facts appear under Today's dated updates;
   - Bangumi snapshot-only changes do not appear on a fabricated day;
   - exact sessions remain separate and empty when no exact event exists.

## Explicit non-goals

- No frontend filtering or aggregation of the full library.
- No conversion of provider observation time into consumption time.
- No exact date fabricated from month/year-only values.
- No frontend editing workflow for provider history.
- No automatic backfill of malformed legacy event timestamps.
