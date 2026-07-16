# Local development stack

The supported Podman Compose profile runs the stateful local dependencies:

```text
PostgreSQL 18 + PostGIS 3.6  -> localhost:5432
Valkey 8                    -> localhost:6379
```

The full profile also runs `iroha-server`, `iroha-job`, and a Caddy edge container. Caddy serves the static Svelte app on `:5173` and proxies `/api/*` and `/public/*` to the server over the private
container network. The browser therefore uses same-origin requests, including when opened as `http://harus-macmini:5173`.

The profile uses explicit local-development budgets: db `2 CPU / 2 GiB`, Valkey `1 CPU / 256 MiB`, server `1 CPU / 512 MiB`, job `1 CPU / 1 GiB`, and web `1 CPU / 256 MiB`.

The application profile is started by the same runner and shares the host `.iroha-data` directory between the server and worker. The host-process mode remains available for development when needed:

```bash
make db-up       # dependencies + migrations, for host-process development
make dev-up      # all services in containers
make run
make run-job
```

The stack is defined in `compose.yaml` and lifecycle/migration behavior is owned by `scripts/dev_stack.py`:

```bash
uv run python scripts/dev_stack.py start
uv run python scripts/dev_stack.py status
uv run python scripts/dev_stack.py logs
uv run python scripts/dev_stack.py stop
```

`make db-up` is the preferred dependency entrypoint because it enters the Nix shell and provides the pinned `goose` migration CLI. The direct `uv` command is useful for diagnosing the runner when the
outer Nix shell is unavailable.

## Podman boundary

On macOS, Podman isolates containers in a Podman machine. `podman-compose` provides the private project network, so server and worker use `db` and `valkey` service names rather than generated host
IPs.

The full application profile uses the repo-local `.iroha-data` bind mount for raw files because Apple container VMs can share host bind mounts reliably while named-volume attachment across isolated
application VMs is not required by the Podman profile. The server and worker mount it at `/data`.

The runner refuses to initialize a machine implicitly. If no machine exists, create one with a deliberately small disk, for example `podman machine init --disk-size 30`, then run
`podman machine start`.
