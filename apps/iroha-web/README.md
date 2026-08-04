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
