# Frontend request audit

Date: 2026-08-04

This audit covers every top-level Svelte route plus sampled activity, media, and sleep detail routes. The baseline was captured with headless Chromium against `https://iroha.h.azusachino.icu` before
the control-room request fix. The count is the initial API traffic after navigation settles; cursor pages are counted individually.

## Findings

| Route           | Baseline | Decision                                                                                                                                                                          |
| --------------- | -------: | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/`             |        3 | Keep: briefing, day-index lookup for the scrubber, and the To-go strip are separate visible concerns.                                                                             |
| `/overview`     |        6 | Keep for now: summary plus five activity pages power the heatmap/streak/recent-history view. Route footprint remains opt-in and made no initial request.                          |
| `/motion`       |        2 | Keep: one visible activity page and one summary used by filters/statistics.                                                                                                       |
| `/patterns`     |       13 | Fixed: initial load no longer sweeps all daily history or fetches yearly aggregates; Day loads one selected month (maximum 31 rows), Year loads one aggregate only when selected. |
| `/design`       |        1 | Keep: the design lab requests the live briefing when available and otherwise uses its sample content.                                                                             |
| `/library`      |        2 | Keep: one library page and one aggregate set are both visible on first paint. Filter changes reload only the library page.                                                        |
| `/night`        |        4 | Keep: session list, year/month trends, and the selected session's visible stage timeline. Both trend granularities are visible simultaneously.                                    |
| `/to-go`        |        3 | Fixed: one task request replaces separate open/completed requests; jobs are scoped to top-level media-sync actions.                                                               |
| Activity detail |        4 | Keep: activity, route, heart-rate samples, and laps are independent detail panels.                                                                                                |
| Media detail    |        1 | Keep: one detail request.                                                                                                                                                         |
| Sleep detail    |        2 | Keep: session metadata and stage segments are both visible.                                                                                                                       |

## Control-room job propagation

The two sync buttons enqueue durable `media_sync_anilist` and `media_sync_bangumi` jobs. Each connector run then fans out into many `media_intake_parse` jobs. Those are worker/import jobs, not
personal tasks; the old UI displayed them in the same unfiltered list, making two clicks look like a runaway task list. The control room now requests and displays only the two top-level sync kinds and
explains that child importer work is tracked outside the personal queue.

The buttons also disable while the same connector action is queued or running, preventing accidental duplicate syncs.

## Live route audit — 2026-08-14 pre-fix baseline

This is a historical browser/network audit against `https://iroha.h.azusachino.icu` after the v0.4.1 cache deployment and before the 2026-08-15 request/read correction. Each canonical route was
hard-navigated, the request log was cleared, the route was reloaded, and the settled trace was captured after 6.5 seconds. API URLs were replayed with `xh`; every captured API URL returned
successfully. Cursor values and detail identifiers are intentionally omitted here.

The total includes JavaScript, stylesheets, images, and map tiles. It is not an API count. The stable overview trace was 57–60 total browser requests, not 83; 11 were API calls, 17–19 were
OpenStreetMap tiles for one map, and the remainder were static assets. A higher browser counter can include a different initial chunk/cache state or a longer tile-settling window.

| Route           | Total | API | OSM tiles | Settled API shape                                                                                                                                                    |
| --------------- | ----: | --: | --------: | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/`             |    46 |   4 |         0 | briefing, daily index, open tasks, plus the global metric-catalog fetch                                                                                              |
| `/overview`     |    57 |  11 |        17 | summary, five activity pages (initial + four cursor pages), one route collection, sleep list + year aggregate, media aggregate, plus the global metric-catalog fetch |
| `/motion`       |    47 |   6 |         0 | summary, the same activity page twice, two movement series, plus the global metric-catalog fetch                                                                     |
| `/patterns`     |    47 |   3 |         0 | daily month aggregate, latest-day page, plus the global metric-catalog fetch                                                                                         |
| `/night`        |    48 |   5 |         0 | month + year sleep aggregates, 31-session page, selected-session segments, plus the global metric-catalog fetch                                                      |
| `/library`      |    48 |   3 |         0 | media aggregate, media page, plus the global metric-catalog fetch                                                                                                    |
| `/expenses`     |    43 |  22 |         0 | one canonical expense page, 20 metric-series requests for the default four-currency/eleven-category view, plus the global metric-catalog fetch                       |
| `/reports`      |    47 |   3 |         0 | one 12-month monthly-series response, one selected monthly report, plus the global metric-catalog fetch                                                              |
| `/metrics`      |    46 |   3 |         0 | metric catalog twice (route + command palette), one metric series                                                                                                    |
| `/admin`        |    43 |   3 |         0 | metric catalog twice (route + command palette), one jobs page                                                                                                        |
| `/manual`       |    38 |   1 |         0 | global metric-catalog fetch only                                                                                                                                     |
| `/to-go`        |    45 |   4 |         0 | tasks, top-level sync jobs, media resolution tasks, plus the global metric-catalog fetch                                                                             |
| `/design`       |    39 |   2 |         0 | briefing plus the global metric-catalog fetch                                                                                                                        |
| `/motion/[id]`  |    94 |   5 |        43 | activity, route, heart-rate samples, laps, plus the global metric-catalog fetch; 43 tiles are one map                                                                |
| `/night/[id]`   |    47 |   3 |         0 | sleep detail, stage segments, plus the global metric-catalog fetch                                                                                                   |
| `/library/[id]` |    42 |   2 |         0 | one media detail response plus the global metric-catalog fetch                                                                                                       |

The legacy `/activities`, `/daily`, `/dashboard`, `/media`, and `/sleep` routes are redirects and were not counted as separate pages.

### Findings from the baseline

1. **Global metric catalog fetch is unnecessary on most routes.** `CommandPalette.svelte` loads `/api/v1/metrics` during layout mount even while the palette is closed. This adds one request to every
   route; `/metrics` and `/admin` each fetch the same catalog twice. Load it when the palette opens or use one memoized client-side catalog promise.

2. **Motion has a real duplicate request.** The reactive loader depends on `summary`, so it runs once before the summary resolves and once after it resolves with the same filters. The identical
   activity page is requested twice at the same timestamp. The loader should have one initial path and one filter-change path, with the summary used for display rather than as a request trigger.

3. **Overview is cacheable but structurally over-fetches.** The five activity pages are not duplicate calls: they are the initial page plus four keyset cursor pages needed to reach the current
   500-item client sweep. `listAllActivities({}, 500)` exists only to build a heatmap, current streak, and five-row recent list. Cache reduces the database cost after the first read, but it does not
   remove five network requests or the 500-row payload. The canonical backend contract should expose the active days/streak input and a small recent-activity window, or provide one overview
   projection; the browser should not walk the entire archive for those widgets.

4. **Expenses has a fan-out contract, not a cache failure.** The default page launches 20 distinct series requests: four currency totals, four currency counts, one daily series, and eleven category
   series. Each series request is server-aggregated, but `metricseries` calls `PeriodExpenses` independently, so those requests reread the same month’s canonical expense rows. A cacheable
   expense-dashboard projection or a batch series endpoint should return the currency totals/counts, category totals, and selected daily series in one response. The direct expense list should remain a
   live canonical read.

5. **Motion’s combined sport + year/month summary falls back to the loaded page.** When both filters are active, `displaySummary` sums the currently loaded activity page instead of requesting a
   complete filtered aggregate. With 24-row pages this can be numerically wrong once the filtered set exceeds one page. The summary contract needs to be called with the active filters (and a
   month/half-open period when selected), rather than deriving totals from a partial page in Svelte.

6. **The remaining route data boundaries are mostly sound.** Patterns consumes server daily aggregates; Reports consumes one server-built 12-month series plus the selected report; Night uses server
   aggregates for its headline values and fetches segments for the visibly selected session; detail pages request independent detail panels. Their client work is presentation mapping, sorting, and
   formatting rather than recomputing domain totals.

### Cursor-cache verdict

Each activity cursor page is a stable cache candidate as an individual representation. The cursor is part of the canonical cache key, the activity query has deterministic `(started_at DESC, id DESC)`
keyset ordering, and import/geocode changes invalidate the activities namespace. The live replay returned `X-Iroha-Cache: HIT` for the initial page and all four cursor pages.

This does not make the five-page walk a transactional snapshot. If canonical data is inserted between page requests, a normal keyset traversal can observe a moving archive and potentially skip or
repeat a boundary item. That is acceptable for the single-user dashboard and the cache invalidation model; a future export/synchronization job that requires a frozen view should use an explicit
snapshot/version token instead of treating the cursor as one.

### Corrective implementation status

1. Complete: the command-palette metric catalog is lazy and memoized while the palette is closed.
2. Complete: Motion has separate list/summary effects and filtered totals come from the server summary contract.
3. Complete: Overview uses one server-owned activity projection instead of a 500-row sweep; sleep uses a server-owned projection.
4. Complete: expense metric dimensions are batched per metric request, canonical expense source rows are loaded once, and direct GET reads are cacheable.
5. Complete: reports return one v2 series response with the current report embedded; date coverage, lifetime sleep aggregation, and half-open range semantics are server-owned.
6. Complete for the deployed date-scope candidate: live browser/network capture, exact-day briefing smoke, and the request-budget checks below passed after the cache representation version bump.

## Date-scoped correction pass — 2026-08-15

The reported missing `2026-08-13` day was not a lost canonical row. The daily endpoint and date index both contained the day; the root briefing returned empty daily and sleep sections because those
two contributors sent `From == To` to half-open list filters. The compatibility URL `/today` also had no route even though the canonical cockpit is `/`.

The correction is intentionally narrow:

- daily and sleep briefing contributors now query `[date, date + 1 day)`;
- the existing `/api/v1/daily/dates` path now unions the four cockpit domains (daily, activities, sleep, and dated media events), while retaining its date-only response shape;
- `/today` is a permanent application redirect to `/`, preserving `?date=YYYY-MM-DD`;
- the integration regression seeds one record per domain, verifies daily and sleep briefing rows on their selected days, and verifies all four dates in the shared index;
- the frontend route test verifies the redirect target and query preservation.

The source audit rechecked every canonical date-scoped route and found no other production `From == To` wiring. Motion, night, patterns, expenses, reports, and metrics use shared half-open month
bounds or server-owned aggregate/series requests; detail routes request only their visible panels. This pass also retains the existing request-budget checks for duplicate/fan-out regressions.

## Presentation correction pass — 2026-08-15

The wire contract still uses typed machine states (`ready`, `empty`, and `unavailable`) and list envelopes with `items`; those values are not visual copy. The shared report coverage component now
translates report availability into `Included` or `No records`, and the cockpit uses a themed `EmptyState` for a genuinely data-free day. No page is designed from `{}` or `[]` as a visual surface; the
design workshop continues to use its populated sample fixture when live data is incomplete.

The cockpit now uses stale-while-revalidate navigation. The first load presents a route-shaped skeleton; changing the day keeps the last committed snapshot mounted, labels the request as an update,
and commits the new snapshot only when its request version is current. Briefing and to-go requests both reject stale responses, so rapid left/right navigation cannot mix dates or collapse the lower
layout.

The browser's default calendar contract is the configured `PUBLIC_IROHA_TIMEZONE`, defaulting to `Asia/Tokyo` and intended to match the server's `IROHA_TIMEZONE`. A shared date helper now drives
today, month, year, heatmap, streak, and default timestamp presentation. The command palette is a mobile bottom sheet with a bounded internal list and safe-area padding. On small screens the global
fixed background and appbar blur are disabled to avoid repaint-heavy scrolling while retaining the theme palette.

## Regression checks

Use `agent-browser` for the live browser harness and inspect traffic from the same named session:

```sh
make web-visual-check BASE=https://iroha.h.azusachino.icu THEME=field-journal ROUTE=overview
agent-browser --session iroha-visual network requests --json
```

For a local frontend, run `make web-dev` first and point the same command at `http://127.0.0.1:5173`. The corrected implementation's expected initial API requests are:

| Route       | Initial API requests | Expected reads                                                            |
| ----------- | -------------------: | ------------------------------------------------------------------------- |
| `/`         |                    3 | briefing, daily date coverage, open tasks                                 |
| `/overview` |                    4 | activity overview, routes, sleep overview, media aggregates               |
| `/motion`   |                    4 | activity page, filtered summary, distance series, duration series         |
| `/patterns` |                    2 | selected-month daily aggregate, latest daily row                          |
| `/design`   |                    1 | briefing                                                                  |
| `/library`  |                    2 | media aggregate, first media page                                         |
| `/night`    |                    4 | sleep page, year aggregate, month aggregate, lifetime aggregate           |
| `/expenses` |                    5 | canonical expense pages, amount/count month series, daily/category series |
| `/reports`  |                    1 | monthly-report-series.v2                                                  |
| `/metrics`  |                    2 | metric catalog, selected metric series                                    |
| `/admin`    |                    2 | metric catalog, jobs                                                      |
| `/manual`   |                    0 | static manual content                                                     |
| `/to-go`    |                    3 | personal tasks, top-level sync jobs, open resolution tasks                |

## Read cache boundary

The v0.4.1 server cache-aside layer covers successful GET responses under /api/v1/briefing, /activities, /sleep, /daily, /media, /metrics, /reports, and /expenses. Direct expense records remain
canonical reads and are cached under `read_expenses`; derived expense metric series and reports are cached under their respective namespaces. Keys include the method, path, canonical encoded query
string, response version, aggregation version where applicable, and the server's effective timezone, so repeated exact reads cannot reuse a representation from another calendar interpretation. The
cache is best effort with a 24-hour safety TTL; successful canonical mutations advance their dependent namespaces after commit. Generation-safe writes prevent in-flight pre-mutation responses from
repopulating a post-mutation namespace, and a known invalidation failure makes that namespace bypass cache reads until recovery. Tasks, jobs, raw files, imports, sync actions, and other mutations are
intentionally live. X-Iroha-Cache: HIT|MISS|BYPASS is available for browser and deployment verification. See [the cache and aggregation plan](plans/2026-08-14-iroha-0.4.1-cache-and-aggregation.md).

API URL coverage remains in `src/lib/api.test.ts`; the normal `make check` gate covers type checking, formatting, backend tests, and frontend tests.
