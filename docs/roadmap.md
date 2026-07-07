# Roadmap

This roadmap tracks MVP v0. The product goal is to prove one vertical first: personal running data import, canonical storage, and private activity browsing.

Implementation should stay narrow:

- one Go app: `apps/iroha-server`
- no shared Go package until a second Go module exists
- no in-repo Telegram bot
- no private frontend until milestones 1-3 are complete
- Nix as the universal developer environment
- `uv` for repo scripts and local operational helpers
- Apple `container` for local Postgres/PostGIS on macOS

## Milestone 0: Data Rescue

Goal: make sure user-owned history exists outside vendor silos.

Tasks:

- Export Apple Health data from iPhone.
- Export Strava archive if available.
- Collect standalone GPX, FIT, and TCX files.
- Store raw files under a backed-up personal data directory.

This milestone can happen before iroha-server exists.

## Milestone 1: Iroha Server Import Core

Goal: upload and preserve raw source files.

Tasks:

- Add Nix flake for universal local tooling.
- Add `uv`-driven repo scripts for local checks and operational helpers.
- Add Apple `container`-based local Postgres/PostGIS runtime scripts for macOS.
- Create Go server skeleton under `apps/iroha-server`.
- Do not add `go.work` yet unless another Go module exists.
- Add GORM for application database access.
- Add UUIDv7 generation and prefixed API ID encoding.
- Add TOML config loading with environment variable overrides.
- Add SQL migration directory under `apps/iroha-server/db/migrations`.
- Add initial Postgres extensions migration for `postgis`.
- Add import-core migration for `tb_raw_files` and `tb_import_jobs`.
- Add `/api/v1/raw-files`.
- Store raw uploads on local filesystem.
- Compute SHA-256 and dedupe exact files.
- Add basic import status responses.
- Keep local development unauthenticated.

Exit criteria:

- A file can be uploaded through the API.
- The original bytes are stored unchanged.
- A raw file row exists with hash, size, source kind, and storage path.
- The local database can be migrated from empty to import-core schema.

## Milestone 2: Import Job Lifecycle

Goal: make imports observable and repeatable before parser depth grows.

Tasks:

- Add `/api/v1/imports`.
- Create import jobs from uploaded raw files.
- Add job statuses: `queued`, `parsing`, `completed`, `failed`.
- Run persisted jobs with an in-process worker first.
- Store parser version and error message.
- Model reprocessing as a new import job for the same raw file.

Exit criteria:

- A raw file can create an import job.
- Import status can be polled.
- Failed imports preserve the raw file and expose an error.
- A second import job can be created for the same raw file.

## Milestone 3: First Parser Path

Goal: parse Apple Health export zip and GPX into canonical activities.

Tasks:

- Parse Apple Health `export.xml` workout records.
- Parse route GPX files from `workout-routes/`.
- Parse standalone GPX.
- Add activity-core migration for `tb_activities`, `tb_external_refs`, `tb_activity_route_points`, `tb_activity_samples`, and `tb_activity_laps`.
- Insert activities and route points.
- Record external references where source identity is available.
- Make reprocessing idempotent.

Exit criteria:

- Uploading an Apple Health export creates activities.
- At least one activity has route points.
- Reprocessing the same raw file does not duplicate activities.

## Milestone 4: Activity Read API

Goal: expose enough data for the first private UI.

Tasks:

- Add `/api/v1/activities`.
- Add `/api/v1/activities/{activityId}`.
- Add route, sample, and lap endpoints.
- Add filters for sport type and date range.

Exit criteria:

- A client can list imported runs.
- A client can open one activity and render route and metrics.

## Milestone 5: Private Frontend

Goal: make the imported data feel useful.

Tasks:

- Build SvelteKit app shell.
- Add activity list.
- Add activity detail page.
- Render route with MapLibre.
- Render pace, heart-rate, and elevation charts where data exists.

Exit criteria:

- The first imported run is browsable in the UI.

## Milestone 6: External Upload Client Contract

Goal: let existing personal automation upload files without implementing a bot in this repo.

The personal Telegram bot is an external client. Iroha only owns the server-side contract it calls.

Tasks:

- Add bearer-token auth for upload clients.
- Document the upload flow for external clients.
- Support `uploaded_via=telegram` on `tb_raw_files`.
- Return stable JSON from raw-file creation and import creation.
- Make import status responses compact enough for bot messages.
- Keep parser and dedupe logic only in `iroha-server`.

Exit criteria:

- An external Telegram bot can upload a GPX/FIT/TCX file to iroha-server.
- The bot can create an import job and poll its status.
- No Telegram-specific implementation exists inside this repo.

Full Apple Health export zips may exceed normal Telegram Bot API file limits, so web upload remains the reliable large-file path.

## Milestone 7: Privacy and Publishing

Goal: separate private canonical data from public output.

Tasks:

- Add privacy zones.
- Add sanitized activity projection generation.
- Add `tb_published_activities`.
- Add public read path backed by sanitized payloads.

Exit criteria:

- A public activity can be generated without exposing exact private route data.

## Future Module: Reading and Watching Stats

Goal: track personal media consumption without turning iroha into a social media clone.

This module should reuse the same three-layer pattern as running:

```text
raw imports
  -> canonical normalized records
  -> private dashboards and optional public summaries
```

In scope for a first reading/watching module:

- Manual entry for books, manga, articles, films, shows, and videos.
- CSV/JSON import from existing trackers where available.
- Canonical `media_items` records for title, type, creators, release year, and external refs.
- `media_events` records for started, progressed, completed, abandoned, reread, and rewatched events.
- Progress units that fit the media type: pages, chapters, episodes, minutes, percent.
- Ratings, notes, tags, and favorite quotes/snippets.
- Year/month summaries: count, time spent, pages read, episodes watched, favorite creators, top tags.

Out of scope initially:

- Social follow/follower graphs.
- Recommendation engine.
- Full metadata scraping pipeline.
- DRM or paid-content extraction.
- Public publishing beyond sanitized yearly summaries.

Candidate future tables:

```text
tb_media_items
tb_media_external_refs
tb_media_events
tb_media_notes
tb_media_collections
```

This should wait until running import, activity read API, and the first private UI prove the core product loop.

## Deferred

- Native iOS HealthKit sync app.
- Background sync.
- Strava API integration.
- FIT and TCX full fidelity.
- In-repo Telegram bot.
- `go.work`.
- Shared Go packages.
- Separate worker binary.
- Dedicated CLI.
- Route clustering and similarity.
- Heatmaps.
- Annual yearbook.
- Reading and watching stats.
- Photos, location history, notes, and other personal data modules.
