# Development Runtime

## Current Assumption

Iroha development happens on macOS. mise manages the project tools, while Podman with `podman-compose` owns the containerized services. Make remains the only task and lifecycle entrypoint;
`scripts/dev_stack.py` implements its backend behavior.

The runtime design should be capability-based. The checked-in Compose files target the OCI-compatible Podman runtime; Docker compatibility is incidental and not the supported local contract.
`make dev-up` starts the database first, waits for `pg_isready`, applies migrations, and then starts server, worker, and web so application containers do not race database initialization.

## Tool management with mise

Use the checked-in `.mise.toml` to install the project tools:

```bash
mise install
```

Project tools follow LTS where available, otherwise latest stable: Node uses `lts`; Go, uv, Bun, SQLx CLI, rumdl and golangci-lint use `latest` in `.mise.toml`. uv selects stable Python3 through
`.python-version`, subject to `pyproject.toml`'s minimum. Application dependencies remain lockfile-controlled. Podman and its machine remain host prerequisites. Database readiness uses `pg_isready`
inside PostGIS; no separate host PostgreSQL installation is required.

Make automatically uses `mise exec --`, so the normal workflow stays unchanged:

```bash
make db-up
make run
make run-job
make dev-up
make db-down
```

CI provisions the same `.mise.toml` selection policy with `jdx/mise-action`. Floating selectors can resolve differently across installation dates; record actual versions when comparing results.

Podman and `podman-compose` remain macOS host prerequisites rather than project-managed tools. The uv-managed runner detects missing tools and reports a clear prerequisite error.

## Incremental local checks

Use `make local-check` while iterating. It caches successful check groups under
ignored `.cache/local-checks/`, printing `RUN` or `HIT` for each group.
`make local-check ARGS='--force'` runs every group fresh; `ARGS='--groups web'`
selects a group explicitly. Neither command is PR/release acceptance evidence:
`make check`, `make validate`, browser, integration and CI gates remain unchanged
and never consume this cache. The runner refuses CI use.

| Group | Inputs and checks |
| --- | --- |
| Go workspace | All Go apps and `go.work*`; formatting, vet, lint, tests with aggregate coverage, HTTP contract tests. All six dependent modules invalidate together. Native Go build/test caches remain enabled; no per-package cache is added. |
| Scripts | Python coverage/tests. Changes under `scripts/` conservatively invalidate every group because runners/checkers define their behavior. |
| Contracts | Both frontend apps and shared packages; theme placement, responsive and motion contracts. |
| Web | Private frontend and shared packages; formatting, type checks and Vitest. Public-site edits also invalidate this group because private type checks include both-host browser fixtures. |
| Public | Public frontend and shared packages; formatting and type checks. |

Unknown paths, root configuration, locks outside module trees and docs invalidate
all groups. Keys use sorted file names and content hashes, including nonignored
untracked files and deletions, and checkout-contained symlink targets (including
shared fonts). Ignored `.env*` files at the root and app/package roots participate
as well. Symlinks outside the checkout or cycles fail closed. Tool versions,
resolved tool paths, Python/platform, commands and environment also participate.
Terminal/agent-session metadata is excluded from the environment key; outer Make
control flags and local-check selectors are stripped from child commands and keys.
Environment values and dotenv bytes are hashed, not stored. A failed/interrupted run removes prior success;
inputs changing during a run prevent publication. Corrupt records are misses.

Frozen frontend dependency installation and the diff-sensitive quality-floor guard
always run, even on hits. This is a local source-input cache, not dependency
integrity verification: after manually modifying ignored dependencies/build state,
use `--force`. It does not cache browser, integration, builds or live checks and
does not restore coverage/build outputs from earlier runs.

The normal target enters mise once, verifies that Go/uv/Node/Bun/golangci-lint
are the executables selected by mise for this checkout, then invokes child Make
targets directly with `TOOL_ENV=`. Keys include actual resolved paths and versions,
not the `latest`/`lts` strings. If your shell already has those selected tools,
`make local-check TOOL_ENV=` avoids even that bootstrap; mismatched/missing tools
fail closed. This override is for `local-check`, not a global tool-policy change.

Lead macOS measurements during #112 implementation: cold local checks 43.38s,
unchanged repeat 1.47s (five hits), and an added private-web source comment 25.13s
(only Web/Contracts ran; Go/Scripts/Public hit). Independent initial review measured
fresh local checks 40.27s, repeat hits about 1.22s and full `make validate` 55.4s;
the lead's source-mutation measurement was not independently reproduced. These
are local iteration times, not CI or production performance claims. Tests retain
the other input-boundary, failure, force-fresh and tool-mismatch regressions.

## CI execution

The `make check` GitHub status aggregates mandatory validation/build/integration
and two parallel browser-shard jobs. It succeeds only when both job results are
`success`; skipped, cancelled or failed dependencies block it. The PostGIS and
public-serving checks remain in the validation job. Each browser runner installs
frozen frontend dependencies and Chromium, then runs
`make e2e ARGS='--shard=1/2'` or `--shard=2/2`. All Chromium tests still run;
one worker, retries and fail-on-flaky behavior are unchanged. Sharding is by file,
so balancing can vary as the suite grows.

CI restores native Go build/module caches keyed by OS/architecture, resolved Go version, tool configuration,
workspace/module manifests and revision, with a dependency-scoped fallback.
Go validates its own content keys; validation commands always execute regardless
of a cache hit. CI does not consume the incremental local result cache.
Parallel jobs target wall time, not a reduction in total billed runner minutes.

Baseline for #114: PR #113's successful CI run `37101989926` took 12m54s;
validation/build 3m29s, integration 24s and browser installation/tests 8m21s
(200 tests using one worker, 7.9m test execution). Post-change savings must be
measured from exact-head CI, not inferred from local runs.

References: [Playwright sharding](https://playwright.dev/docs/test-sharding) and
[GitHub dependency caching](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows).

## `uv`

Use `uv` for repo scripts, smoke checks, fixtures, import experiments, and one-off operational helpers.

Current repo files:

```text
pyproject.toml
uv.lock
.python-version
main.py
```

Expected usage:

```bash
uv run python main.py
uv run python scripts/<name>.py
```

Future scripts should prefer Python under `scripts/` or `pyscripts/` over shell when the logic is more than a thin command wrapper.

`uv` manages Python script dependencies under the mise-provided Python/uv layer.

## Go Workspace

Use a Go workspace if the repo has more than one Go module.

Preferred monorepo shape:

```text
go.work
apps/
  iroha-server/
    go.mod
    cmd/iroha-server/
    pkg/
  iroha-job/
    go.mod
    main.go
  iroha-web/
    package.json
```

The repo currently uses independent Go modules for `iroha-core`, `iroha-providers`, `iroha-runtime`, `iroha-imports`, `iroha-server`, and `iroha-job`, resolving local dependencies with `replace`
directives. The dependency direction is one-way: provider contracts sit in core, runtime infrastructure is shared by imports and both executables, and the server/job modules consume the import
pipeline.

The private frontend lives alongside the server at `apps/iroha-web` (SvelteKit on Node LTS). It, `apps/iroha-public-site` and `packages/iroha-shared` share one native Bun workspace and root `bun.lock`.
`make frontend-install` performs one frozen workspace install; the older install target names are aliases. Both hosts declare `@iroha/shared` as `workspace:*`. The root Svelte dependency supplies
one runtime for repository-scoped tests and shared peers. Bun installs and launches existing tooling; Vite/Vitest/Playwright remain. Vitest V8 coverage uses Node, not forced Bun execution. Source aliases remain for SvelteKit's shared TS/Svelte/CSS imports, not to repair runtime resolution. See [ADR0009](adr/0009-native-frontend-workspace.md).

`make web-test` measures all private app/shared TS and Svelte sources, including never-imported files, and then checks the report against a filesystem inventory. The historical app-scoped report
omitted all101 shared files; `allowExternal` alone still omitted17. Neither numeric floors nor old baseline records are lowered by repairing that denominator.

### Frontend Browser Checks

`make e2e` runs the Playwright specs in `apps/iroha-web/e2e/` in headless Chromium. The config starts its own Vite server on `127.0.0.1:5183`, so a dev server on 5173 can keep running, and each spec fakes the `/api` calls it needs with `page.route` (`e2e/session.ts` fakes the session every route asks for), so no Go backend or database is involved. svelte-check types the specs against `src/lib/api.ts`.

```bash
make e2e-install              # once: the Chromium build this Playwright version needs
make e2e                      # the checks; ARGS='-g "sign in"' filters by name
make e2e-probe ROUTE=/path    # before writing a spec: ARIA snapshot, errors, failed requests, screenshot
```

A failing check leaves its screenshot, error context and trace under `apps/iroha-web/test-results/`; open a trace from the web directory with `mise exec -- bun run playwright show-trace <trace.zip>`. `make web-visual-check` stays the tool for themed screenshots.

### Frontend Browser Smoke

Use `agent-browser` for browser screenshots and harness checks. It owns its Chromium session outside the web dependency tree; no frontend browser package is required.

Install its browser once if needed:

```bash
make web-visual-install
```

Then run the web dev server and check a route:

```bash
make web-dev
make web-visual-check THEME=field-journal ROUTE=overview
```

The equivalent direct commands are:

```bash
agent-browser --session iroha-visual open "http://127.0.0.1:5173/overview"
agent-browser --session iroha-visual wait 800
agent-browser --session iroha-visual snapshot -c
agent-browser --session iroha-visual screenshot --full .visual-check/overview.png
```

Pitfalls to avoid:

- Kill stale Vite listeners on port 5173 before starting a new frontend smoke run.
- Use a named `--session` so navigation, storage, screenshots, and error checks share one browser.
- Clear or close the named session after a one-off check so browser processes do not accumulate.
- Keep this as a browser smoke check. Unit/e2e behavior should stay in `make web-test`, `make scripts-test`, and `make check` so CI does not depend on a headed browser session.

### Public-site preview

The public site is a static client shell that reads the validated sanitized projection from the local `iroha-server` through `/public/v1`. Both preview targets proxy that namespace to `IROHA_DEV_API_TARGET`, which defaults to `http://127.0.0.1:8080`; they do not call private `/api/v1` routes. Start the local server with the dataset you intend to review, then run:

```bash
# Fast iteration with Vite's development server.
make public-site-dev

# Production-like build and preview.
make public-site-preview
```

See [public-site publishing](public-site-publishing.md) for the current deployed pipeline and privacy boundary.

For manual inspection outside the site, `make export-public` writes a snapshot to disk. The default includes route traces; an export run made with `--privacy` omits traces while retaining metrics, samples, and laps. This CLI snapshot does not feed the local site preview.

Both commands serve the site at `http://127.0.0.1:4173/` (the development server may choose a different port if 5173 is occupied). Leave `BASE_PATH` unset locally; GitHub Pages supplies `/iroha` in
`public-site.yml`, and that workflow smoke-checks the deployed project-page path. To test a new export locally, replace the ignored working snapshot files temporarily, run the preview, then restore
them without committing personal data. Close the preview process after a one-off check so stale listeners do not accumulate.

Do not use a Git submodule for `iroha-server` unless it must live in a separate repository with independent release ownership. In this product phase, `iroha-server` should be a subdirectory module
inside the iroha repo, not an external Git submodule.

## Podman

Podman is the supported local container runtime. On macOS it runs the containers inside a Podman machine; the machine disk is separate from the repository and should be sized deliberately rather than
using an oversized default.

Verified local tools on the development host:

```text
podman 5.8.2
podman-compose 1.5.0
```

Initialize a lean machine once when needed, then start it before using the stack. Do not run `podman system prune` or remove volumes as part of normal development; raw imports remain in `.iroha-data`,
while the database volume is explicitly managed by the stack.

For MVP v0, the important local service is Postgres with PostGIS.

Target shape:

```text
ops/local-dev/compose.yaml
  -> Postgres/PostGIS dependency (cache is Postgres-backed)
  -> server, worker, and web services
  -> fixed private service names plus host-published developer ports
  -> named database volume and a shared `.iroha-data` bind mount

scripts/dev_stack.py
  start     -> start/build the complete Podman Compose stack, wait for DB, apply migrations
  deps      -> start only Postgres, wait for DB, apply migrations
  stop      -> stop and remove stack containers/network
  status    -> show stack status
  logs      -> show database logs
  reset     -> recreate local dev database volume, wait for DB, apply migrations
```

The script should hide local command differences while keeping behavior explicit.

## Real Import Smoke

Keep real exports under ignored local storage:

```bash
mkdir -p .iroha-data/imports
cp ~/Downloads/export.zip .iroha-data/imports/apple-health-export.zip
```

With `iroha-server` running locally, smoke the product HTTP route.

> [!WARNING] File imports are processed asynchronously via a database-backed queue. Therefore, running `make smoke-real-import` **requires** the `iroha-job` worker to be running alongside the server.
>
> - Start the server: `make run`
> - Start the worker in another terminal: `make run-job`
>
> **Workspace/Data Dir Warning**: If running from different subdirectories, make sure both processes point to the same directory by sharing `IROHA_DATA_DIR` (e.g.
> `export IROHA_DATA_DIR=$PWD/.iroha-data`).
>
> Run the smoke check:
>
> ```bash
> make smoke-real-import FILE=.iroha-data/imports/apple-health-export.zip
> ```

This script uses the same upload/import APIs an external client uses:

```text
POST /api/v1/raw-files
POST /api/v1/imports
GET  /api/v1/imports/{importId}
GET  /api/v1/activities
GET  /api/v1/activities/{activityId}/route
```

No manual `uv` environment setup is needed. Run repo scripts through mise or Make; `uv` uses the existing `pyproject.toml` and `uv.lock`.

## Postgres/PostGIS Runtime Shape

Use OCI images that support arm64. The validated local image is `docker.io/kartoza/postgis:18.4-3.6.4--v2026.06.21`. The Kartoza image is a temporary local development choice; production deployments
should pin and validate their own PostGIS image separately.

Runtime requirements:

```text
POSTGRES_DBNAME=iroha
POSTGRES_USER=postgres
POSTGRES_PASS=iroha_dev
published port: 5432
persistent local data volume
```

Application config should use a normal database URL:

```text
DATABASE_URL=postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable
```

Kartoza uses `POSTGRES_PASS` and `POSTGRES_DBNAME` for initialization. The local bootstrap keeps `postgres` as the image’s bootstrap superuser and creates the application-owned `iroha` role through
`ops/local-dev/initdb/001-iroha-user.sql`.

## Database Migrations

Use GORM for application data access and SQL migrations for schema changes. Migration files are owned by `apps/iroha-server`.

Preferred layout:

```text
apps/iroha-server/db/migrations/
```

Preferred CLI:

```text
SQLx CLI (latest stable)
```

The CLI comes from mise for both local development and CI. A `uv` script wraps common operations:

```bash
uv run python scripts/db.py apply
uv run python scripts/db.py rollback
uv run python scripts/db.py status
```

The wrapper should read `DATABASE_URL` and pass it to the migration CLI. It should not contain schema logic.

`apply` maps to `sqlx migrate run`: apply pending schema changes.

`rollback` maps to `sqlx migrate revert`: revert the most recent schema change when that is safe during development.

Migrations use paired `<version>_<description>.up.sql` and `.down.sql` files. SQLx records applied versions in `_sqlx_migrations`.

## Configuration

Use TOML config with environment variable overrides.

Development default:

```text
./iroha.toml
```

Environment variables should override matching TOML fields:

```text
IROHA_SERVER_ADDR
IROHA_TIMEZONE
IROHA_DATABASE_URL
IROHA_DATA_DIR
IROHA_CACHE_BACKEND
IROHA_ALLOWED_ORIGINS
```

The server and job receive the database, shared cache, and data-directory settings. The shared runtime cache module supports `postgres`, `valkey`, and `none`; Postgres is the default and canonical
data remains authoritative in Postgres in every mode. Set `IROHA_CACHE_BACKEND=valkey` only for compatibility deployments that provide a Valkey URL. There is no production process-memory cache. Only
the server receives private CORS origins (`IROHA_ALLOWED_ORIGINS`). The private API requires an owner session; see `docs/iroha-server.md#auth`.

The cache cutover is reversible because cache entries are disposable. A compatibility rollback uses `IROHA_CACHE_BACKEND=valkey` and `IROHA_VALKEY_URL=redis://...` with the Valkey service restored;
switching backends causes misses and regeneration, not data migration. The worker performs bounded Postgres cache cleanup on startup and hourly while running; this is housekeeping, not
daily/monthly/yearly aggregation.

## Commands

```bash
make db-up       # Postgres/PostGIS, migrations, host-process development
make dev-up      # complete Podman Compose stack: db, server, job, web
make dev-watch   # rebuild changed server/job/web services while developing
make db-status
make db-logs
make db-down
make db-reset    # destructive: removes the database volume, then migrates
```

The Makefile is intentionally thin. Lifecycle, readiness, migration, and command construction live in `scripts/dev_stack.py`; business/import exploration and smoke assertions live in uv scripts.

`make dev-watch` polls source mtimes and rebuilds only affected Compose services. Go changes rebuild `server` and `job`; web changes rebuild `web`; Compose changes rebuild the affected profile. It is
deliberately dependency-free and runs inside the same Podman machine.

The real import smoke runs against host-published ports while the worker remains a container:

```bash
make smoke-local FILE=.iroha-data/imports/apple-health-export.zip
```

For a non-mutating local-stack soak, use the same boundary:

```bash
make soak-local SOAK_ARGS="--duration-s 300 --interval-s 2"
```
