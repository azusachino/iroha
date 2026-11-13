# Local development stack

The supported Podman Compose profile runs the stateful local dependencies:

```text
PostgreSQL 18 + PostGIS 3.6  -> localhost:5432

The local database uses docker.io/postgis/postgis:18-3.6-alpine — the
same image the k3s cluster runs for iroha-postgis, so local development and
production share one PostGIS build (see ops/k8s/README.md).
Cache                       -> Postgres (`tb_cache_entries`)
```

The full profile also runs `iroha-server`, `iroha-job`, and a Caddy edge container. Caddy serves the static Svelte app on `:5173` and proxies `/api/*` and `/public/*` to the server over the private
container network. The browser therefore uses same-origin requests, including when opened as `http://harus-macmini:5173`.

The profile uses explicit local-development budgets: db `2 CPU / 2 GiB`, server `1 CPU / 512 MiB`, job `1 CPU / 1 GiB`, and web `1 CPU / 256 MiB`.

The private API (`/api/v1`) requires an owner session. On a fresh database the web app at <http://127.0.0.1:5173> shows a one-time setup screen that creates the owner account; after that it shows a
login screen. See [Auth](../../docs/iroha-server.md#auth). Set `IROHA_ALLOWED_ORIGINS` to the web origin(s) that should be allowed to call it; do not expose `iroha-server` directly to an untrusted
network.

Health Auto Export intake is refused with `503` until an intake token is issued. Issue one on the Admin page, or against the local database with
`go -C apps/iroha-server run ./cmd/iroha-admin intake-token issue`; see [the setup guide](../../docs/health-auto-export-setup.md).

The application profile is started by the same runner and shares the host `.iroha-data` directory between the server and worker. The host-process mode remains available for development when needed:

```bash
make db-up       # dependencies + migrations, for host-process development
make dev-up      # all services in containers
make run
make run-job
```

To check the private app from another device on the LAN/Tailscale network, use the published web port (the compose service already binds it to all host interfaces):

```bash
make dev-up
make db-status
podman-compose -f ops/local-dev/compose.yaml -f ops/local-dev/compose.app.yaml -p iroha-dev ps
```

Open `http://<machine-ip>:5173/motion/<activity-id>` in the other browser. The standalone public snapshot can be checked separately with:

```bash
make public-site-dev PORT=5174  # http://<machine-ip>:5174
make public-site-preview HOST=0.0.0.0 PORT=4173  # production-like build
```

The private host-process frontend uses the same all-interface binding: `make web-dev`, then open `http://<machine-ip>:5173/motion/<activity-id>`.

The stack is defined in `compose.yaml` and lifecycle/migration behavior is owned by `scripts/dev_stack.py`:

```bash
uv run python scripts/dev_stack.py start
uv run python scripts/dev_stack.py status
uv run python scripts/dev_stack.py logs
uv run python scripts/dev_stack.py stop
```

`make db-up` is the preferred dependency entrypoint because it uses the pinned SQLx CLI from the active mise toolchain. The direct `uv` command is useful for diagnosing the runner when the mise
environment is unavailable.

## Podman boundary

Development happens on the Linux workstation, where Podman runs containers
natively — no Podman machine is involved. `podman-compose` provides the private
project network, so server and worker reach the database at the `db` service
name rather than generated host IPs.

The full application profile uses the repo-local `.iroha-data` bind mount for
raw files, mounted at `/data` by the server and worker; on Linux this is a
plain host bind mount. The database lives in the named `iroha-dev_pgdata`
volume managed by the stack.

On macOS the same profile works through a Podman machine: create one with a
deliberately small disk (`podman machine init --disk-size 30`) and start it
before using the stack; the runner refuses to initialize a machine implicitly.
