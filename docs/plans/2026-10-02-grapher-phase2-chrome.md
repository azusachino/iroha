# Grapher Phase 2: chrome proposal

Status: **proposed; owner approval required before implementation**. The request authorizes this proposal, not the UI changes below.

## Outcome and provenance

Make the route title, time context, summary and evidence read in one predictable order without changing the quantities or scope behavior underneath them.

[Issue #85, workstream 5](https://github.com/azusachino/iroha/issues/85) names Phase 2 as compact headers, one header time bar per route, a single panel frame, period-aware StatTile summaries and one reading order. Its referenced original plan was not available; the replacement [Phase 1 plan](2026-10-01-grapher-ui-fine-tuning.md) did not approve later phases. This is a new proposal grounded in that issue and current code, not a reconstruction of missing acceptance criteria.

Baseline: main `bafd38ceee464d0f05b2c6b978457723441c7c4f`, after merged #103 and #106; #102 and #105 are closed. Keep the [Phase 1 acceptance](../audits/2026-10-01-grapher-pilot-baseline.md) and [verification follow-up](../audits/2026-10-02-grapher-verification-gaps.md) as bounded evidence, including their limits.

## Recommended scope

Start with the same four pilots: private Overview, Expenses, Metrics and the public landing dashboard. Prove the shared chrome there before proposing a route-by-route rollout. Completing this pilot does **not** mean every route has completed Phase 2.

One cohesive route-chrome capability uses these existing owners:

| Responsibility | Current owner / proposed change |
| --- | --- |
| Route heading and time context | Shared Grapher compositions plus `PeriodToolbar`/`PeriodSelector`; compose one shared header from the existing primitives. |
| Panel appearance | `MetricPanel` currently wraps metric rows, provenance, chart/table view and CSV, not a generic bordered panel. Share presentation without erasing that contract. |
| Summary tiles | `StatTile` exists; migrate Overview's bespoke `.stat-card` composition while preserving links, missing states and per-tile context. |
| Async state, selectors and navigation | App adapters retain ownership; do not move requests or `$app` callbacks into shared visual code. |

### Proposed decisions for owner approval

| ID | Recommendation | Trade-off / alternative |
| --- | --- | --- |
| P2-D1 | Compact **route** headers, not the shell rail. One visible h1, one header/time-context region, no duplicate promotional route heading. Use existing type/spacing roles; keep required labels fully readable. | Removes repeated chrome without a navigation redesign. Proposed populated-fixture budget: header including time controls <=8rem high at 1280px/16px root; narrow/zoomed layouts may wrap and have no fixed height cap. |
| P2-D2 | One route-level period-control group in the header where supported. Overview has a read-only mixed-window context, not a new selector. Keep public year buttons rather than forcing its static projection through the private month selector. | Consistent location does not mean identical scope capabilities or a new global filter. Filters such as currency, sport and city retain their existing meaning. |
| P2-D3 | Extract one small shared **presentation frame**, consumed by `MetricPanel` and non-metric panels. Keep MetricPanel's required metric identity/rows/provenance/CSV contract intact. | A deliberate refinement of #85's “MetricPanel as the single panel frame”: do not fabricate metric rows for a map or ledger. Alternative: make MetricPanel a generic optional-metadata component; not recommended because it weakens the truthfulness contract. Owner must approve this refinement. |
| P2-D4 | Use shared StatTile presentation for pilot quantity summaries, with visible period/window context and an optional existing-action slot where needed. Derive retained-data context from the loaded payload, not the pending selector. | Overview needs recent-sleep and library links and independent missing states; a migration that silently drops them is not acceptable. Do not create another stat-card style. |
| P2-D5 | DOM and visual order agree: route heading/context -> filters/actions -> summaries -> plots -> exact records/detail -> supplemental context. A route may omit absent groups. | Geography counts as a plot, not a new domain. Keep within-route reading order explicit; avoid CSS `order` rearrangements that disagree with Tab/screen-reader order. |

## Preserve the actual scope model

| Pilot | Current behavior to preserve | Proposed header/stat context |
| --- | --- | --- |
| Overview | Activity summary is unfiltered; recent activity list is capped at 5; sleep fetch uses `recent: 30`; media aggregates are separately loaded. There is no routine successful-load refetch affordance. | State that this is a mixed-window overview. Movement/library totals use their actual scope; recent sleep/list/heatmap panels carry their own window/count labels. Do not call 30 recent records “30 days”. No newly reachable year/month/lifetime selector. |
| Expenses | Supported month/year/lifetime queries update actual records/totals; currency/category selectors keep their current meaning. Loaded scope remains visible during a delayed change. | Move the existing controls, not their state or request logic. Header context distinguishes pending selection from observed data; stat/panel captions describe the loaded scope/currency. |
| Metrics | Selector anchors a rolling 12-month window; “All months” is deliberately disabled. Metric identity, dimensions and observed window come from the series response during refetch. | One existing period group; preserve the catalog selector and response-derived metric/window labels. Do not imply an unsupported all-time aggregation. |
| Public landing | Year/sport selection derives summaries, activity rows and monthly data from already-loaded sanitized records. Year/sport changes reset city selection; city filtering affects the existing geography view, not an invented route-wide data scope. | Move the year group into the dashboard header; preserve year/sport behavior and independently label geography context. No new month/lifetime option or private request. Public detail/back navigation stays intact. |

The public landing also renders selected-activity details. This proposal does not impose dashboard controls on detail views or change their back-navigation semantics.

## Ownership and implementation boundary

- Visual header/frame/tile assets belong under `packages/iroha-shared/src/`; search existing components first. Shared code accepts data, snippets and callbacks, never app API clients or route imports.
- Private adapters remain in `apps/iroha-web/src/routes/`; public sanitized-data and page-selection adapters remain in `apps/iroha-public-site/src/routes/`.
- The new frame is presentation only. Keep chart/table toggles, exact CSV rows/filenames, metadata and metric identity in MetricPanel. Remove duplicate outer borders/headers when migrating a panel; do not wrap one visible frame inside another.
- Chart-only changes, new public CSV capabilities and metadata fabrication are not part of extracting a frame. Preserve each pilot panel's existing capabilities.
- Default behavior of shared primitives must remain compatible with other consumers. Inventory those consumers and run the full existing suite even though the visual migration is pilot-only.
- Keep the existing fonts, tokens, chart library, async resources and calendar-scope model. No new dependency or framework. This proposal specifies behavior, not an approved component API signature.

## Proposed acceptance

| ID | Falsifiable pilot criterion | Evidence |
| --- | --- | --- |
| P2-A1 | Each populated pilot dashboard has one visible route h1 and one header/time-context region. The approved 1280px canonical fixture meets the 8rem header target without hiding required labels. Long-title and 320px cases wrap without overlap or clipping. | DOM counts, measured header rectangle, screenshots; owner reviews target before coding. |
| P2-A2 | Exactly one interactive route period group where supported; Overview has zero and truthful read-only context. Existing month/year/lifetime/rolling-window/public-year controls update the same data/URLs as before. No unsupported period option is exposed. | Existing scope fixtures plus control-count, reload/navigation and concrete payload/value assertions. |
| P2-A3 | Pilot panels consume one shared presentation-frame definition; MetricPanel retains metric identity, provenance, coverage, chart/table and exact CSV behavior. No nested duplicate visible frame or invented map/ledger metadata. | Source/import scan, panel DOM/visual review and chart/table/CSV row/filename regressions. |
| P2-A4 | Pilot summaries use StatTile; each quantity has a visible, accessible loaded-period/window context. Unknown distances, zeros, independent loading/failures, recent-sleep semantics and existing action links remain truthful. | Populated/empty/error/deferred fixtures; test header, tile and panel contexts together during refetch. |
| P2-A5 | DOM, visual and Tab order follow D5. Controls, open navigation disclosures, chart/table/CSV actions and retained-data retry remain reachable at 320/768/1280 in both modes and native 200% zoom at 320 CSS pixels. | Keyboard cycles and open-disclosure checks; verify reachable actions, not merely explicit focus+Enter. |
| P2-A6 | Existing text/focus/primary-chart and resolved canvas-text paint contrast floors remain intact. Reduced motion retains loading/refetch feedback; numeric changes do not shift surrounding summaries. | Existing contrast/native-zoom/motion/tabular suites plus new chrome fixtures; same documented glyph/group-opacity and conformance limits. |
| P2-A7 | Equivalent private/public quantities, units and identity colors still agree; public fixtures retain no-private-endpoint/owner-identity checks and the unchanged sanitized projection. | Existing equivalence/palette/privacy suite; no new privacy or field contract. |
| P2-A8 | `make validate`, disposable integration and full `make e2e CI=1` pass; fresh independent acceptance reviews actual journey and shared ownership before readiness. No suppressions, assertion deletions, skipped tests or lowered floors. | Exact-commit gate results and independent criterion verdicts on the owning implementation PR. |

Browser claims remain bounded Chromium synthetic evidence, not a recovered full cockpit audit or blanket screen-reader/color-vision/production conformance.

## Suggested delivery after approval

1. Inventory pilot headers, controls, panels, summaries and other consumers of the shared primitives. Capture the canonical/header-long-title/empty/error baselines and finalize the approved compact-height target.
2. Land one header/time-context slice, using the existing state handlers and actual supported scope inventory. Add a regression before moving controls; verify Expenses first, then the other pilots.
3. Share the presentation frame through MetricPanel and one non-metric pilot panel. Prove exact table/CSV/provenance parity and remove double framing before expanding to other pilot panels.
4. Migrate pilot summaries and reading order; cover mixed windows, action links and stale-versus-pending context. Keep changes small and independently testable.
5. Run full acceptance and independent review; record explicit nonpilot debt. Propose the remaining route rollout separately, not as an implied consequence of pilot approval.

This is an approval-level sequence, not a dispatched implementation task graph. Record approved work in Asobi and its owning issue/PR before implementation.

## Commands and boundaries

Run from `vendor/iroha`, using pinned Make targets:

```sh
# This documentation proposal
make fmt-docs-check
make check
make validate

# Later implementation acceptance, only after owner approval
make test-integration
make e2e CI=1
make e2e-pilot-audit OUT=/absolute/path/to/.tmp/<task>/pilot.json
```

Always preserve source evidence, missing-data semantics, retained-data truthfulness, privacy, shared import direction and CONSTRAINTS.md. Use existing Svelte snippets/data contracts; apps supply state and callbacks. Ask before changing scope behavior, API/schema/privacy policy, dependencies, CI configuration or the agreed pilot/height target. Never deploy, release, mutate live data or claim the original missing audit recovered as part of this proposal.

Out of scope: shell rail/navigation regrouping, chart-type replacement or smoothing changes, coverage/import annotations, synchronized crosshairs, sleep-window/heatmap redesign, transitions/tweening/clock, new domains, Liquid Glass adoption and universal nonpilot rollout. Source freshness already shipped; do not duplicate it here.

## Approval requested

Approve or amend **P2-D1–D5**, **P2-A1–A8**, the four-pilot boundary and the explicit D3 panel-frame refinement. In particular: is pilot-first the right delivery boundary, is the proposed header-height target useful, and may non-metric panels share the frame without impersonating metrics?

Merging this documentation records a proposal. It does not approve these decisions or start Phase 2 implementation.
