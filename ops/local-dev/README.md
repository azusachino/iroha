# Local development stack

The supported Podman Compose profile runs the stateful local dependencies:

```text
PostgreSQL 18 + PostGIS 3.6  -> localhost:5432

The local database uses the pinned multi-architecture image
`docker.io/kartoza/postgis:18.4-3.6.4--v2026.06.21`.
Cache                       -> Postgres (`tb_cache_entries`)
```

The full profile also runs `iroha-server`, `iroha-job`, and a Caddy edge container. Caddy serves the static Svelte app on `:5173` and proxies `/api/*` and `/public/*` to the server over the private
container network. The browser therefore uses same-origin requests, including when opened as `http://harus-macmini:5173`.

The profile uses explicit local-development budgets: db `2 CPU / 2 GiB`, server `1 CPU / 512 MiB`, job `1 CPU / 1 GiB`, and web `1 CPU / 256 MiB`.

The private API (`/api/v1`) is unauthenticated by design: iroha is a single-user personal deployment, and the network boundary (private LAN/NAS, not exposed publicly) is the security boundary rather
than an application-level credential. Set `IROHA_ALLOWED_ORIGINS` to the web origin(s) that should be allowed to call it; do not expose `iroha-server` directly to an untrusted network.

The application profile is started by the same runner and shares the host `.iroha-data` directory between the server and worker. The host-process mode remains available for development when needed:

```bash
make db-up       # dependencies + migrations, for host-process development
make dev-up      # all services in containers
make run
make run-job
```

To check the private app from another device on the LAN/Tailscale network, use
the published web port (the compose service already binds it to all host
interfaces):

```bash
make dev-up
make db-status
podman-compose -f ops/local-dev/compose.yaml -f ops/local-dev/compose.app.yaml -p iroha-dev ps
```

Open `http://<machine-ip>:5173/motion/<activity-id>` in the other browser.
The standalone public snapshot can be checked separately with:

```bash
make public-site-dev PORT=5174  # http://<machine-ip>:5174
make public-site-preview HOST=0.0.0.0 PORT=4173  # production-like build
```

The private host-process frontend uses the same all-interface binding:
`make web-dev`, then open `http://<machine-ip>:5173/motion/<activity-id>`.

The stack is defined in `compose.yaml` and lifecycle/migration behavior is owned by `scripts/dev_stack.py`:

```bash
uv run python scripts/dev_stack.py start
uv run python scripts/dev_stack.py status
uv run python scripts/dev_stack.py logs
uv run python scripts/dev_stack.py stop
```

`make db-up` is the preferred dependency entrypoint because it uses the pinned `goose` migration CLI from the active mise toolchain. The direct `uv` command is useful for diagnosing the runner when
the mise environment is unavailable.

## Podman boundary

On macOS, Podman isolates containers in a Podman machine. `podman-compose` provides the private project network, so server and worker use the `db` service name rather than generated host IPs.

The full application profile uses the repo-local `.iroha-data` bind mount for raw files because Apple container VMs can share host bind mounts reliably while named-volume attachment across isolated
application VMs is not required by the Podman profile. The server and worker mount it at `/data`.

The runner refuses to initialize a machine implicitly. If no machine exists, create one with a deliberately small disk, for example `podman machine init --disk-size 30`, then run
`podman machine start`.
