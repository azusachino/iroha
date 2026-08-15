# AGENTS.md

Project conventions and orientation for contributors and coding agents working in this repo. Read this before making changes. User-facing contribution flow is in [CONTRIBUTING.md](CONTRIBUTING.md).

## What this is

A personal data cockpit. Raw exports are canonical evidence; they are normalized into a durable Postgres/PostGIS store and exposed through a private API/web app. First module: running & fitness.

## Layout

```
apps/iroha-runtime/   Shared runtime packages (cache, IDs, jobs, persistence models)
apps/iroha-server/    Go service (cmd/iroha-server, pkg/{httpapi,activities,daily,sleep,config,rawfiles})
apps/iroha-job/       Go background worker service
apps/iroha-web/       Svelte 5 + Vite web app (bun)
apps/iroha-server/db/migrations/   goose SQL migrations (00001_...)
scripts/              uv-run Python dev scripts (dev_stack.py, real_import_smoke.py, db.py)
docs/                 design docs
```

## Toolchain & tasks

- **mise-first**: every tool is pinned in `.mise.toml` (`mise install`). `make` targets run through `mise exec --`, and CI (`ci.yml`, `public-site.yml`) resolves the same tool versions — see
  `docs/dev-runtime.md`.
- **`make` is the task runner** — always reference `make <target>`. `make check` is the pre-commit gate; `make validate` is the pre-PR gate (both enforced by local hooks).
- Migrations run through **goose** via `make db-up` / `make db-reset` (which call `scripts/dev_stack.py`). There is **no** GORM AutoMigrate — the SQL migration is the source of truth and the
  hand-written structs in `internal/models/models.go` must match it.

## Conventions

- **Uber Go Style Guide** orientation, enforced by `golangci-lint` (`.golangci.yml`), formatted with `gofumpt`. Run `make lint`.
- **Named constants, not magic literals** — versions/defaults/status/table-names/env-keys live in a single `const` (e.g. `imports.DefaultParserVersion`); never re-inline the literal.
- **2-space indent** for YAML/TOML/JSON; tabs for Go.
- **Conventional commits**, no emojis. Stage specific files, never `git add -A`.
- Prefer extracting **pure, DB-free helpers** so logic is unit-testable without Postgres (see `imports/decision.go`, `parsers` key/hash helpers).

## Theme asset boundary (hard rule)

Iroha's registered design languages and adopted design compositions are core product assets, not `iroha-web` implementation details. The source of truth for design identities, the registry,
theme-specific compositions, shared visual primitives, charts, controls, and theme-aware presentation components **must live under `packages/`** (currently `packages/iroha-shared/`, or a dedicated
package when the boundary is split). They must not be created or maintained solely under `apps/iroha-web/src/` or `apps/iroha-public-site/src/`.

The applications are adapters and hosts:

- `iroha-web` owns API clients, route state, loading/error behavior, and navigation callbacks. It may adapt canonical API data into the shared view contracts, but it must not own a theme composition
  or duplicate a theme primitive.
- `iroha-public-site` owns its static content/data adapter and site shell. It consumes the same shared theme assets; it must not fork a visual component because its data is static.
- Shared theme code accepts typed data, snippets, and callbacks. It must not import `$lib/api`, route modules, server packages, or another application's source path.

The current `apps/iroha-web/src/lib/themes/` tree is migration debt. Do not add new theme files there or add a web-local shared visual primitive to work around the debt. Before adding a component,
search `packages/` first and decide whether it is an app adapter or a reusable theme asset. A review is incomplete unless it checks both file placement and import direction:

```bash
rg -n 'src/lib/themes|from "\$lib/(api|routes)' apps packages
rg -n 'THEME_IDS|THEME_IDENTITIES|ThemeRoute|ThemeDefinition' apps packages
```

Design identities, route registrations, design compositions, and theme-aware CSS must have one canonical definition. The registry is extensible: a new design must be promoted into the shared registry
and receive a deliberate shared implementation before it is considered adopted. A web-local copy, wrapper that changes the design language, or a CSS switch without a deliberate composition is a
boundary violation.

Some `theme-ui/<language>/` files (`Media.svelte`, `Shell.svelte`, and similar) are six independent per-language copies rather than one parameterized component, because their content genuinely differs
(labels, widths, decoration). That does not make them exempt from "one canonical definition" for whatever _is_ shared across them — a loading guard, a filter-chip set, a width convention. Before
calling a fix in one theme's copy complete, `rg` the component's filename across all six `theme-ui/*/` directories and check whether the same fix applies there too.

## Loading state (hard rule)

Route-level async data goes through `apps/iroha-web/src/lib/asyncResource.svelte.ts` (`createAsyncResource`) — never a hand-rolled `loading`/`error` pair plus a manual request-id counter.
`LoadingBoundary` takes a `resource`/`resource[]` prop, not `loading`/`ready` booleans, specifically so a caller cannot recompute `ready` as `!loading`: that expression is non-sticky and flips back to
`false` on every refetch, which re-triggers the boundary's first-load overlay (and its `inert`/`aria-hidden` content) on a routine period or filter change instead of the intended quiet "Updating…"
pill. This was a real, shipped bug on four routes before `AsyncResource` made the mistake unrepresentable — see the `v0.4.1` release audit's "Post-tag hardening" section. If a route needs `ready`
outside `LoadingBoundary` (a fallback branch, a button's disabled state), read it off the resource (`resource.ready`); never re-derive it from `loading`.

## Deployment scope

Classify the changed paths before building a local k3s image. A web-only visual or route change uses `make image-web VERSION=<tag>` and the k3s repo's `make apply-iroha-web`; it does not use
`make images` or `make apply-iroha`. Use the full image/apply path only when server, job, export, configuration, or migration changes require it. If a shared package has multiple runtime consumers,
build and verify each affected consumer explicitly.

## Data & import model (important)

- A full Apple Health export is a **complete snapshot**, reconciled — not appended.
- Workout identity is a **stable source key** (`sourceName|normalized-device|type|start|end|duration`), **not** the zip hash. The HKDevice string carries a volatile `0x` pointer + creation date that
  must be stripped before use in keys/hashes.
- `tb_import_snapshots` (per export) + `tb_apple_source_items` (per source record, with `content_hash`) drive skip-unchanged / upsert-changed / insert-new reconciliation.
- Routes are attached to their owning workout; selected `Record` streams become `tb_activity_samplings` via a second streaming pass (records precede workouts in `export.xml`, so window-association is
  a two-pass, binary-searched lookup — keep it streaming; do not buffer the ~900MB file).
- **Reprocess**: a completed import at a _different_ `parser_version` purges everything derived from the raw file (`apple_source_items` → snapshots → activities-cascade, in that order) then
  re-persists. Deleting source items **first** is load-bearing — otherwise their `content_hash` makes change detection skip re-creating workouts (silent data loss). `parser_version` is
  `IROHA_PARSER_VERSION` (default `imports.DefaultParserVersion`); bump it when parser semantics change.

## Verification

- Unit tests: `make test`. DB-backed: `make test-integration`.
- Real end-to-end: run both the server (`make run`) and the background worker (`make run-job`) with a shared `IROHA_DATA_DIR` environment variable, then run `make smoke-real-import FILE=...`;
  `real_import_smoke.py` also has `--assert` (delta / no-dup checks) and `--assert-reprocess` modes.
- Never run `make db-reset` casually — it wipes locally imported data.

## Gotchas

- Root `.gitignore` was once a Python template; over-broad rules (e.g. bare `lib/`) silently ignored `apps/iroha-web/src/lib`. Watch template rules against this TS/Go monorepo.
