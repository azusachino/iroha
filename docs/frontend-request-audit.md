# Frontend request audit

Date: 2026-08-04

This audit covers every top-level Svelte route plus sampled activity, media, and sleep detail routes. The baseline was captured with headless Chromium against `https://iroha.h.azusachino.icu` before
the control-room request fix. The count is the initial API traffic after navigation settles; cursor pages are counted individually.

## Findings

| Route           | Baseline | Decision                                                                                                                                                                          |
| --------------- | -------: | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/`             |        3 | Keep: briefing, day-index lookup for the scrubber, and the To-go strip are separate visible concerns.                                                                             |
| `/dashboard`    |        6 | Keep for now: summary plus five activity pages power the heatmap/streak/recent-history view. Route footprint remains opt-in and made no initial request.                          |
| `/activities`   |        2 | Keep: one visible activity page and one summary used by filters/statistics.                                                                                                       |
| `/daily`        |       13 | Fixed: initial load no longer sweeps all daily history or fetches yearly aggregates; Day loads one selected month (maximum 31 rows), Year loads one aggregate only when selected. |
| `/design`       |        1 | Keep: the design lab requests the live briefing when available and otherwise uses its sample content.                                                                             |
| `/media`        |        2 | Keep: one library page and one aggregate set are both visible on first paint. Filter changes reload only the library page.                                                        |
| `/sleep`        |        4 | Keep: session list, year/month trends, and the selected session's visible stage timeline. Both trend granularities are visible simultaneously.                                    |
| `/admin`        |        3 | Fixed: one task request replaces separate open/completed requests; jobs are scoped to top-level media-sync actions.                                                               |
| Activity detail |        4 | Keep: activity, route, heart-rate samples, and laps are independent detail panels.                                                                                                |
| Media detail    |        1 | Keep: one detail request.                                                                                                                                                         |
| Sleep detail    |        2 | Keep: session metadata and stage segments are both visible.                                                                                                                       |

## Control-room job propagation

The two sync buttons enqueue durable `media_sync_anilist` and `media_sync_bangumi` jobs. Each connector run then fans out into many `media_intake_parse` jobs. Those are worker/import jobs, not
personal tasks; the old UI displayed them in the same unfiltered list, making two clicks look like a runaway task list. The control room now requests and displays only the two top-level sync kinds and
explains that child importer work is tracked outside the personal queue.

The buttons also disable while the same connector action is queued or running, preventing accidental duplicate syncs.

## Regression checks

The opt-in browser audit is available with a running deployment:

```sh
cd apps/iroha-web
bun run request-audit -- --base https://iroha.h.azusachino.icu
```

For a local frontend against the deployed API, use `--api-base` while the Vite server is running:

```sh
bun run request-audit -- --base http://127.0.0.1:5174 --api-base https://iroha.h.azusachino.icu
```

The audit asserts that Admin makes one task request and one scoped job request, Daily does not sweep the entire history or fetch yearly aggregates before the Year tab is selected, Year makes one lazy yearly request, and Day requests at most one selected month with `limit=31`. The current post-fix initial traffic is:

| Route | Initial API requests |
| --- | ---: |
| `/` | 3 |
| `/dashboard` | 6 |
| `/activities` | 2 |
| `/daily` | 2 |
| `/design` | 1 |
| `/media` | 2 |
| `/sleep` | 4 |
| `/admin` | 2 |

API URL coverage remains in `src/lib/api.test.ts`; the normal `make check` gate covers type checking, formatting, backend tests, and frontend tests.
