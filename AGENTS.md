# AGENTS.md

Project conventions and orientation for contributors and coding agents working in this repo.
Read this before making changes. User-facing contribution flow is in [CONTRIBUTING.md](CONTRIBUTING.md).

## What this is

A personal data cockpit. Raw exports are canonical evidence; they are normalized into a durable
Postgres/PostGIS store and exposed through a private API/web app. First module: running & fitness.

## Layout

```
apps/iroha-server/    Go service (cmd/iroha-server, internal/{httpapi,imports,parsers,activities,models,config,ids})
apps/iroha-web/       Svelte 5 + Vite web app (bun)
apps/iroha-server/db/migrations/   goose SQL migrations (00001_...)
scripts/              uv-run Python dev scripts (dev_stack.py, real_import_smoke.py, db.py)
docs/                 design docs
```

## Toolchain & tasks

- **Nix-first**: every tool comes from the devShell (`nix develop`). `make` targets auto-wrap into
  it via `NIX_DEV` when run outside the shell.
- **`make` is the task runner** — always reference `make <target>`. `make check` is the pre-commit
  gate; `make validate` is the pre-PR gate (both enforced by local hooks).
- Migrations run through **goose** via `make db-up` / `make db-reset` (which call `scripts/dev_stack.py`).
  There is **no** GORM AutoMigrate — the SQL migration is the source of truth and the hand-written
  structs in `internal/models/models.go` must match it.

## Conventions

- **Uber Go Style Guide** orientation, enforced by `golangci-lint` (`.golangci.yml`), formatted with
  `gofumpt`. Run `make lint`.
- **Named constants, not magic literals** — versions/defaults/status/table-names/env-keys live in a
  single `const` (e.g. `imports.DefaultParserVersion`); never re-inline the literal.
- **2-space indent** for YAML/TOML/JSON; tabs for Go.
- **Conventional commits**, no emojis. Stage specific files, never `git add -A`.
- Prefer extracting **pure, DB-free helpers** so logic is unit-testable without Postgres (see
  `imports/decision.go`, `parsers` key/hash helpers).

## Data & import model (important)

- A full Apple Health export is a **complete snapshot**, reconciled — not appended.
- Workout identity is a **stable source key** (`sourceName|normalized-device|type|start|end|duration`),
  **not** the zip hash. The HKDevice string carries a volatile `0x` pointer + creation date that must
  be stripped before use in keys/hashes.
- `tb_import_snapshots` (per export) + `tb_apple_source_items` (per source record, with `content_hash`)
  drive skip-unchanged / upsert-changed / insert-new reconciliation.
- Routes are attached to their owning workout; selected `Record` streams become `tb_activity_samplings`
  via a second streaming pass (records precede workouts in `export.xml`, so window-association is a
  two-pass, binary-searched lookup — keep it streaming; do not buffer the ~900MB file).
- **Reprocess**: a completed import at a *different* `parser_version` purges everything derived from
  the raw file (`apple_source_items` → snapshots → activities-cascade, in that order) then re-persists.
  Deleting source items **first** is load-bearing — otherwise their `content_hash` makes change
  detection skip re-creating workouts (silent data loss). `parser_version` is `IROHA_PARSER_VERSION`
  (default `imports.DefaultParserVersion`); bump it when parser semantics change.

## Verification

- Unit tests: `make test`. DB-backed: `make test-integration`.
- Real end-to-end: run the server, then `make smoke-real-import FILE=...`; `real_import_smoke.py`
  also has `--assert` (delta / no-dup checks) and `--assert-reprocess` modes.
- Never run `make db-reset` casually — it wipes locally imported data.

## Gotchas

- The flake `shellHook` must write to **stderr**, not stdout — stdout pollutes `nix develop --command`
  captured output (once broke `make fmt-check`).
- Root `.gitignore` was once a Python template; over-broad rules (e.g. bare `lib/`) silently ignored
  `apps/iroha-web/src/lib`. Watch template rules against this TS/Go monorepo.
