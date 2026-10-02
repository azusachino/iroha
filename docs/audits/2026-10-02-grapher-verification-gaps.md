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

Initial focused matrix: 26/26 passed (24 contrast cells and two equivalence cases). The renderer-only negative control separately passed. At `4d5173edc115bb54161a44401dd08b71d621948d`, lead docs/validation and full browser 102/102 passed. Fresh task-created Herdr verifier `iroha-gap-verifier` (`wV:p41`, session `d56dae4f-c959-48d3-acf7-4ab6bedbc3bc`, actual `claude-sonnet-5-5`, startup `--effort medium`) independently inspected the complete five-file diff and ran `make validate && make e2e CI=1`: chain exit 0, 102/102 browser cases. All three criteria met; verdict ready, no blocking findings. Exact-code [CI 36993237388](https://github.com/azusachino/iroha/actions/runs/36993237388) succeeded with matching full SHA. No extra integration run was required for this test/doc-only diff; the prior Phase 1 integration record remains unchanged. Results and final-head CI are carried in [PR #106](https://github.com/azusachino/iroha/pull/106), tracking [#105](https://github.com/azusachino/iroha/issues/105).

These measurements close the resolved axis-text foreground-paint gap, not canvas glyph pixel geometry, antialiasing, clipping/overpainting conformance or color-vision accessibility. The helper checks each span's opacity, not parent-group opacity; arbitrary group-opacity effects are outside its model. The later private detail route overrides are not observed by the fixture's original unknown-request collector; their explicitly synthetic routing is independently source-checked, while public no-private-endpoint assertions remain directly observed. Matching pace forms are compared across the existing private detail and public table surfaces, not identical layouts. The fixtures deliberately preserve independent input quantities; they are not a physical consistency check between pace, distance and elapsed time. No production samples, transfer measurements, screen-reader or Firefox/Safari conformance is inferred. Unmounted MonthNavigator debt and the earlier unroot-caused initialization exception remain outside this follow-up.

The prior bounded Phase 1 report remains [historical evidence](2026-10-01-grapher-pilot-baseline.md). Phase 2 still requires its own scope and acceptance decision.
