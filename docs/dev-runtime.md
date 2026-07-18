# Development Runtime

## Current Assumption

Iroha development happens on macOS. Nix is the universal manager for tools and developer entrypoints. The repo uses `uv` for Python-based scripts inside that Nix-managed environment and Podman with
`podman-compose` for local containerized services.

The runtime design should be capability-based. The checked-in Compose files target the OCI-compatible Podman runtime; Docker compatibility is incidental and not the supported local contract.

## Nix

Use a flake as the outer contract for local development.

Expected responsibilities:

```text
nix develop
  -> provides Go toolchain
  -> provides uv
  -> provides Node/frontend tooling when needed
  -> provides database and migration CLIs
  -> exposes repo checks
```

Nix should manage tool availability and versions. Project scripts should assume they are running inside `nix develop`, but keep commands plain enough that CI can reuse them.

Podman and `podman-compose` may remain macOS system tools if it is not practical to package them through Nix. The uv-managed runner detects missing tools and reports a clear prerequisite error.

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

`uv` is not the universal tool manager here. It manages Python script dependencies under the Nix-provided Python/uv layer.

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

The private frontend lives alongside the server at `apps/iroha-web` (SvelteKit, built with `bun`); see `apps/iroha-web/README.md`.

### Frontend Browser Smoke

Use the on-demand Playwright CLI for browser screenshots. Chrome is not assumed to exist on the local machine; if `playwright-cli open ... --headed` fails with
`Chromium distribution 'chrome' is not found`, install WebKit once:

```bash
cd apps/iroha-web
bunx -p @playwright/cli@latest playwright-cli install-browser webkit
```

Then run the web dev server:

```bash
make web-dev
```

Open a route and capture the current page:

```bash
cd apps/iroha-web
bunx -p @playwright/cli@latest playwright-cli open "http://127.0.0.1:5173/dashboard" --browser webkit --headed
bunx -p @playwright/cli@latest playwright-cli screenshot --full-page --filename=artifact.png
```

The screenshot command acts on the currently opened page. Use `--browser webkit` unless Chrome is intentionally installed.

Pitfalls to avoid:

- Kill stale Vite listeners on port 5173 before starting a new frontend smoke run.
- Do not assume Chrome exists on this macOS host. The Playwright CLI defaults can fail until an installed browser is selected explicitly.
- `playwright-cli screenshot` is not a navigation command. Passing both a URL and output path fails with `too many arguments`; open the page first, then capture the current page.
- Keep this as a browser smoke check. Unit/e2e behavior should stay in `make web-test`, `make scripts-test`, and `make check` so CI does not depend on a headed browser session.

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

No manual `uv` environment setup is needed. Run repo scripts through `make` or `nix develop ... uv run`; `uv` uses the existing `pyproject.toml` and `uv.lock`.

## Postgres/PostGIS Runtime Shape

Use OCI images that support arm64. The validated local image is `docker.io/kartoza/postgis:18.4-3.6.4--v2026.06.21`. The Kartoza image is a temporary local
development choice; production deployments should pin and validate their own PostGIS image separately.

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

Kartoza uses `POSTGRES_PASS` and `POSTGRES_DBNAME` for initialization. The
local bootstrap keeps `postgres` as the image’s bootstrap superuser and creates
the application-owned `iroha` role through
`ops/local-dev/initdb/001-iroha-user.sql`.

## Database Migrations

Use GORM for application data access and SQL migrations for schema changes. Migration files are owned by `apps/iroha-server`.

Preferred layout:

```text
apps/iroha-server/db/migrations/
```

Preferred CLI:

```text
Goose
```

The CLI should come from Nix. A `uv` script can wrap common operations:

```bash
uv run python scripts/db.py apply
uv run python scripts/db.py rollback
uv run python scripts/db.py status
```

The wrapper should read `DATABASE_URL` and pass it to the migration CLI. It should not contain schema logic.

`apply` maps to migration-tool `up`: apply pending schema changes.

`rollback` maps to migration-tool `down`: revert the most recent schema change when that is safe during development.

## Configuration

Use TOML config with environment variable overrides.

Development default:

```text
./iroha.toml
```

Environment variables should override matching TOML fields:

```text
IROHA_SERVER_ADDR
IROHA_DATABASE_URL
IROHA_DATA_DIR
IROHA_CACHE_BACKEND
IROHA_LOCAL_NO_AUTH
IROHA_ALLOWED_ORIGINS
IROHA_JWT_SECRET
IROHA_JWT_ISSUER
IROHA_JWT_AUDIENCE
```

The server and job receive the database, Postgres-backed cache, and shared data-directory settings. Set `IROHA_CACHE_BACKEND=valkey` only for compatibility deployments that still provide a Valkey URL. Only the server receives the JWT verification secret and private CORS origins. For an authenticated static web
deployment, pass a read-only `PUBLIC_IROHA_API_TOKEN` as the web image build argument; it is intentionally public within the private site and must not be a signing secret or write token.

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
