# Grapher UI fine-tuning: replacement Phase 1

Status: owner approved P1-D1–D6 and P1-A1–A9 on 2026-10-01 after PR #89 merged. After PR #101 merged, the owner requested one combined PR for all remaining Phase 1 work and acceptance evidence. Use small verified commits internally. Implementation and bounded P1-A1–A9 independent acceptance are delivered in [PR #103](https://github.com/azusachino/iroha/pull/103) at code revision `b471075dc43ca9b5129e281798aaf6164595233d`, pending owner merge; the pilot record preserves measurement limits and explicit nonpilot debt.

Active acceptance issue: [iroha #102](https://github.com/azusachino/iroha/issues/102), follow-up to closed [iroha #85](https://github.com/azusachino/iroha/issues/85). The original plan named there was absent from this checkout. The owner requested a replacement proposal. These criteria and decisions are new; they do not reconstruct or claim approval of the original A1–A9/D1–D6. Later phases remain outside this slice.

## Outcome and scope

Give the private cockpit and public site one shared set of number/unit formatting, Grapher typography/spacing and data-color definitions. Reuse the existing components and CSS; do not introduce a second design system, chart library, font dependency or global refactor.

The owner reports the prior cockpit accessibility audit complete. Its referenced `docs/audits/2026-09-30-v0.6-cockpit-quality.md` is still absent locally. After PR #98 merged, the owner authorized a fresh bounded pilot baseline instead of waiting for that missing handoff. See [the separate pilot record](../audits/2026-10-01-grapher-pilot-baseline.md) for its findings and unverified areas. Capture that baseline before overlapping typography/chart changes; do not overwrite another agent's audit or treat the earlier run's failures as current findings. This authorization does not make the absent full cockpit matrix complete.

## Source baseline

- `packages/iroha-shared/src/format/format.ts` already owns metric formatting and several unit helpers. `apps/iroha-web/src/lib/format.ts` still duplicates duration, distance, pace and other presentation logic. Preserve the web adapter's configured timezone when consolidating helpers.
- Expense color identity lives in `packages/iroha-shared/src/domain/category-color.ts`; health identity lives in `domain/health-metric-colors.ts`. Their CSS variables are partly defined separately in `apps/iroha-web/src/routes/app.css` and `apps/iroha-public-site/src/routes/app.css`. Consolidate definitions rather than adding another map.
- Shared typography/themes live in `packages/iroha-shared/src/theme/{fonts,themes}.css`; shared chart compositions live in `theme-ui/components/`. Existing `components/{MetricPanel,StatTile,MetricTable}.svelte` are the reuse points.
- Keep the registered Grapher identity and shared-package ownership in `AGENTS.md`. Apps own data/timezone adapters, async state and callbacks, not visual primitives or canonical palette/type definitions.

## Approved decisions

| ID | Proposal | Trade-off |
| --- | --- | --- |
| P1-D1 | Use shared `Intl.NumberFormat` helpers with an explicit locale option and deterministic `en-US` default. Group numbers; return `—` for null/non-finite values. Preserve zeros and currency identity. | Stable public/private output; broader localization remains an explicit future choice. |
| P1-D2 | Use human durations (`1 h 02 min`, `45 min`) in summaries, retain clock notation for pace and exact tables, and place a unit once in each axis title rather than on every tick. Tick steps use 1/2/5 multiples selected for the visible range. | Requires distinguishing quantity display from pace/table formatting; cannot blindly change every duration caller. |
| P1-D3 | Keep the existing shared fonts. Define semantic type roles on a rem scale equivalent to 12/14/16/20/24/32px at a 16px root, with shared 4px-based spacing. Migrate only Grapher and shared primitives used by the pilot routes. | A bounded migration leaves unrelated legacy CSS for later work; it must not create competing role definitions. |
| P1-D4 | Centralize domain/sport/category palette definitions under the shared package; identity is stable, not array-order based. Known categories have distinct resolved colors in each mode; unknown categories get a labeled neutral fallback. | Distinct color does not prove color-vision accessibility: keep labels/legends and verify chart mark contrast separately. |
| P1-D5 | Pilot on private Overview, Expenses and Metrics plus the public landing activity summary. Verify light/dark, 320/768/1280px, keyboard and reduced-motion states; preserve all supported year/month/lifetime scope behavior. | Deliberately not a redesign of every route, header, panel or chart type. |
| P1-D6 | Gate with shared formatter/palette/tick unit tests, existing theme-boundary checks, both consumer builds and browser fixtures. Do not lower quality floors or adopt report-only bundle limits as invented blocking budgets. | Real production transfer/interaction baselines still require their separately agreed audit scope. |

## Acceptance criteria

| ID | Falsifiable criterion | Verification |
| --- | --- | --- |
| P1-A1 | Pilot consumers use the same shared quantity helpers. Explicit `en-US` fixtures show grouped `12,345`; zero remains `0`; null/NaN/infinity show `—`. Units/currencies are never silently converted or summed across currencies. | Unit fixtures plus pilot render assertions. |
| P1-A2 | Summary duration 3720s renders `1 h 02 min`; a pace rounding across 59.6s renders `1:00`, never `0:60`. Negative/missing pace does not render a plausible measurement. Existing exact-table clock semantics remain deliberate. | Boundary-value formatter tests. |
| P1-A3 | Chart axes have one unit label, monotonic finite ticks, no duplicate labels, a range containing all finite data, and stable empty/constant/negative/small/large-value behavior. Existing gaps stay gaps. | Pure tick tests and chart browser assertions. |
| P1-A4 | Shared CSS defines one canonical type/spacing role set; pilot typography uses it without app-local copies. At 200% browser zoom and 320px no controls/labels overlap or become unreachable, and tabular figures do not shift surrounding layout when values change. | Source boundary scan and browser matrix. |
| P1-A5 | Every known category and canonical sport resolves to its documented shared identity color; category colors are pairwise distinct within each resolved mode. Reordering data does not change identity colors. Unknown identity is explicitly labeled. | Palette fixtures and computed-style checks for both hosts. |
| P1-A6 | Text passes 4.5:1 (large text 3:1); meaningful chart marks and focus indicators pass 3:1 against their adjacent surfaces. Labels/legends carry distinctions without color alone; no claim of color-vision conformance from distinct hex values. | Contrast computation and keyboard/legend review on pilot fixtures. |
| P1-A7 | Private and public pilot values/units/colors agree for equivalent synthetic data. Public privacy projection is unchanged; no private samples or endpoints enter public fixtures. | Separate consumer fixtures plus privacy regression suite. |
| P1-A8 | Pilot loading/error/empty/refetch states retain truthful missing-data labels and keyboard access. Supported month/year/lifetime changes update data and labels, not just the selector. Reduced-motion keeps state feedback without added loops. | Playwright deferred-response and scope/motion fixtures. |
| P1-A9 | `make validate`, disposable integration and `make e2e` pass without checker suppressions, deleted assertions or lowered floors; independent review checks these criteria and shared import direction. | Recorded exact-commit gate/review evidence. |

## Delivery steps

1. Capture the owner-approved decisions and audit handoff; add failing formatter/tick/palette boundary fixtures before changing behavior.
2. Consolidate shared formatter helpers and pilot adapters; verify quantity versus pace/table callers. Commit after focused tests.
3. Move shared data-color definitions and migrate pilot consumers, preserving identities and adding computed-style checks. Commit after both consumer checks.
4. Introduce the bounded shared type/spacing roles and pilot migration. Browser-verify the matrix and capture any remaining route debt explicitly.
5. Independent review against P1-A1–P1-A9, full gates and handoff. No release, deployment, public field-policy change or production measurement is inferred.

## Out of scope

Next proposal: [Phase 2 route chrome](2026-10-02-grapher-phase2-chrome.md), based on #85's chrome workstream. It is not approved for implementation; its pilot boundary and panel-frame interpretation require owner review.

Later issue phases: header/time-bar redesign, universal panel adoption, chart-type replacement, coverage/import annotations, crosshair synchronization, sleep-window/heatmap redesign, view transitions/tweening/clock, and navigation/copy overhaul. Source freshness shipped separately and must not be duplicated here. Existing font licenses and assets remain unchanged. Approval of Phase 1 does not approve Phases 2–5.
