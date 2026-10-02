# Post-merge Grapher verification gaps — 2026-10-02

## Scope and acceptance

Owner approved the follow-up after PR #103 merged as `3729c90d3e41d8d19551cb1845c9842a160ba044`. Issue #102 was closed with its independently reviewed bounded acceptance evidence. This follow-up strengthens tests before a separately proposed Phase 2; it does not redesign or deploy either consumer.

Acceptance:

1. Measure the already-loaded canvas renderer's resolved axis-text paint against the existing chart surfaces in all 24 pilot × mode × width cases. Require 4.5:1, fail closed on unsupported paint, and prove the detector rejects a low-contrast renderer-only change without trusting option defaults.
2. Compare equivalent human-summary duration and pace forms between private and public synthetic consumers, retaining the existing exact-duration and distance assertions and the public privacy/no-private-endpoint checks.
3. Pass owning `make validate` and the full CI browser suite; obtain fresh independent verification before readiness. No runtime implementation, policy, dependency, quality-floor, release or deployment changes.

## Method

`rendered-contrast.ts` now inspects `chart.getZr().storage.getDisplayList()` from the existing loaded ECharts instance and reads the renderer's resolved `tspan` foreground fills. It uses the same conservative solid/constant-hue-gradient surface compositing as the existing DOM/mark checks. Browser assertions require each named axis title to occur in the measured text and require all measured canvas text to meet 4.5:1. The pilot labels are outside chart marks; this surface model does not infer arbitrary canvas overpainting or overlapping-series backgrounds.

A public/light/320 negative control makes one actual rendered axis title fully transparent while its ECharts option remains unchanged. Its measured contrast must become 1:1. The original fill is restored and repainted in `finally`; baseline assertions are not removed or weakened. This tests the detector's sensitivity to renderer paint, not just a configured CSS token.

The scoped `single-year` synthetic fixture uses the same 2026 record and bucket on both hosts. Both summaries must render `1 h 02 min`; the private Overview retains `1:02:00` in its exact table. Opening that record's existing private Motion detail verifies `1:00 /km`, matching the public table's pace. Detail/route/lap/sampling responses are synthetic, explicitly routed only on the private page. The public fixture still rejects private APIs and must not show the owner identity.

## Evidence and limits

Initial focused matrix: 26/26 passed (24 contrast cells and two equivalence cases). The renderer-only negative control separately passed. Complete gates and independent results will be recorded on the owning PR.

These measurements close the resolved axis-text foreground-paint gap, not canvas glyph pixel geometry, antialiasing, clipping/overpainting conformance or color-vision accessibility. The fixtures deliberately preserve independent input quantities; they are not a physical consistency check between pace, distance and elapsed time. No production samples, transfer measurements, screen-reader or Firefox/Safari conformance is inferred. Unmounted MonthNavigator debt and the earlier unroot-caused initialization exception remain outside this follow-up.

The prior bounded Phase 1 report remains [historical evidence](2026-10-01-grapher-pilot-baseline.md). Phase 2 still requires its own scope and acceptance decision.
