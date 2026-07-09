# Development Runtime

## Current Assumption

Iroha development happens on macOS. Nix is the universal manager for tools and developer entrypoints. The repo uses `uv` for Python-based scripts inside that Nix-managed environment and can use Apple `container` for local containerized services.

The runtime design should be capability-based. Do not assume Docker Compose compatibility unless it has been tested locally.

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

Apple `container` may remain a macOS system install if it is not practical to package through Nix. In that case, Nix-managed scripts should detect it and report a clear missing-runtime error.

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

We use a Go workspace to manage multiple modules (`apps/iroha-server` and `apps/iroha-job`), resolving local dependencies cleanly with `replace` directives.

The private frontend lives alongside the server at `apps/iroha-web` (SvelteKit, built with `bun`); see `apps/iroha-web/README.md`.

Do not use a Git submodule for `iroha-server` unless it must live in a separate repository with independent release ownership. In this product phase, `iroha-server` should be a subdirectory module inside the iroha repo, not an external Git submodule.

## Apple `container`

Apple `container` is the preferred macOS container runtime candidate for local services.

Verified local fact:

```text
container CLI version 1.1.0
```

Upstream facts to preserve:

- `container` runs Linux containers as lightweight VMs on Apple silicon Macs.
- It consumes and produces OCI-compatible images.
- It can build images from Dockerfiles or Containerfiles.
- The service is started with `container system start`.
- Basic workflows include `container build`, `container run`, `container list`, `container logs`, `container exec`, and `container stop`.
- Newer upstream docs describe macOS 26 as the supported target and note limitations on macOS 15 networking.

## Practical Rule

Use Apple `container` for explicit local services, not as a transparent Docker Compose replacement. The repo uses `bianpai --backend container` as the local orchestration layer for supported host-published services.

For MVP v0, the important local service is Postgres with PostGIS.

Target shape:

```text
ops/local-dev/compose.yaml
  -> single Postgres/PostGIS service
  -> publishes 5432 to localhost
  -> uses a named persistent volume

scripts/dev_stack.py
  start     -> start Apple container + bianpai stack, wait for DB, apply migrations
  stop      -> stop and remove stack containers/networks
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

With `iroha-server` running locally, smoke the product HTTP route:

```bash
make smoke-real-import FILE=.iroha-data/imports/apple-health-export.zip
```

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

Use an OCI image that supports arm64. The validated local image is `ghcr.io/baosystems/postgis:17-3.5`. The upstream `postgis/postgis:17-3.5` image failed under bianpai on this Apple Silicon host with `unsupported platform ... linux/arm64`.

Runtime requirements:

```text
POSTGRES_DB=iroha
POSTGRES_USER=iroha
POSTGRES_PASSWORD=iroha_dev
published port: 5432
persistent local data volume
```

Application config should use a normal database URL:

```text
DATABASE_URL=postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable
```

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
IROHA_LOCAL_NO_AUTH
IROHA_IMPORT_TOKEN
```

## Networking Constraint

Do not design local dev around container-to-container service DNS yet.

Safer MVP shape:

```text
Postgres/PostGIS container
  -> publishes 5432 to localhost

iroha-server
  -> runs on host during early development
  -> connects to localhost:5432
```

Later, if Apple `container` networking proves reliable for this repo, the app server can also run in a container.

## Open Checks Before Implementation

Before writing runtime scripts, verify on the local machine:

```bash
container system start
container build --help
container run --help
container volume --help
container network --help
```

If local help output does not expose subcommand flags, test with a disposable image before encoding flags in scripts.

The first runtime script should be conservative and easy to replace.
