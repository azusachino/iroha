# Grapher pilot baseline — 2026-10-01

## PR #103: remaining first-load and scope-label fixes

New light/dark regressions reproduced a blank public first load while sanitized requests were held and an Expenses header labeling retained August observations as September during a held scope change. Expenses now stores the requested scope with its successful AsyncResource payload and passes that loaded scope to the shared composition. Public `app.html` supplies a plain live bootstrap status for the client-only route; the layout removes it on mount of either the success or error shell. SvelteKit still owns PageLoad and error recovery; no hand-rolled request/resource lifecycle or public data policy changes were added. Twelve focused private/public state tests pass after the fixes.

Overview has no routine period/refetch action after a successful load: its reload callback is exposed only in the error state. Its initial failure/recovery behavior is covered, but this record does not invent a successful-load refresh affordance or claim an exercised quiet refetch there. Public year/sport selection uses already-loaded projection data rather than refetching it. Quiet network refetch applies to the exercised Expenses/ Metrics selectors.

## PR #103: native-zoom keyboard cycle

The eight native Chromium zoom cases now enumerate reachable controls in the populated closed-disclosure state and traverse a complete Tab cycle at 200% / 320 CSS pixels, checking every expected target was visited, focus is neither inert nor hidden, the focused target is within the viewport, and an outline is present. All eight pass. The first run exposed ArchiveTotals suppressing the shared focus outline; removing its local `outline: none` restores the existing canonical focus treatment. Closed details descendants are explicitly excluded from enumeration because Chromium can report their rectangles despite suppressing tab navigation. This is not a popup/dialog/open-disclosure keyboard matrix or a measured focus/canvas contrast result.

## PR #103: private first-load and public repeated-retry coverage

Six additional light/dark browser cases hold Overview/Expenses first-load responses until the loading status and absent record controls are verified, then return two failures and recover using keyboard activation of the existing Retry overview/Refresh actions. Expenses also retains its prior canonical record and keyboard access through a held month change, then replaces it with the September record and removes the August record. The existing public compact error tests now execute another failed retry before restoring sanitized data; recovery still rejects private requests. These checks passed without application changes.

This does not close P1-A8: public deferred first-load feedback and Overview quiet refetch still need separate verification, as do stale scope-label semantics in Expenses. Keyboard activation here explicitly focuses the recovery control and uses Enter; it is not proof of a complete Tab traversal. No independent final review is claimed.

## PR #103: Metrics async-state increment

Metrics previously removed its chart during every series refetch and rendered failures without a retry action. Four light/dark browser cases reproduced the absent quiet-update status and missing catalog retry before implementation. The route now uses its existing AsyncResource objects through LoadingBoundary, keeping loaded observations keyboard-accessible while a subsequent request is pending. The shared RetryNotice retries the catalog or series that failed. Retained observations use the returned metric definition, dimensions and period, rather than relabeling old data with the pending selector; the observed window is explicit and CSV filenames retain the response's ending month.

Eight blocking light/dark tests cover deferred initial catalog load, deferred rolling-window refetch, two consecutive catalog failures followed by keyboard recovery at 320px, and two consecutive series failures with prior observations retained. The deferred-refetch assertions distinguish the old returned window from the newly selected window and verify the updated response after release. These use synthetic route interception and controlled promise release, not sleeps or a live endpoint.

Verification: focused 8/8; full `CI=1 make e2e` 59/59; `make validate` including both consumer builds; disposable `make test-integration`; report-only pilot 35/35, all pass. Integration ran before the final presentation-only observed-window label adjustment. No new dependency, registry configuration, suppressed checks or shared import-direction changes. This is a Metrics slice, not complete P1-A8 acceptance: Overview/Expenses/public deferred and repeated-recovery matrices, full focus/reachability and independent final review remain outstanding.

Status: bounded baseline, with outstanding findings. Not Phase 1 acceptance or a full cockpit audit.

The owner authorized a fresh pilot check after [PR #98](https://github.com/azusachino/iroha/pull/98) merged. The application baseline is `1bd768185c942525f4f568de435cf13fc6b4046a`. This slice adds fixtures, diagnostics and this record; it changes no application UI. The missing `2026-09-30-v0.6-cockpit-quality.md` remains missing. This document does not replace or reconstruct that agent's audit.

## Reproduce and retain evidence

```sh
make e2e-pilot-audit
# Optional absolute output path; the default is dist/grapher-pilot-audit.json.
make e2e-pilot-audit OUT=/absolute/path/to/report.json
make validate
CI=1 make e2e
```

The pilot target uses the existing Playwright configuration, both local Vite hosts, one worker and no retries. It is report-only and separate from the blocking `chromium` regression project. No CI jobs, dependencies, checker suppressions or thresholds were added. Passing probes mean evidence was collected, not that the recorded problems passed acceptance.

The JSON reporter embeds screenshots and measurement/ARIA attachments as base64 bodies. Retain this report before rerunning: Playwright's ordinary test-results directory is transient. The local run and extracted evidence are under workstation `.tmp/iroha-grapher-pilot-baseline/`; they are not published artifacts. Fixtures contain synthetic records only; public fixtures use public-projection fields and reject private API requests.

## Scope and coverage

| Area | Evidence | Boundary |
| --- | --- | --- |
| Routes/layout | Overview, Expenses, Metrics and public landing, each light/dark at 320/768/1280px: 24 populated captures | No document overflow in these captures. Public table controls outside the initial 320px view belong to its horizontally scrolling table; that is not a page-overflow finding. |
| Accessibility | ARIA snapshots, visible main-landmark count, control rectangles and first 12 Tab stops per populated capture | No sampled focus inside inert/ARIA-hidden content. Full tab cycles, popup/dialog interaction, focus-indicator contrast and screen-reader testing are not verified. Public landing has no main landmark. |
| Colors | Computed text colors composited over solid ancestor backgrounds, WCAG relative-luminance ratios | Gradients/images and non-unit ancestor opacity are excluded. Canvas marks, palette identity across all categories and color-vision accessibility are not verified. |
| Typography | Computed heading sizes/weights/line heights and sample numeric font/variant inventory | Current Grapher hierarchy is recorded, not replaced. No full semantic-role migration or dynamic figure-width test. |
| Writing/UI | Human inspection of the four 320px light screenshots and initial Metrics desktop probe; error/empty ARIA snapshots | Other screenshots were captured but not exhaustively human-reviewed. Brand copy and decorative choices are not treated as defects. |
| Motion | Fresh reduced-motion reload at each populated matrix cell; CSS animation counts and loaded ECharts options | CSS zoom at 200% is only a diagnostic surrogate. Native 200% browser zoom remains unverified. |
| States/scope | Eight empty/error captures; Expenses month → year → another month, Metrics rolling-window change, public 2026 ↔ 2025 year changes | Initial loading, deferred responses, quiet refetch, recovery actions and full lifetime behavior remain unverified. Empty/error states were not repeated across the full mode/width matrix. |

The fixture model includes two distinct activity years (2025: 8km; 2026: 12.5km), separate JPY/USD expenses and one observed Steps point. Routes, sleep and media are empty. Private activity totals/breakdowns/active days describe the same activity fixture; absence of route geometry is deliberate. This does not exercise populated maps or every health/expense dimension.

## Findings

### HIGH — a public year change retains a future-year series

Resolved by the follow-up below; retained here as baseline evidence.

On the public landing, select 2025 after the initial 2026 render. The chart label and activity table switch to 2025, but the loaded `YearProgressChart` option still contains both the 2025 (8,000m) and 2026 (12,500m) series. Repeating the switches reproduces the stale series. Selecting 2025 has no 2024 prior series in this fixture, so 2026 must not remain as its comparison.

Evidence: `chart-lifecycle.json` attachment, inspected through the already-loaded ECharts module's public `getInstanceByDom`/`getOption` APIs. `packages/iroha-shared/src/components/YearProgressChart.svelte` updates with ordinary option merging. Add a blocking series-removal regression before changing its update policy. This is a reproduced data/scope defect; it is **not** a root-cause explanation for the earlier intermittent initialization exception.

### MEDIUM — light-mode text contrast misses 4.5:1

All three measured widths reproduce these pairs:

| Surface/text | Foreground | Adjacent solid background | Ratio |
| --- | --- | --- | --- |
| Public selected year and GitHub footer link | `#0e8f86` | `#f5f7fa` | 3.70:1 |
| Expenses Delete button | `#d64545` | `#f7f9fc` | 4.15:1 |

These are normal-sized text, requiring 4.5:1. Correct the owning semantic tokens rather than assigning arbitrary per-route colors. No equivalent failure was found among the supported dark-mode text pairs; excluded surfaces remain unverified.

### MEDIUM — one observed Metrics point has no visible mark

Resolved by the isolated-point follow-up below; retained here as baseline evidence.

The Steps fixture has a finite value of 12,345 and coverage 1/1. Its line chart shows axes but no point. The loaded option and `packages/iroha-shared/src/theme-ui/components/DailySmallMultiples.svelte` both have `showSymbol: false`; one point cannot draw a connecting line. Exact values remain available through the trend data/table controls, but the visual does not distinguish this observation from an empty plot. Include a one-point case in the chart/tick slice.

### MEDIUM — narrow expense composition tick labels overlap

Resolved for the bounded JPY/USD fixtures by the narrow-axis follow-up below; retained here as baseline evidence.

The 320px light Expenses capture shows full JPY tick labels crowded together on the category composition chart. The record detail remains readable. Apply the approved bounded tick/unit formatting to the shared chart; retain exact currency values in its table and tooltip.

### MEDIUM — public monthly chart does not disable entrance animation for reduced motion

Fresh reduced-motion reloads leave `MonthlyBarChart` with `animation: "auto"` and a 550ms duration; the app component explicitly sets the duration without checking reduced motion. The installed ECharts series implementation treats this truthy animation option as enabled for this small dataset. The cumulative and shared expense/metric charts disable animation in the same captures. Resolve this through shared chart ownership, without adding another public-local visual primitive.

### MEDIUM — public error state has no useful recovery action

A synthetic 503 response renders the default “500 / Internal Error” page and footer, without the activity shell or a retry control. It does not pretend the fetch succeeded, but does not satisfy the planned pilot error/recovery treatment. Loading/error orchestration belongs in the host; reuse existing shared presentation assets for the view.

### LOW — public structure and target candidates need follow-up

The populated public route lacks a main landmark. Table sort controls measure about 20.4px high. Review their surrounding spacing against the 24px target-size exception before calling that a WCAG failure; the inline footer link has its own exception. Do not fix this by disabling sorting or removing controls.

## Verification and next slice

- `make e2e-pilot-audit`: 35/35 probes completed without retries; no captured page exceptions or unknown fixture API requests.
- `make validate`: passed with existing quality floors.
- `CI=1 make e2e`: 12/12 blocking browser regressions passed.
- No backend changes, database mutation, deployment or release. Disposable PostGIS integration was not rerun for this audit-only slice.

Keep issue #85 and Grapher Phase 1 open. The stale-series follow-up below addresses the first chart-correctness finding; proceed next with the remaining findings and approved tick/palette/type slices. Retain page-error assertions and stack diagnostics: the earlier CI `__ec_inner_*` initialization exception has not reproduced here and is still unresolved. Complete deferred-state, native-zoom, chart-mark/focus contrast and equivalent-scope checks before claiming P1-A1–A9 acceptance.

## Follow-up: stale year-series removal

The baseline was committed separately as `1576fa3`. The blocking `chart-scope.spec.ts` regression first failed in both modes: choosing 2025 left a 2026 series with 12,500m in the plot. It observes the rendered chart through the same loaded ECharts public API, not merely the already-correct label/table.

`YearProgressChart` now replaces its complete option with `notMerge: true`, matching the existing shared `MediaBarChart` pattern. This component rebuilds the whole scope option; retaining prior models also retained removed years and legend-selection state. No new chart, visual primitive or initialization/disposal workaround was introduced. The animation settings and missing-value data are unchanged.

Verification: focused regressions 2/2; `make validate` passed; `CI=1 make e2e` 14/14; pilot probes 35/35, retained separately in `fixed-report.json`. The original `report.json` remains baseline evidence. Other findings and the earlier initialization exception remain unresolved; this is not Phase 1 completion.

## Follow-up: isolated metric observations

After PR #99 merged, the shared `DailySmallMultiples` chart enables native markers when a series contains a finite observation without a finite neighbor. Fully connected series keep their unmarked appearance and native hover feedback. Mixed sparse series mark their finite observations; nulls remain gaps and observed zero remains a measurement. `showAllSymbol` prevents category-label thinning from hiding the isolated marker, while `connectNulls: false` is explicit. No chart-type replacement or palette identity change is involved.

A zero-sized-symbol approach was not retained: ECharts' default hover scaler divides by symbol size. The native per-series visibility flag avoids that invalid emphasis geometry and keeps its normal symbol sizing.

Screenshot review also exposed unresolved CSS `var(--accent)` paint passed directly to the canvas renderer. The existing `BarChart` simple-token resolver was moved unchanged into shared `chart-color.ts` and reused for small-multiple marks and axis names. Computed colors now reach the renderer; no per-route color map was added. This resolver supports the same simple-token/literal cases as before, not a new general CSS parser.

Blocking browser tests cover one point, observed zero, separated points, mixed sparse/connected points and fully connected points in both modes, using ECharts' public `getOption`/`getVisual` APIs plus exact-table assertions. Marker sizes and resolved paint are checked separately: a symbol-size flag alone did not catch the faint-marker problem. Focused tests failed before each behavior correction; ten focused cases and all 24 blocking browser tests now pass. `make validate` passes, including color-helper unit tests and both consumer builds. Full palette/contrast/state/zoom acceptance and the original initialization exception remain outside this slice.

## Follow-up: narrow expense value-axis labels

After PR #100 merged, shared `BarChart` value axes enable ECharts' native `axisLabel.hideOverlap`. Horizontal charts also reserve the existing 32px bottom gutter already used by vertical charts; the previous 12px gutter clipped the value labels. The engine chooses which labels to show; the formatter, values, currency exponents, tooltip and exact tables are unchanged. No compact-number formatter, currency conversion or custom tick algorithm was added.

Eight blocking browser cases cover JPY/USD in light/dark at 320/1280px. Before the fix they fail on the missing overlap policy and 12px gutter; after it they pass and retain exact `¥12,345`/`$1,234.56` table cells. Automated checks inspect the actual chart's public option contract, not canvas text geometry. Human review of all four 320px composition captures confirms separated, unclipped labels and readable exact bar-end values. This is bounded visual evidence, not a complete 1/2/5 tick/precision/range/negative/extreme-value matrix or P1-D2 unit-once implementation.

`make validate` and both consumer builds pass, all 32 blocking browser tests pass, and all 35 report-only pilot probes complete. Remaining contrast/palette/type/state/native-zoom work, live-theme debt and the original initialization exception stay open.

## Combined Phase 1 branch: palette increment

After PR #101 merged, the owner requested one PR for all remaining Phase 1 work. The palette increment is an internal commit, not a new slice PR or full Phase 1 acceptance.

Both hosts import `packages/iroha-shared/src/theme/palette.css`. Domain mark/sport/ring colors have one shared owner instead of app-local declarations. Light mode uses darker same-family marks; sport `other` is a neutral fallback with existing explicit labels. Expense groceries, shopping, utilities, health and subscriptions use dedicated identity tokens so food/housing and shopping/health no longer collide. The existing food/transport token contracts remain. Unknown categories use labeled neutral chart/table data rather than rank-based colors; health data no longer borrows a host's interactive accent. Data/currency/timezone and public projections are unchanged.

The new browser fixtures compare all 11 category and six canonical sport colors on both hosts in light/dark, check pairwise distinctness within each domain, and check explicit preference overrides the OS. They measure 204 opaque rendered CSS-token text pairs across canvas/surface/raised-surface backgrounds at >=4.5:1; these are controlled color probes, not canvas mark geometry, gradients, hover/focus or a color-vision audit. The original light Delete failure was reproduced at 4.151:1 before changing the destructive text token. Actual Delete and public selected-year/footer text now pass their measured thresholds.

Unit fixtures retain previous assertions and add reorder stability, known identity uniqueness and neutral unknown-category/health fallbacks. Full canvas/focus/legend, equivalent-data scope, native zoom and deferred-state evidence remains pending. No independent review has been performed on this increment.

## Combined Phase 1 branch: public motion/error increment

Four light/dark regressions first failed on monthly `animation: "auto"`/550ms and the default error page without a main landmark or recovery action. The monthly visual component is now shared under `packages/iroha-shared/src/components/MonthlyBarChart.svelte`; its public host consumes it directly, with no fork. Reduced motion explicitly disables animation and uses zero duration. Live motion preference, OS contrast-mode and explicit theme changes redraw it; observer/listener cleanup remains tied to disposal. The native public option/paint assertions cover reduce → normal → reduce and both explicit contrast-mode flips. This does not fix the other charts' existing live-theme tracking debt.

The public host keeps SvelteKit's load/error lifecycle. Failed anonymous responses retain their HTTP status using `error(status, safeMessage)`, without rendering response bodies or internal endpoint details. A custom error adapter uses the existing shared `RetryNotice` and `invalidateAll()` to retry the sanitized projection. Light/dark 320px fixtures prove the retry is keyboard reachable and that restored responses replace the alert with actual activity content. The layout supplies a main landmark for both success and error. All requests remain anonymous `/public/v1/`; private API requests are rejected. Framework references: [SvelteKit errors](https://svelte.dev/docs/kit/errors), [invalidateAll](https://svelte.dev/docs/kit/$app-navigation#invalidateAll).

`make public-site-fmt` wraps the existing formatter script; no new dependency or CI job. `make validate` including both consumer builds passes, all 38 blocking browser tests pass, and 35 report-only probes complete. Initial loading/deferred responses, repeated failed retries/quiet refetch, full keyboard/focus/canvas/native-zoom/equivalent-scope evidence and independent review remain pending. No whole-Phase-1 completion is inferred.

## Combined Phase 1 PR #103: axis and presentation increment

Shared `format/axis.ts` selects zero-referenced finite 1/2/5 intervals with precision derived from the interval. Pure fixtures cover empty/non-finite, zero/constant, negative, small/large, gaps, integer counts, explicit currency-exponent presentation and finite extreme bounds. Axis-only scientific notation handles very small/large labels. Extreme bound unit fixtures do not constitute a complete native canvas geometry matrix.

The pilot expense, Overview, Metrics and public monthly/cumulative plots consume the shared axis policy. Expense axes name the currency once; ticks present major units using the declared exponent while plot data, exact table and tooltip formatters remain unchanged. Distance axes name km once; count axes avoid fractional count steps. Eight existing narrow expense regressions first fail on absent unit names, then assert native bounds and distinct numeric labels alongside their original exact-money, overlap and gutter assertions.

Shared rem type and 4px spacing roles now have one definition in `theme/themes.css`. Shared stat/meta/table primitives, expense/metric titles and chart fonts consume those roles. Chart captions resolve rem to CSS pixels, and narrow small multiples use adequate axis gutters and one column with per-row height. The rest of the bounded typography migration and actual native zoom evidence remain in progress.

A shared chart-presentation observer now redraws all four pilot chart families on live motion, contrast-mode and root-style changes, with disposal cleanup. Four browser cases prove animation switches, changed resolved paint with unchanged numeric values/gaps, and a 20px base font producing 15px chart captions. This is base-font evidence, **not native 200% browser zoom**.

Current `make validate` including both builds passes, all 42 blocking browser tests pass, and 35 report-only pilot probes complete. Full deferred/refetch/repeated-error/keyboard/focus/canvas/equivalent-scope/native-zoom acceptance, final disposable integration and independent review remain open; the combined PR remains in progress.

## Combined Phase 1 PR #103: native browser zoom increment

The shared YearProgressChart's min-content width prevented its public-site grid from reflowing at a 320 CSS-pixel viewport; `min-width: 0` on the shared chart surface fixes that constraint. Both shared chart cards now use the canonical spacing/type roles, and both chart axis titles receive the resolved type-role font size.

A blocking Chromium fixture creates an isolated temporary tabs-only extension and uses `chrome.tabs.setZoom(tabId, 2)`/`getZoom` (not CSS zoom or CDP pinch scaling). Across Overview, Expenses, Metrics and public landing in light/dark, it confirms the 640-to-320 CSS-pixel viewport reflow, doubled device-pixel ratio, unchanged CSS zoom, no document-level horizontal overflow, no page errors, and no unexpected fixture requests. Public light/dark cases specifically cover the chart min-content regression. The eight viewport screenshots were captured through CDP without resizing the `viewport: null` browser and manually inspected for responsive reflow; the old 320-pixel-wide full-page captures are stale and not evidence.

The assertions do not establish that every label/control is unclipped (private shell uses `overflow-x: clip`), that focus remains visible, or that every control is reachable through a full keyboard cycle. These remain open acceptance work. `make validate`, the full blocking browser suite (50/50), disposable `make test-integration`, and the report-only pilot (35/35 probes) pass on this increment. The pilot is evidence collection, not conformance. The original initialization exception, equivalent-scope value/privacy matrix, deferred/refetch/repeated-error states, full keyboard/focus/canvas-contrast review and independent acceptance review remain outstanding; Phase 1 is not complete.

## Combined Phase 1 PR #103: remaining pilot typography roles

The Overview composition, public landing and shared chart-data disclosure/table styles now consume the existing Grapher type roles instead of local fractional rem sizes. Dense Overview record-table text uses the 12px caption role; promoting it to the 14px label role made its intentionally horizontally scrollable table report clipped activity buttons at 320px. Returning that dense table to the caption role restored the existing compact layout without changing the horizontal-scroll affordance. The dynamic heading sizes remain clamped between shared role tokens rather than adding new roles.

The four existing chart-presentation browser cases now compare rendered font sizes against the actual resolved shared role tokens for each pilot host; all 4 pass. The report-only light/dark × 320/768/1280 pilot completes 35/35 with no document overflow. After the dense-table adjustment, the Overview 320px fixture reports no clipped controls; the public activity table still uses its documented horizontal scroll at 320px. Screenshots from Overview desktop and public 320/1280 were manually reviewed. `make validate` and the full blocking browser suite (50/50) pass. This is a bounded type-role migration, not a complete review of every pilot control or spacing declaration; deferred states, full native-zoom keyboard reachability, focus/canvas contrast, equivalent-scope privacy evidence, disposable integration on this latest CSS-only revision and independent acceptance remain outstanding.

## Combined Phase 1 PR #103: public unknown-distance handling

The public projection contract makes activity `distance_m` optional and carries `distance_known_count`/`distance_unknown_count` in its aggregate buckets. The public landing previously converted a missing individual distance to zero in year totals, left cumulative buckets at zero, and selected monthly distance based only on the sum. A synthetic public fixture with one 2026 activity whose distance is omitted reproduced this behavior before the fix.

The landing now counts known and unknown distances separately. When every scoped distance is unknown, the stat tile shows `—` and an unavailable-count label; when only some are unknown, it labels the displayed subtotal as known-only. The monthly plot switches to activity counts if its selected year has any unknown distances rather than presenting a partial distance sum as complete. Shared YearProgressChart stops the cumulative line at the first unknown bucket, reports the unknown boundary, and labels affected data-table cells `Distance unknown`; it does not resume a supposedly exact cumulative sum after the gap.

The blocking public browser regression verifies the unavailable label, count-chart selection, cumulative null at the unknown month, explicit table labels, and no private API requests. Existing light/dark scope-series regressions still pass. `make validate`, full `CI=1 make e2e` (51/51), disposable `make test-integration`, and report-only pilot (35/35) pass. The pilot remains bounded collection, not conformance. Public privacy policy and wire fields are unchanged. Mixed known/unknown copy is implemented but the current new fixture specifically exercises an all-unknown selected year; keyboard/focus contrast, broader scope equivalence and independent review remain open.
