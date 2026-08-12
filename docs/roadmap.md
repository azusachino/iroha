# Roadmap

> This is the original MVP roadmap baseline. Several constraints below are historical milestones rather than current architecture; current runtime and release behavior is documented in the README,
> `docs/dev-runtime.md`, and `docs/contracts/`.

This roadmap tracks MVP v0. The product goal is to prove one vertical first: personal running data import, canonical storage, and private activity browsing. The broader cockpit direction is described
in [Personal Data Cockpit Model](personal-data-cockpit.md).

Implementation should stay narrow:

- one private API app first: `apps/iroha-server`
- one worker app once jobs need to outlive API requests: `apps/iroha-job`
- no shared Go package until a second Go module exists
- no in-repo Telegram bot
- no private frontend until milestones 1-3 are complete
- mise as the universal developer environment
- `uv` for repo scripts and local operational helpers
- Podman + podman-compose for the local Postgres/PostGIS stack on macOS

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

- Add `.mise.toml` for universal local tooling.
- Add `uv`-driven repo scripts for local checks and operational helpers.
- Add Podman Compose-based local Postgres/PostGIS runtime scripts for macOS.
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
- Keep the job contract compatible with a later `iroha-job` worker process.

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
- Add activity-core migration for `tb_activities`, `tb_external_refs`, `tb_activity_route_points`, `tb_activity_samplings`, and `tb_activity_laps`.
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
- Add route, sampling, and lap endpoints.
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

Goal: separate private canonical data from public output. See [Public-site publishing workflow](public-site-publishing.md) for the operational pipeline, review loop, and rollback path (issue #41).

Status: partially shipped. Privacy zones and sanitized activity projections are implemented — `apps/iroha-server/pkg/activities` trims route endpoints and masks auto-detected private zones (home,
work, and other frequent start/end hubs) across every route, and `apps/iroha-server/pkg/publicexport` builds the sanitized activity/summary/route projection from that. The original design served this
from a `/public/v1` HTTP surface living in the same process as the private API; that surface has been removed (it was never actually exposed to the internet — an open-CORS route and a second
rate-limit budget serving a page nobody could reach). The replacement is a static export instead of a second live API surface:

```text
apps/iroha-server/cmd/iroha-export-public (sanitized JSON/GeoJSON snapshot)
  -> committed by a k3s CronJob (ops/images/Containerfile.server's export-public target;
     the CronJob resource itself is defined by the deployment environment, not this repo), from inside the
     private network
  -> apps/iroha-public-site (static SvelteKit app) built and deployed to GitHub Pages
     by an ordinary GitHub-hosted Actions workflow, triggered by that commit
```

No self-hosted GitHub Actions runner is used anywhere in this design. iroha's repo is public, and GitHub's own guidance is to avoid attaching self-hosted runners to public repos — a workflow change
merged through a PR can execute on that runner, turning it into a path from "a PR gets merged" to code execution inside the private network. The export step instead runs entirely outside GitHub
Actions, on infrastructure GitHub never touches (`ops/scripts/export-public-cron.sh` clones a disposable copy of the repo, runs the export, and pushes only if the data actually changed).

Exit criteria:

- Public activities use the sanitized projection by default; an explicitly approved activity may publish its full grapher detail, including route, samples, and laps.

Status: the GitHub Pages deployment and k3s CronJob are live. The recurring publication loop uses the dedicated sealed `IROHA_EXPORT_GITHUB_PAT` credential and publishes only the validated snapshot
under `apps/iroha-public-site/static/data/`; the approved-detail allowlist is maintained in the exporter code.

## Milestone 8: Durable Worker Backbone

Goal: move long-running and repeatable work out of request handlers without adding external queue infrastructure.

Tasks:

- Add a Postgres-backed `tb_jobs` queue.
- Add `apps/iroha-job` as the worker app module.
- Move import execution from in-process API handling into job handlers.
- Support retry, `run_after`, locking, attempts, and compact error reporting.
- Add explicit job kinds for parser reprocess and projection refresh.
- Add a small schedule model for regular connector syncs.
- Treat user-triggered dumps, such as requesting a fresh health-data export, as durable trigger jobs even when the source device/client performs the dump.

Exit criteria:

- The API can enqueue work and return immediately.
- The worker can claim and complete queued imports.
- A regular sync job can be scheduled and inspected.
- A manual "sync now" or "request full dump" action creates an observable job.

Status: shipped for imports, geocoding, and the first AniList/Bangumi connector syncs. The private API queues media sync jobs at `POST /api/v1/media/sync/{connectorId}`; remaining media work is
ontology expansion and cross-provider resolution.

## Release 0.4: Expense Ledger

Goal: capture lightweight personal expenses from Telegram or another external client, then report them beside the existing activity, night, and media aggregates without making iroha own a bot.

The intake boundary follows Milestone 6: Telegram remains an external client, while iroha owns authentication, idempotency, canonical records, corrections, and read/report APIs.

First vertical slice:

- Define a canonical `tb_expenses` record with amount, currency, occurred-at date, category, merchant/description, source, and stable external idempotency key.
- Add an authenticated create/list/update API for external clients, including a compact response suitable for Telegram confirmation messages.
- Keep corrections and deletion auditable; never silently overwrite an imported expense.
- Add weekly and monthly aggregates by total, category, and currency, with explicit timezone boundaries.
- Add an Overview expense tile and weekly/monthly report views without changing existing activity, night, or media totals.
- Exclude expenses from public export by default; add a separate opt-in sanitized summary only after the private ledger is trustworthy.

Exit criteria:

- A Telegram-side client can submit one expense, receive a stable confirmation, and safely retry without duplication.
- A user can correct an expense and see the correction reflected in weekly and monthly reports.
- Aggregate totals are covered by API and database tests across timezone, currency, retry, and empty-period cases.
- The private UI and API make the expense boundary clear, while the public projection remains expense-free by default.

## Future Module: Reading and Watching Stats

Goal: track personal media consumption without turning iroha into a social media clone.

See [Reading and Watching History Research](media-history-research.md) for the researched data model, source connectors, and implementation boundary. See also
[Personal Data Cockpit Model](personal-data-cockpit.md) for the shared intake, job, sync, and trigger model.

This module should reuse the same three-layer pattern as running:

```text
raw imports
  -> canonical normalized records
  -> private dashboards and optional public summaries
```

In scope for a first reading/watching module:

- Manual entry for books, manga, articles, films, shows, and videos.
- CSV/JSON import from existing trackers where available.
- Manual and scheduled connector syncs where APIs are available.
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
- Strava API integration.
- FIT and TCX full fidelity.
- In-repo Telegram bot.
- `go.work`.
- Shared Go packages.
- Dedicated CLI.
- Route clustering and similarity.
- Heatmaps.
- Annual yearbook.
- Reading and watching stats.
- Photos, location history, notes, and other personal data modules.
