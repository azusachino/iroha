# iroha-web

Private, read-only SvelteKit viewer for the iroha activity data. It consumes the
`iroha-server` read API and renders an activity list plus a per-activity detail
page with a MapLibre route map and uPlot pace / heart-rate / elevation charts.

This is a personal single-user viewer. It sends an optional deployment-provided
JWT bearer header and performs no
writes.

## Stack

- SvelteKit (TypeScript), client-rendered SPA (`adapter-static`, SSR disabled)
- MapLibre GL for the route map (key-free OpenStreetMap raster tiles)
- uPlot for the line charts

Node/bun tooling comes from the Nix devShell (`nix develop`). Use `bun`, not
npm/pnpm/yarn.

## Configuration

The API base URL is read from `PUBLIC_IROHA_API_BASE` and defaults to
`http://127.0.0.1:8080` (where `iroha-server` listens locally). In a same-origin
container deployment, set it to an empty value so the browser uses the Caddy
`/api` proxy.

```bash
cp .env.example .env
# then edit PUBLIC_IROHA_API_BASE if the server is elsewhere
```

`PUBLIC_IROHA_API_TOKEN` is an optional deployment-provided, read-only JWT for
authenticated private API mode. It is compiled into the static browser bundle
and is therefore not a secret. Do not use a signing secret or an
`iroha:write` token here. The web image accepts it as the
`PUBLIC_IROHA_API_TOKEN` build argument.

## Develop

```bash
bun install
# optional: export PUBLIC_IROHA_API_BASE=http://127.0.0.1:8080
bun run dev
```

Open the printed URL (default http://localhost:5173). A running `iroha-server`
is needed to see data, but not to build.

## Build and check

```bash
bun run build   # production build into ./build
bun run check   # svelte-check type checking
bun run preview # serve the production build locally
```
