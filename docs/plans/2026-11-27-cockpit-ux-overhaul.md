# Cockpit UX overhaul: structure before tokens

Date: 2026-11-27. Status: Owner-approved direction & implementation plan (issue [#236](https://github.com/azusachino/iroha/issues/236))
Target workspace: `packages/iroha-shared` & `apps/iroha-web`
Reference baseline: `287e66677ce2d3c12b3a306ed503329e3b3630e2`

Batch 1 (tinted-canvas tokens, type/spacing/radius/control/depth scales, and shell/primitive token adoption in commits `39fe7efc`, `935d4e9c`, and `287e6667`) is already merged into this branch.

---

## 1. Symptom priority, corrected

Batch 1 lifted light surfaces onto the tinted canvas, yet rendered screens looked largely unchanged because layout clutter and repeated figures dominated. Symptoms are therefore re-ordered:

1. **Busy and noisy** — repeated scope banners, duplicated rollup rows, empty placeholder tiles, and per-chart chrome compete for attention on every screen.
2. **Inconsistent** — each route invents its own grid spans, pastes table CSS locally, and renders `h1` and card titles at the same visual weight.
3. **Cramped and small** — six-column tile rows wrap to four or five lines and bury the primary signal among secondary figures.
4. **Flat and grey** — addressed in batch 1 via `#eef4f4` canvas, `#ffffff` cards, and `--tile-shadow`; kept as the foundation, not the primary fix.

## 2. The rejected premise

**Rejected premise:** that hierarchy is a bottom-up consequence of tokens, and that every existing card, scope banner, and metadata line survives the overhaul. Crisper cards on a tinted canvas only frame the clutter more sharply.

Evidence from rendered fixtures and source inspection:

- **Rendered fixtures:** Expenses repeats `Observed period: 2026-08` four times, shows four currency tiles of which three read `—` / *Record count unavailable*, and gives each of two charts *View chart data* plus provenance plus `CHART | TABLE | CSV`. Night stacks a summary row directly above a second card row repeating `1 / 7:00:00 / 88%`. Overview fits six equal narrow tiles that each wrap to four or five lines.
- **Code anchors:** every `h1` takes one size (`packages/iroha-shared/src/theme/themes.css:138`); the same table CSS is pasted into four compositions (`Activities.svelte:371-383`, `Sleep.svelte:277-292`, `Today.svelte:453-463`, `Dashboard.svelte:644-656`) while `packages/iroha-shared/src/components/MetricTable.svelte:67-68` goes unused; nothing shared decides hierarchy, each route sets its own grid spans (`Dashboard.svelte:445-460`, `Today.svelte:359`); `StatTile.featured` has no caller, and every Dashboard tile passes `compact` (`Dashboard.svelte:107-165`).

---

## 3. Preserved foundation and hard rules

Keep the batch 1 tokens and scales unchanged:

- **Light tokens:** `--bg: #eef4f4`, `--surface: #ffffff`, `--surface-2: #f4f8f8`, `--border: #d7e3e2`, `--text: #12201f`, `--text-muted: #4b5f5e`, `--accent: #076660` (`6.13:1` on `#eef4f4`, clearing the `5.04:1` focus floor), `--accent-2: #be123c`, `--tile-shadow: 0 2px 4px rgb(10 40 38 / 0.05), 0 16px 40px rgb(10 40 38 / 0.1)`. Dark mode keeps `#0b0f19` / `#121826` / `#1b2438`.
- **Scales:** `--type-caption` (`0.75rem` floor) through `--type-hero` (`clamp(2rem, 4vw, 2.75rem)`), `--space-1`..`--space-10`, `--gutter-page`, `--gap-section`, `--gap-grid`, `--pad-card`, `--radius-*`, `--control-height*`, and `--tile-rim`.
- **Hard rules:**
  - **Theme asset boundary** (`AGENTS.md`, `make theme-boundary-check`): all shared primitives, tokens, and Grapher compositions live under `packages/iroha-shared/`. Never create `apps/iroha-web/src/lib/themes/` or import `$lib/api` or route modules from `packages/`.
  - **Loading state** (`apps/iroha-web/src/lib/asyncResource.svelte.ts`): `LoadingBoundary` takes `resource`/`resource[]`; never re-derive `ready` from `!loading`.
  - **Month/year/lifetime scope** (`read_scope.go`, `scope.ts`): every rendered `PeriodSelector` option must work end-to-end; unsupported levels stay hidden (`showAllYears={false}` / `showAllMonths={false}`, as on Reports).
  - **Motion, responsive, and quality floor** (`CONSTRAINTS.md`): honor `--motion-*` (`make motion-tokens-check`), canonical breakpoints (`make responsive-check`), and `make quality-floor-check`.

---

## 4. `FocalBand` specification

`packages/iroha-shared/src/components/FocalBand.svelte` is the single top-of-page hierarchy primitive. It replaces per-route span grids and duplicated rollup rows, while `MetricTable.svelte` replaces the four pasted table CSS blocks.

- **Props:**
  - `heading: string` — section or region label for the band.
  - `primary: Snippet | { label: string; value: string; unit?: string; detail?: string }` — the single `--type-hero` focal metric.
  - `secondary: Array<{ label: string; value: string; detail?: string }> | Snippet` — two or three non-empty secondary key-value figures.
  - `empty?: string | Snippet` — optional muted summary line when items are empty (e.g. `"3 currencies empty: USD, EUR, GBP"`).
- **Boundary contract:** accepts typed data, snippets, and callbacks only; never imports `$lib/api` or a route module.
- **Grid:** asymmetric `2fr 1fr` at `>= 1024px`, stacked (`1fr`) below `1024px`.
- **Tokens:** `--gap-section`, `--gap-grid`, `--pad-card`, `--radius` / `--radius-lg`, `--tile-shadow`, `--tile-rim`, `--type-hero` (largest step inside the band; `h1` page titles and card titles use distinct type steps so they never share a size).
- **Accessibility:** the primary metric renders inside a real heading or `aria-labelledby` region; the `1fr` secondary card renders as a semantic `<dl>` (`<dt>`/`<dd>`).

---

## 5. Per-screen cut table

| Route | Repeated / redundant today | Moves into `FocalBand` | Muted empty line | Stays |
| --- | --- | --- | --- | --- |
| Today | Per-route span grid (`Today.svelte:359`); pasted table CSS (`Today.svelte:453-463`) | Primary daily signal (`2fr`) + 2–3 non-empty ring/day figures (`1fr`) | Any zero/unavailable daily ring or metric | Ring gauges, session list via `MetricTable` |
| Overview | Six narrow `compact` tiles wrapping 4–5 lines (`Dashboard.svelte:107-165`); custom spans (`445-460`); pasted table CSS (`644-656`) | Period distance focal metric (`2fr`) + 2–3 non-empty domain totals (`1fr`) | Empty domain/period counts | Activity heatmap, monthly chart, recent table via `MetricTable` |
| Motion | Dense summary row; pasted table CSS (`Activities.svelte:371-383`) | Period total distance/time (`2fr`) + 2–3 non-empty activity counts (`1fr`); detail headline on `/motion/:id` | Empty sport/filter buckets | Pill filters, activity table via `MetricTable`, detail splits/map |
| Night | Summary row stacked above second card row repeating `1 / 7:00:00 / 88%`; pasted table CSS (`Sleep.svelte:277-292`) | Primary sleep duration (`2fr`) + 2–3 non-empty quality/stage figures (`1fr`); second row cut | Missing stage/night metrics | Circadian timeline, stage aggregate chart, nights table via `MetricTable` |
| Expenses | `Observed period: 2026-08` repeated 4x; 4 currency tiles (3 empty); per-chart *View chart data* + provenance + `CHART \| TABLE \| CSV` | Single scope line; primary period spend (`2fr`) + non-empty currency totals (`1fr`) | Empty currencies (e.g. `"3 currencies empty: USD, EUR, GBP"`) | Category/currency charts (shared single toggle), `ExpenseLedger` |
| Reports | To confirm in the slice (check `Reports.svelte` for repeated scope/metric cards) | Primary monthly headline (`2fr`) + 2–3 non-empty report facts (`1fr`) | Empty report sections | Month-only `PeriodSelector` (`showAllMonths={false}`), `ReportFactGrid` |
| Library | To confirm in the slice (check `Media.svelte` / `MediaDetail.svelte` for redundant summary/chrome) | Primary library/period count (`2fr`) + 2–3 non-empty status counts (`1fr`) | Empty media statuses | `MediaAssetCard` grid/list and detail view |
| Patterns | To confirm in the slice (check `Daily.svelte` for repeated scope/table CSS) | Primary daily pattern metric (`2fr`) + 2–3 non-empty trend figures (`1fr`) | Unavailable daily metrics | Pattern charts and daily breakdown |
| To-Go | To confirm in the slice (audit `apps/iroha-web/src/routes/to-go/+page.svelte` vs shared `packages/` composition) | Primary open/due count (`2fr`) + 2–3 non-empty list counts (`1fr`) | Empty list/filter states | Task list and completion controls |

---

## 6. Implementation slices (ordered)

Execute in order. Every slice passes its done-when Makefile targets before the next starts.

### Slice 1: `FocalBand`, `MetricTable` adoption, and type voice

- **Owns:** `packages/iroha-shared/src/components/FocalBand.svelte`, `packages/iroha-shared/src/components/MetricTable.svelte`, `packages/iroha-shared/src/components/StatTile.svelte`, `packages/iroha-shared/src/theme/themes.css`.
- **Changes:** Build `FocalBand.svelte` (`2fr 1fr` at `>= 1024px`, `<dl>` secondary card, muted empty slot). Establish distinct `h1` vs card-title vs `--type-hero` type steps in `themes.css`. Prepare `MetricTable.svelte` so routes drop pasted table CSS.
- **Moves:** **inconsistent**, **cramped and small**, **busy and noisy**.
- **Done when:** `make theme-boundary-check responsive-check motion-tokens-check web-fmt-check web-check web-test`

### Slice 2: Today and Overview

- **Owns:** `packages/iroha-shared/src/theme-ui/grapher/Today.svelte`, `packages/iroha-shared/src/theme-ui/grapher/Dashboard.svelte`, `packages/iroha-shared/src/theme-ui/components/ActivityHeatmap.svelte`, `packages/iroha-shared/src/theme-ui/components/RingGauge.svelte`.
- **Changes:** Replace per-route span grids and Dashboard's six-column `compact` tile row with `FocalBand`. Replace pasted table CSS with `MetricTable`. Collapse empty tiles into the muted empty line.
- **Moves:** **busy and noisy**, **cramped and small**, **inconsistent**.
- **Done when:** `make theme-boundary-check responsive-check web-check web-test`

### Slice 3: Night and Motion

- **Owns:** `packages/iroha-shared/src/theme-ui/grapher/Sleep.svelte`, `packages/iroha-shared/src/theme-ui/grapher/Activities.svelte`, `packages/iroha-shared/src/theme-ui/grapher/ActivityDetail.svelte`, `packages/iroha-shared/src/theme-ui/components/SleepTimelineChart.svelte`, `packages/iroha-shared/src/theme-ui/components/SleepAggregateChart.svelte`, `packages/iroha-shared/src/theme-ui/components/SleepScopeSummary.svelte`.
- **Changes:** Remove Night's duplicated second rollup row (`1 / 7:00:00 / 88%`) and adopt `FocalBand` on Night and Motion (plus headline metric on `/motion/:id`). Replace pasted table CSS in `Sleep.svelte` and `Activities.svelte` with `MetricTable`.
- **Moves:** **busy and noisy**, **inconsistent**, **cramped and small**.
- **Done when:** `make theme-boundary-check responsive-check web-check web-test`

### Slice 4: Expenses, Reports, Library, Patterns, and To-Go

- **Owns:** `packages/iroha-shared/src/theme-ui/grapher/Expenses.svelte`, `Reports.svelte`, `Media.svelte`, `MediaDetail.svelte`, `Daily.svelte`, supporting components in `packages/iroha-shared/src/theme-ui/components/`, and shared composition extraction for `apps/iroha-web/src/routes/to-go/+page.svelte` if needed under `packages/`.
- **Changes:** Collapse Expenses' four `Observed period` banners to one line, move empty currency tiles into `FocalBand`'s muted empty line, and deduplicate chart controls/provenance. Adopt `FocalBand` across Reports (`showAllMonths={false}`), Library, Patterns, and To-Go.
- **Moves:** **busy and noisy**, **inconsistent**.
- **Done when:** `make theme-boundary-check responsive-check web-check web-test`

### Slice 5: Full validation and browser checks

- **Owns:** Cross-route verification and visual/accessibility polish in `packages/iroha-shared/` and `apps/iroha-web/src/routes/app.css`.
- **Changes:** Run full pre-PR validation and headless Chromium E2E; verify light/dark contrast, zero horizontal overflow (`320`–`1280px`), focus rings, and `CONSTRAINTS.md` bundle baselines. Run `make web-mobile-check` when a local seeded server is available.
- **Moves:** **busy and noisy**, **inconsistent**, **cramped and small**, **flat and grey**.
- **Done when:** `make validate && make e2e`

---

## 7. Acceptance criteria, revised

- **One focal band per route:** every private route renders exactly one `FocalBand` (`2fr 1fr` at `>= 1024px`).
- **No duplicated figures:** no page states the same figure or scope banner twice.
- **No empty tiles:** no tile renders whose only content is `0`, `—`, or an unavailable note; empty items collapse into one muted line.
- **Distinct type voice:** page title (`h1`) and card title differ in size; the `--type-hero` metric owns the largest step inside `FocalBand`.
- **One table implementation:** no pasted table CSS outside `MetricTable.svelte`.
- **Contrast, targets, focus, overflow:** body/muted contrast `>= 4.5:1`, large/UI `>= 3:1`, focus contrast `>= 5.04:1`; interactive targets `>= 24x24` CSS px (`36px`–`44px` primary controls); `:focus-visible` `2px solid var(--color-focus)` with `3px` offset; zero horizontal overflow at `320`, `375`, `414`, `768`, and `1280` px in light and dark modes.

## 8. What this plan does not do

- **No API or backend changes:** no modifications to `apps/iroha-server/`, `apps/iroha-job/`, DTOs, or pagination/date semantics.
- **No new routes:** no new pages, URL restructures, or navigation IA changes.
- **No domain-hue changes:** `packages/iroha-shared/src/theme/palette.css` sport, ring, sleep, and category hues stay as defined unless a contrast check fails.
- **No `/showcase` content changes:** `apps/iroha-web/src/lib/public/` data, copy, and projection boundaries remain untouched.
- **No second design language:** Grapher remains the single registered language.

## 9. Risks and mitigations

- **Cutting a figure a future feature needs** — mitigated by the per-screen cut table and by moving, not deleting, any figure the owner names.
- **Boundary violations from doing layout work inside `apps/iroha-web/`** — all shared primitives and compositions live in `packages/iroha-shared/`, enforced by `make theme-boundary-check` in every slice.
- **Dense screens fighting the `#eef4f4` tint** — `FocalBand`, collapsed empty lines, and single scope banners remove competing visual noise.
- **Chart colour floors and `--accent` split** — keep `#076660` (`6.13:1` on `#eef4f4`) unified across `app.css`, `themes.css`, and `palette.css`, and register chart-read tokens with `@property`.
- **Regressing loading or scope contracts** — keep `createAsyncResource` / `LoadingBoundary` and `PeriodSelector` wiring intact; verify with `make web-test` and `make e2e`.
