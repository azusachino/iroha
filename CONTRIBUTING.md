# Contributing to iroha

Thanks for your interest in improving iroha. This guide covers the development workflow, coding standards, and the pull-request process. Deeper project conventions (for both humans and coding agents)
live in [AGENTS.md](AGENTS.md).

## Development environment

Tooling is pinned by the checked-in `.mise.toml` — do not install Go, goose, bun, or Postgres separately.

```sh
mise install
```

`make` targets run through `mise exec --`, so the normal workflow is just `make <target>` after `mise install`. CI (`ci.yml`, `public-site.yml`) provisions the same `.mise.toml` tools via
`jdx/mise-action`, so local and CI resolve identical versions. See `docs/dev-runtime.md` for the runtime contract.

`uv` manages the Python used only by dev scripts.

## Common tasks

The `Makefile` is the task runner — prefer `make <target>` over raw commands.

| Command                           | What it does                                                       |
| --------------------------------- | ------------------------------------------------------------------ |
| `make check`                      | Pre-commit gate: `fmt-check` + `vet` + `lint` + tests + web checks |
| `make validate`                   | Pre-PR gate: `check` + full server and web builds                  |
| `make lint`                       | Run `golangci-lint`                                                |
| `make test`                       | Go unit tests                                                      |
| `make test-integration`           | DB-backed integration tests (starts the dev database)              |
| `make db-up` / `make db-down`     | Start / stop Postgres + PostGIS (applies migrations)               |
| `make smoke-real-import FILE=...` | Import a real file end to end through the HTTP API                 |

`make check` runs before every commit and `make validate` before PRs (enforced locally by hooks).

## Code style

Go code is oriented from the **[Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)** and enforced by `golangci-lint` (see `.golangci.yml`). Formatting is `gofumpt` (a stricter
`gofmt`). Run `make lint` before pushing.

House rules that go beyond the linter:

- **Named constants, not magic literals.** Versions, defaults, status strings, table names, and env keys live in one named `const` as the single source of truth (e.g. `imports.StatusCompleted`,
  `imports.DefaultParserVersion`); reference the const, never re-inline the literal.
- **Raw files are canonical.** Never mutate imported evidence; derive from it. Reprocessing a source replaces its derived rows, it does not append.
- **Config files** (YAML, TOML, JSON) use **2-space** indentation; Go uses tabs (gofumpt).
- Prefer small, testable pure helpers for logic that would otherwise need a database to exercise.

## Commit conventions

[Conventional Commits](https://www.conventionalcommits.org), no emojis:

```
feat:     a new capability
fix:      a bug fix
chore:    tooling / housekeeping
refactor: behavior-preserving restructuring
docs:     documentation only
test:     tests only
```

Keep commits focused; stage specific files rather than `git add -A`.

## Pull requests

1. Branch off `main`.
2. Make the change with matching tests; keep the diff scoped.
3. Run `make validate` — it must pass.
4. Open a PR against `main` using the template; describe the change, the verification you ran, and any follow-ups.

## Reporting bugs & security

- Functional bugs and feature ideas: open an issue using the templates.
- Security vulnerabilities: **do not** open a public issue — see [SECURITY.md](SECURITY.md).

By contributing you agree that your contributions are licensed under the project's [AGPL-3.0](LICENSE) license.
