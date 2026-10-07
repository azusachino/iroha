# Public Site: Activity Heatmap & Fast Static-Style Loading

- **Date:** 2026-10-07
- **Goal:** Transform `apps/iroha-public-site` into an instant-loading, static-style publication on mobile and desktop by integrating `ActivityHeatmap` and eliminating multi-megabyte bundle and GeoJSON network bottlenecks from the critical path.
- **Repository:** `vendor/iroha` (`apps/iroha-public-site`, `packages/iroha-shared`)
- **Git Branch:** `feat/public-site-heatmap-perf`

---

## 1. Problem Statement & Objectives

### Current Latency on Mobile Chrome
1. **Critical JS Payload > 2.1 MB:** Static imports of `RoutesMap` (MapLibre GL > 1MB + 510KB worker) and `MonthlyBarChart` / `YearProgressChart` (ECharts + ZRender > 550KB) force Vite to bundle all visualization engines into the entry chunk.
2. **Blocking Network Waterfall:** `+page.ts` blocks initial render until `/public/v1/routes` completes, transferring the entire GeoJSON coordinate archive across mobile networks before anything paints.
3. **Blank Shell Flash:** `app.html` renders only `<p id="public-bootstrap-status">Loading public activity data…</p>` while waiting for JS and network.

### Target Architecture
1. **Instant Paint (<200ms):** Initial entry bundle under 80 KB gzipped.
2. **Heatmap as Centerpiece:** Pure Svelte/CSS Grid `<ActivityHeatmap />` (0 external dependencies) renders immediately with summary tiles and activity list.
3. **Progressive Enhancement:** MapLibre GL and ECharts chunks load lazily on-demand.
4. **Preserve k3s Simplicity:** Zero changes to Caddy deployment or Kubernetes manifests (stays 32Mi RAM request).

---

## 2. Tasks & Responsibilities

### Phase 1: Heatmap Aggregation & Layout Integration
- **Task 1: Add Daily Aggregation Helper & Tests**
  - File: `apps/iroha-public-site/src/lib/aggregate.ts`, `aggregate.test.ts`
  - Function: `activeDaysFromActivities(activities: Activity[]): ActivityActiveDay[]`
- **Task 2: Mount ActivityHeatmap in Public Site Layout**
  - File: `apps/iroha-public-site/src/routes/+page.svelte`
  - Mount `<ActivityHeatmap days={activeDays} endDay={latestDay} embedded />`

### Phase 2: Decouple Critical Path & Progressive Code-Splitting
- **Task 3: Dynamic Import for Heavy Engines (ECharts & MapLibre)**
  - File: `apps/iroha-public-site/src/routes/+page.svelte`
  - Code-split `RoutesMap`, `MonthlyBarChart`, and `YearProgressChart` via dynamic imports (`{#await import(...) }`).
- **Task 4: Defer `/public/v1/routes` GeoJSON Fetch**
  - File: `apps/iroha-public-site/src/routes/+page.ts`, `+page.svelte`
  - Load `summary` and `activities` in `load()`; fetch `routes` lazily when map is mounted/requested.
- **Task 5: Inlined Critical Skeleton in `app.html`**
  - File: `apps/iroha-public-site/src/app.html`
  - Add styled dark-theme header and tile skeleton to eliminate white flash and blank delay.

### Phase 3: Verification & Performance Gates
- **Task 6: Build, Bundle Size & Test Verification**
  - Verify chunk sizes (`bun run build`), run `make check`, run e2e specs (`public-states.spec.ts`).

---

## 3. Success Criteria
- [ ] Initial bundle entry size drops from >2.1MB to <100KB.
- [ ] `<ActivityHeatmap />` displays active days and intensities accurately.
- [ ] Maps and ECharts charts load smoothly on-demand without layout breaks.
- [ ] `make check` and all public-site tests pass with zero errors.
