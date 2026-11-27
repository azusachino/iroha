# Cockpit UX overhaul: tinted canvas, calm hierarchy, and one rhythm

Date: 2026-11-27. Status: Accepted design direction & implementation plan (issue [#236](https://github.com/azusachino/iroha/issues/236))
Target workspace: `packages/iroha-shared` & `apps/iroha-web`
Reference baseline: `79a014d78681548426995fa471807587fc10474c`

---

## 1. Outcome and constraints

The [2026-10-08 visual overhaul](2026-10-08-cockpit-visual-overhaul.md) landed obsidian dark surfaces (`#0b0f19` / `#121826`), restored Grapher's teal/rose accents, introduced `--radius: 14px`, and added top-rim styling to `StatTile` and `PanelFrame`. What it left undone is the day-to-day UX feel across the private cockpit: light mode stayed on a muted beige-grey (`#f7f6f2` / `#efede5`), typography remained undersized (with hardcoded `0.62rem`–`0.78rem` labels in `Shell.svelte` and no focal metric tier), every `StatTile` carries the same radial glow and accent bar regardless of importance, and spacing/radii drift from route to route.

This plan fixes four symptoms, in priority order:

1. **Flat and grey** — lift light mode off dull neutral surfaces onto a tinted canvas with crisp floating white cards.
2. **Cramped and small** — raise the metric and label floor, add a hero focal step, and give cards and sections breathing room.
3. **Busy and noisy** — remove decorative per-tile glows and establish one focal metric per screen so secondary data recedes.
4. **Inconsistent** — replace ad-hoc margins, radii, and control heights with one shared scale across every private route.

### Hard rules

- **Theme asset boundary** (`AGENTS.md`, `docs/frontend-theme-architecture.md`, `make theme-boundary-check`): all shared primitives, tokens, and Grapher compositions live under `packages/iroha-shared/`. Do not create `apps/iroha-web/src/lib/themes/` or import `$lib/api` or route modules from `packages/`.
- **Loading state** (`AGENTS.md`, `apps/iroha-web/src/lib/asyncResource.svelte.ts`): `LoadingBoundary` takes `resource`/`resource[]`; never re-derive `ready` from `!loading`.
- **Month/year/lifetime scope** (`AGENTS.md`, `apps/iroha-server/pkg/httpapi/read_scope.go`, `packages/iroha-shared/src/format/scope.ts`): every rendered `PeriodSelector` option must work end-to-end; unsupported levels stay explicitly hidden (`showAllYears={false}` / `showAllMonths={false}`, as on Reports).
- **Motion and responsive contracts** (`packages/iroha-shared/src/theme/themes.css`, `docs/frontend-design-contract.md`): use `--motion-*` tokens (`make motion-tokens-check`), canonical breakpoints (`make responsive-check`), and `prefers-reduced-motion` semantics.
- **Quality floor and truthfulness** (`CONSTRAINTS.md`, `make quality-floor-check`): no weakened assertions, suppressions, or skipped tests; preserve API DTOs, IDs, UTC day semantics, and truthful empty/partial/error copy.

---

## 2. The accepted direction

In issue [#236](https://github.com/azusachino/iroha/issues/236), the owner chose the **tinted canvas** light direction out of four rendered candidates. These are the accepted light-mode tokens (not to be re-opened):

```css
--bg: #eef4f4;
--surface: #ffffff;
--surface-2: #f4f8f8;
--border: #d7e3e2;
--text: #12201f;
--text-muted: #4b5f5e;
--accent: #0f766e;
--accent-2: #be123c;
--tile-shadow: 0 2px 4px rgb(10 40 38 / 0.05), 0 16px 40px rgb(10 40 38 / 0.1);
```

Tinting `--bg` cool sea-glass lets pure-white `#ffffff` cards lift cleanly off the canvas through contrast and soft depth rather than heavy borders or per-card gradients.

Dark mode keeps its obsidian family (`#0b0f19` canvas, `#121826` surface, `#1b2438` raised) and inherits the new type scale, spacing rhythm, and depth model without changing hue family. Domain hues in `packages/iroha-shared/src/theme/palette.css` are not re-litigated; only a verified contrast failure may adjust one.

**Primary design risk:** a tinted canvas is harder to keep calm on dense screens where charts, tables, and multi-column stats compete with the backdrop. This plan contains that risk by enforcing **one spacing rhythm** and **one focal metric per screen**, while stripping decorative radial glows off secondary tiles.

---

## 3. Scales

Extend the existing custom properties in `packages/iroha-shared/src/theme/themes.css` and `apps/iroha-web/src/routes/app.css`. No route-local literals outside these tokens.

### Type scale

| Token | Value | Owns |
| --- | --- | --- |
| `--type-caption` | `0.75rem` | Metadata, table headers, provenance footnotes, chart ticks (hard floor; retire `<0.75rem` literals in `Shell.svelte`) |
| `--type-label` | `0.875rem` | Tile labels, nav links, controls, pills, secondary copy |
| `--type-body` | `1rem` | Body copy, table cells, list items, descriptions |
| `--type-title-small` | `1.25rem` | Panel and section headings (`PanelFrame`, `MetricPanel`) |
| `--type-title` | `1.625rem` | Standard `StatTile` numeric values (`tabular-nums`) |
| `--type-display` | `2.125rem` | Page titles (`RouteHeader`, `--grapher-utility-title-size`) |
| `--type-hero` | `clamp(2rem, 4vw, 2.75rem)` | Single focal metric per screen (new) |

### Spacing scale

Keep `--space-1` (`0.25rem`), `--space-2` (`0.5rem`), `--space-3` (`0.75rem`), `--space-4` (`1rem`), `--space-6` (`1.5rem`), `--space-8` (`2rem`), and add:

- `--space-5: 1.25rem` — default card and panel padding.
- `--space-10: 2.5rem` — top-of-page and major section breathing room.
- Semantic rhythm aliases: `--gutter-page: var(--space-8)` (`var(--space-6)` at `<=1024px`, `var(--space-4)` at `<=640px`), `--gap-section: var(--space-6)`, `--gap-grid: var(--space-4)`, `--gap-dense: var(--space-2)`, `--pad-card: var(--space-5)`.

### Radius and control-target scales

- Radii (replacing `calc(var(--radius) - 3px)` math): `--radius-sm: 8px` (compact toggles, badges), `--radius-md: 10px` (nav items, inputs, selects), `--radius: 14px` (standard `StatTile`, `PanelFrame`), `--radius-lg: 18px` (focal hero panels, dialogs), `--radius-pill: 999px` (segmented controls, filter chips, mobile nav pills).
- Targets: `--control-target-min: 1.5rem` (24px WCAG 2.5.8 floor for inline links/summaries), `--control-height: 2.25rem` (36px standard buttons, selects, segmented pills), `--control-height-lg: 2.75rem` (44px nav items and command trigger).

### Depth model

- `--tile-surface`: solid `var(--surface)` (`#ffffff`) in light mode so cards stay crisp on `#eef4f4`; `linear-gradient(180deg, rgb(255 255 255 / 0.03), rgb(255 255 255 / 0)) var(--surface)` in dark mode.
- `--tile-shadow`: light `0 2px 4px rgb(10 40 38 / 0.05), 0 16px 40px rgb(10 40 38 / 0.1)`; dark `0 4px 20px -2px rgb(0 0 0 / 0.45)`.
- `--tile-rim` (new): light `inset 0 1px 0 rgb(255 255 255 / 0.85)`; dark `inset 0 1px 0 rgb(255 255 255 / 0.07)`.
- `--tile-shadow-hover` (new): subtle elevation lift for interactive cards only, without colored neon halos.
- Default `StatTile` removes the unconditional radial gradient and top `2px` accent bar; only a focal/featured tile (`featured={true}`) carries the accent rim and `--type-hero` value.

---

## 4. Implementation slices

Execute in order. Every slice stays inside `packages/iroha-shared/` and `apps/iroha-web/src/routes/app.css`, and must pass its done-when command before moving on.

### Slice 1: Token layer

- **Owns**: `apps/iroha-web/src/routes/app.css`, `packages/iroha-shared/src/theme/themes.css`, `packages/iroha-shared/src/theme/palette.css` (only if a contrast check requires adjusting a light shade).
- **Changes**: Apply the accepted tinted-canvas light tokens (`#eef4f4` canvas, `#ffffff` surface, `#f4f8f8` surface-2, `#d7e3e2` border, `#12201f`/`#4b5f5e` text, `#076660`/`#be123c` accents, new `--tile-shadow`). Reconcile light `--accent` to `#076660` across `app.css`, `themes.css`, and `palette.css`, because `#076660` clears the `5.04:1` focus floor at `6.13:1` on `#eef4f4`, where `#0f766e` measures `4.92:1`. Add `--type-hero`, `--space-5`, `--space-10`, layout rhythm aliases, `--radius-*`, `--control-height*`, `--tile-rim`, and `--tile-shadow-hover`. Register any new color token read by charts via `@property` (`syntax: "<color>"`).
- **Moves**: **flat and grey**, **inconsistent**.
- **Done when**: `make theme-boundary-check motion-tokens-check responsive-check web-fmt-check web-check`

### Slice 2: Shell and shared primitives

- **Owns**: `packages/iroha-shared/src/theme-ui/grapher/Shell.svelte`, `packages/iroha-shared/src/components/StatTile.svelte`, `packages/iroha-shared/src/components/PanelFrame.svelte`, `packages/iroha-shared/src/components/MetricPanel.svelte`, `packages/iroha-shared/src/components/RouteHeader.svelte`, `packages/iroha-shared/src/components/PeriodSelector.svelte`, `packages/iroha-shared/src/components/PeriodToolbar.svelte`.
- **Changes**: Upgrade `Shell.svelte` to use `--type-caption`/`--type-label`, `--radius-md`/`--radius-pill`, `--control-height-lg`, and `--gutter-page` so the rail sits cleanly against `#eef4f4`. Add a `featured` focal variant to `StatTile.svelte` (`--type-hero`, accent rim) and calm default tiles (no radial wash). Apply `--pad-card`, `--tile-rim`, `--radius-*`, and `--control-height` to `PanelFrame`, `MetricPanel`, `RouteHeader`, and period controls.
- **Moves**: **cramped and small**, **busy and noisy**, **inconsistent**.
- **Done when**: `make theme-boundary-check responsive-check web-fmt-check web-check web-test`

### Slice 3: Today and Overview

- **Owns**: `packages/iroha-shared/src/theme-ui/grapher/Today.svelte`, `packages/iroha-shared/src/theme-ui/grapher/Dashboard.svelte`, `packages/iroha-shared/src/theme-ui/components/ActivityHeatmap.svelte`, `packages/iroha-shared/src/theme-ui/components/RingGauge.svelte`.
- **Changes**: Give Today (`/`) one focal daily signal card and calm subordinate ring/session cards on `--gap-section`/`--gap-grid`. Give Overview (`/overview`) one focal period distance tile above subordinate domain stats, unify the heatmap ramp and monthly chart framing on `#ffffff`, and align recent-activity table padding with `--pad-card`.
- **Moves**: **flat and grey**, **cramped and small**, **busy and noisy**.
- **Done when**: `make theme-boundary-check responsive-check web-check web-test`

### Slice 4: Motion and Night

- **Owns**: `packages/iroha-shared/src/theme-ui/grapher/Activities.svelte`, `packages/iroha-shared/src/theme-ui/grapher/ActivityDetail.svelte`, `packages/iroha-shared/src/theme-ui/grapher/Sleep.svelte`, `packages/iroha-shared/src/theme-ui/components/SleepTimelineChart.svelte`, `packages/iroha-shared/src/theme-ui/components/SleepAggregateChart.svelte`, `packages/iroha-shared/src/theme-ui/components/SleepScopeSummary.svelte`.
- **Changes**: Replace dense summary rows on `/motion` with one focal period total plus calm subordinate tiles and consistent pill filters; elevate the headline activity metric on `/motion/:id`. On `/night`, lead with one focal sleep-duration readout above the circadian timeline and stage architecture chart, verifying stage legibility on `#ffffff` and dark `#121826`.
- **Moves**: **cramped and small**, **busy and noisy**, **inconsistent**.
- **Done when**: `make theme-boundary-check responsive-check web-check web-test`

### Slice 5: Expenses, Media and Reports

- **Owns**: `packages/iroha-shared/src/theme-ui/grapher/Expenses.svelte`, `packages/iroha-shared/src/theme-ui/grapher/Media.svelte`, `packages/iroha-shared/src/theme-ui/grapher/MediaDetail.svelte`, `packages/iroha-shared/src/theme-ui/grapher/Reports.svelte`, `packages/iroha-shared/src/theme-ui/grapher/Daily.svelte`, `packages/iroha-shared/src/theme-ui/components/ExpenseLedger.svelte`, `packages/iroha-shared/src/theme-ui/components/MediaAssetCard.svelte`, `packages/iroha-shared/src/theme-ui/components/ReportFactGrid.svelte`, `packages/iroha-shared/src/theme-ui/components/ReportMetricCard.svelte`.
- **Changes**: Apply the focal-metric hierarchy, `--pad-card` framing, and `--gap-section` rhythm across Expenses, Media/Library, Reports, and Patterns (`Daily.svelte`). Keep scope rules intact: Expenses year/month/lifetime works end-to-end; Reports remains month-only (`showAllMonths={false}`).
- **Moves**: **busy and noisy**, **inconsistent**.
- **Done when**: `make theme-boundary-check responsive-check web-check web-test`

### Slice 6: Full validation and browser suite

- **Owns**: Cross-route visual polish and verification only (limited to `packages/iroha-shared/` and `apps/iroha-web/src/routes/app.css`).
- **Changes**: Run the full pre-PR validation gate and headless Chromium E2E suite; resolve any contrast, focus, or overflow regression across light and dark modes at 320–1280px, and confirm emitted JS/CSS sizes remain within `CONSTRAINTS.md` baselines. Run `make web-mobile-check` when a local seeded server is available.
- **Moves**: **flat and grey**, **cramped and small**, **busy and noisy**, **inconsistent**.
- **Done when**: `make validate && make e2e`

---

## 5. Acceptance criteria

Tracked against issue [#236](https://github.com/azusachino/iroha/issues/236):

- **Contrast in both themes**: body and muted text meet WCAG AA (`>= 4.5:1`), large display/hero metrics and UI boundaries meet `>= 3:1`, and computed focus-token contrast stays `>= 5.04:1` (`CONSTRAINTS.md`).
- **Zero horizontal overflow**: no accidental page-level horizontal scroll at 320, 375, 414, 768, and 1280 CSS px in light or dark mode.
- **Target sizes**: every interactive control meets `>= 24x24` CSS px (`--control-target-min`), with primary buttons, selects, and pills at `36px`–`44px`.
- **Visible focus ring**: every actionable control shows `2px solid var(--color-focus)` with `3px` offset on `:focus-visible`.
- **One focal metric per screen**: each private route renders one `--type-hero` focal metric; secondary tiles use calm surfaces with no decorative radial glow.
- **Consistent rhythm**: all private pages share `--gutter-page`, `--gap-section`, `--gap-grid`, `--pad-card`, and `--radius-*` with no `<0.75rem` labels or route-local one-off radii.
- **Truthful states and scopes**: `LoadingBoundary` (`AsyncResource`), empty/partial/error copy, and month/year/lifetime scopes work unchanged.

---

## 6. What this plan does not do

- **No API or backend changes**: no modifications to `apps/iroha-server/`, `apps/iroha-job/`, DTOs, or pagination/date semantics.
- **No new routes**: no new pages, URL restructures, or navigation IA changes.
- **No domain-hue re-litigation**: `packages/iroha-shared/src/theme/palette.css` sport, ring, sleep, and category hues stay as defined unless a contrast check fails.
- **No public `/showcase` content changes**: `apps/iroha-web/src/lib/public/` data, copy, and projection boundaries remain untouched.
- **No second design language**: Grapher remains the single registered language; retired languages and ambient WebGL backgrounds are not revived.

---

## 7. Risks and mitigations

- **Dense screens fighting the `#eef4f4` tint** — enforce one focal metric per screen, `--gap-section`/`--pad-card` spacing, and clean `#ffffff` surfaces without per-tile radial glows.
- **Theme-boundary violations from fixing a route in `apps/iroha-web/`** — implement all visual changes in `packages/iroha-shared/` and run `make theme-boundary-check` in every slice.
- **Chart colour floors in light mode** — charts read CSS variables via `getComputedStyle`, so register any new color token with `@property` (`syntax: "<color>"`) in `palette.css` to ensure concrete resolution, and verify light series contrast on `#ffffff`.
- **Split light-mode `--accent` definitions (`#076660` vs `#0f766e`)** — unify `app.css`, `themes.css`, and `palette.css` on `#076660` (`6.13:1` on `#eef4f4`, clearing the `5.04:1` focus floor where `#0f766e` measures `4.92:1`) in Slice 1 so focus rings, active nav, and charts stay consistent. A future slice must not "correct" it back to `#0f766e` without changing the floor first.
- **Regressing loading or period-scope contracts during layout edits** — keep `createAsyncResource` / `LoadingBoundary` and `PeriodSelector` wiring untouched and verify with `make web-test` and `make e2e`.
