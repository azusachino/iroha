# Local development stack

The supported Apple-container profile currently runs the stateful local
dependencies:

```text
PostgreSQL 18 + PostGIS 3.6  -> localhost:5432
Valkey 8                    -> localhost:6379
```

The full profile also runs `iroha-server`, `iroha-job`, and a Caddy edge
container. Caddy serves the static Svelte app on `:5173` and proxies `/api/*`
and `/public/*` to the server over the private container network. The browser
therefore uses same-origin requests, including when opened as
`http://harus-macmini:5173`.

The profile uses explicit local-development budgets instead of Apple
container's large defaults: db `2 CPU / 2 GiB`, Valkey `1 CPU / 256 MiB`,
server `1 CPU / 512 MiB`, job `1 CPU / 1 GiB`, and web `1 CPU / 256 MiB`.
Apple `container run` currently accepts integer CPU counts only, so the small
services use one CPU rather than a fractional allocation.

The application profile is started by the same runner and shares the host
`.iroha-data` directory between the server and worker. The host-process mode
remains available for development when needed:

```bash
make db-up
make run
make run-job
```

The stack is defined in `compose.yaml` and lifecycle/migration behavior is
owned by `scripts/dev_stack.py`:

```bash
uv run python scripts/dev_stack.py start
uv run python scripts/dev_stack.py status
uv run python scripts/dev_stack.py logs
uv run python scripts/dev_stack.py stop
```

`make db-up` is the preferred entrypoint because it enters the Nix shell and
provides the pinned `goose` migration CLI. The direct `uv` command is useful
for diagnosing the container runner when the outer Nix shell is unavailable.

## Current Apple-container boundary

The native Apple `container` backend isolates each container in a lightweight
VM. bianpai can run this Compose-shaped file and publish ports, but service-name
DNS between containers is not available. The current profile therefore keeps
the application processes on the host and publishes only Postgres/Valkey.

The full application profile uses the repo-local `.iroha-data` bind mount for
raw files because Apple container VMs can share host bind mounts reliably while
named-volume attachment across isolated application VMs is not reliable on
this host. The server and worker mount it at `/data`.

## Verified on 2026-07-13

- `bianpai --backend container -f ops/local-dev/compose.yaml up -d` completed.
- Postgres accepted connections and reported PostgreSQL 18.4 on arm64.
- Goose applied migrations through version 9.
- `db` and `valkey` were both reported as running by `bianpai ps`.
