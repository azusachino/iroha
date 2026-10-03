# iroha-web

Private SvelteKit cockpit for iroha's activity data, sleep history, daily
summaries, media library, and personal control room. It consumes the
`iroha-server` API and includes per-activity and per-sleep detail pages.

This is a personal single-user viewer. The private API is unauthenticated; the
deployment's network boundary is the security control, not an application
credential. The control room can create/complete personal tasks and queue
allowlisted media sync jobs.

## Stack

- SvelteKit (TypeScript), client-rendered SPA (`adapter-static`, SSR disabled)
- MapLibre GL for the route map (key-free OpenStreetMap raster tiles)
- Modular ECharts for shared charts

Stable Bun and Node LTS come from `.mise.toml` at the repo root. Both frontend
hosts and shared UI use one native Bun workspace and frozen root lockfile.
Bun manages packages and scripts; Vite, Vitest and Playwright remain unchanged.
Vitest's V8 coverage runs on Node, not Bun's JavaScriptCore.
Use the root Make targets rather than installing each package independently.

## Configuration

The API base URL is read from `PUBLIC_IROHA_API_BASE` and defaults to
`http://127.0.0.1:8080` (where `iroha-server` listens locally). In a same-origin
container deployment, set it to an empty value so the browser uses the Caddy
`/api` proxy.

Calendar behavior is configured independently with `PUBLIC_IROHA_TIMEZONE`, an
IANA timezone such as `Asia/Tokyo` or `America/New_York`. Keep it equal to the
server's `IROHA_TIMEZONE` when the static web bundle and API are deployed
separately. The web sends this value on date-sensitive reads; API clients may
omit `timezone` and use the server default.

```bash
cp .env.example .env
# then edit PUBLIC_IROHA_API_BASE if the server is elsewhere
```

## Develop

```bash
# From the repository root:
make frontend-install
# optional: export PUBLIC_IROHA_API_BASE=http://127.0.0.1:8080
make web-dev
```

Open the printed URL (default <http://localhost:5173>). A running `iroha-server`
is needed to see data, but not to build.

## Build and check

```bash
# From the repository root:
make web-build  # production build into apps/iroha-web/build
make web-check  # svelte-check type checking
make web-test   # app/shared tests, full source coverage inventory

# From this app directory after building:
mise exec -- bun run preview
```
