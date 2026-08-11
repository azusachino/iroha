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

## Regression checks

Use `agent-browser` for the live browser harness and inspect traffic from the same named session:

```sh
make web-visual-check BASE=https://iroha.h.azusachino.icu THEME=field-journal ROUTE=overview
agent-browser --session iroha-visual network requests --json
```

For a local frontend, run `make web-dev` first and point the same command at `http://127.0.0.1:5173`. The current post-fix request expectations remain:

| Route       | Initial API requests |
| ----------- | -------------------: |
| `/`         |                    3 |
| `/overview` |                    6 |
| `/motion`   |                    2 |
| `/patterns` |                    2 |
| `/design`   |                    1 |
| `/library`  |                    2 |
| `/night`    |                    4 |
| `/to-go`    |                    2 |

## Read cache boundary

The server cache-aside layer covers successful `GET` responses under `/api/v1/briefing`, `/activities`, `/sleep`, `/daily`, and `/media`. Keys include the method, path, and canonical encoded query
string, so repeated exact reads can be served without another database aggregation. The cache is best effort with a 24-hour safety TTL; successful import completion advances every read namespace, so
imported data is refreshed immediately. Tasks, jobs, raw files, imports, sync actions, and other mutations are intentionally live. `X-Iroha-Cache: HIT|MISS` is available for browser and deployment
verification.

API URL coverage remains in `src/lib/api.test.ts`; the normal `make check` gate covers type checking, formatting, backend tests, and frontend tests.
