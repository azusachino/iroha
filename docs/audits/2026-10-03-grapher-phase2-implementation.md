# Grapher Phase 2 implementation acceptance

Status: Phase 2 accepted at code `314e293`; the subsequent public CI startup failure is diagnosed and corrected below. Final review, exact-head CI and PR readiness are tracked in PR #108. Owner explicitly requested implementation in the existing [PR #108](https://github.com/azusachino/iroha/pull/108), approving the [four-pilot decisions and criteria](../plans/2026-10-02-grapher-phase2-chrome.md). [Issue #107](https://github.com/azusachino/iroha/issues/107) carries acceptance, not a deployment/release request.

## Scope and ownership

- Shared `RouteHeader` owns title/context presentation. Existing private selector wrappers, public year buttons and app callbacks retain scope/request/navigation ownership. Overview is read-only and mixed-window; no new selector or lifetime aggregation.
- Shared `PanelFrame` owns plot/record/geography presentation. `MetricPanel` opts in with `framed`, while retaining required identity, rows, method/coverage/source metadata, chart/table selection and exact CSV behavior. Default unframed behavior remains for nonpilot callers.
- Heatmap, ledger and public monthly/cumulative widgets opt into embedded presentation only where an external shared frame is supplied. Default standalone behavior remains unchanged.
- Shared `StatTile` accepts visible context and an optional action snippet. Active Grapher Overview no longer has a competing stat-card style; its existing streak moved from header to summary. Sleep/library links and independent loading/failure/missing states remain. Expenses presents the existing currency totals separately, never adding currencies together.
- Expenses snapshots request dimensions and carries the actual chart currency with the loaded bundle. Pending currency selection no longer relabels retained plot units. Default chart currency agrees with the nonzero currency totals rather than choosing a zero bucket first.
- Public year/sport summaries use sanitized loaded records. Running count explicitly remains year-only/all-running-records, independent of sport selection. City context is geography-only; year/sport resets city exactly as before. Public detail/back behavior is not migrated.
- Overview geography precedes exact recent records in DOM and visual order. Recent-list context is up to five records; sleep is recent records with limit 30, not 30 days.

Other themes/nonpilot compositions retain their standalone frames and existing behavior. This is not universal Phase 2 rollout, a shell-rail regrouping, Liquid Glass adoption or a new chart/data/privacy contract. No dependency, font asset, registry or CI setting changed.

## Regression evidence

`chrome-header.spec.ts` first failed for the absent compact region, then passed with a loaded-period deferred change. `chrome-frame.spec.ts` checks distinct metric/ledger capabilities, provenance, chart/table exact values and CSV filename/raw values/unit. `chrome-summary.spec.ts` checks scope, independent missing states, preserved links, pending-currency truthfulness, zero-amount geometry in both modes and geography-before-record ordering.

`chrome-layout.spec.ts` covers all four pilots at 320/768/1280 in both modes, each private navigation disclosure, actual Tab cycles and viewport reachability. Its long-title case changes the mounted heading text as a bounded CSS stress fixture, not a new application route or data contract. Screenshots capture the ordinary populated state before that injection. Native Chrome 200% zoom runs each private disclosure separately and both public modes at 320 CSS pixels; all original layout, keyboard, paint and fixture assertions remain, with no floor exception.

Opening navigation added two bounded fixes without changing groups/layout: hover departure must not close a keyboard-focused submenu, and active/hover link foreground uses the text token because Expenses' active mint foreground measured 4.307:1 against its tinted surface, below the existing 4.5:1 text floor. Keyboard opening uses actual Tab/Enter, not pointer coordinates at native zoom. The dedicated hover test uses Playwright's clock installed before application timers, with no fixed sleep.

Relevant documented patterns: [Svelte 5 typed/optional snippets](https://svelte.dev/docs/svelte/snippet#Passing-snippets-to-components-Optional-snippet-props) and [Playwright clock initialization and runFor](https://playwright.dev/docs/clock#consistent-time-and-timers). Existing pinned Svelte 5.57.1/Playwright 1.63.0 tooling and Make targets are retained.

### Lead gates before independent review

Run from `vendor/iroha`:

| Command | Result |
| --- | --- |
| `make validate` | Exit 0; unchanged quality floor, type/unit/build/bundle and document checks. |
| `make e2e CI=1` | Exit 0; 185/185 at the main implementation, then 186/186 after the initial-error header regression. |
| `make test-integration` | Exit 0; owning disposable test-database lifecycle. |
| `make e2e ARGS='e2e/chrome-summary.spec.ts'` | Exit 0; 5/5. |
| `make e2e-pilot-audit OUT=…/pilot.json` | Exit 0; 35/35 report-only collection cases. |

An earlier combined gate invocation hit its 20-minute harness deadline after 151 passing cases, without a completed browser verdict. Its evidence is retained as `chrome-full-e2e.interrupted.log`; only its orphaned owned preview PIDs were stopped. An unchanged-source later full browser run passed 182/182. The interruption is not diagnosed or called flaky.

Several development failures were concrete feedback, not conformance: initial header exceeded 8rem; unshrunk header actions overflowed narrow public layout; implicit frame grid tracks pushed range controls outside native zoom bounds; old locator/type probes needed migration to the new actual roles. Floors and assertions were retained. Fixture expected values/methods were corrected against their source, not against production guesses.

## Independent acceptance and remaining CI gate

Fresh verifier `iroha-p2-verifier`, task-created pane `wV:p44`, session `fa79fa91-d19d-45fa-93f4-8331510fb79b`, reviewed exact code `f5dfcf124b4ace285b1aeb2d3c40547a388d1711`. Actual model access: `claude-sonnet-5-5`; startup `--effort medium`. It independently ran `make validate`, `make e2e CI=1` (185/185) and `make test-integration`, all exit 0, plus bounded journey probes. No tracked/index mutations.

| Criterion | Independent verdict / evidence |
| --- | --- |
| P2-A1 | Met: four compact populated headers, no overlap/overflow at 320/768/1280 in both modes; bounded mounted-long-title stress cases wrap. |
| P2-A2 | Met: existing month/year/rolling/public-year behavior and reset-city journeys; no lifetime or new Overview route selector. Existing Overview 6M/12M/24M is chart-local, not route-level. |
| P2-A3 | Met: canonical frame imports, no nested visible frames, unchanged nonpilot defaults; exact expense/metric table/CSV filenames, raw values and units; no map/ledger metric fabrication. |
| P2-A4 | Met: additional nonzero sleep/library, independent failure and unknown/partial-distance probes, preserved links, deferred currency/month identity and geography-only scope. Extra probes are bounded scratch evidence, not retained source regressions. |
| P2-A5 | Met: full open-disclosure/native-zoom checks and supplementary Tab-order probes; geography precedes records in DOM/layout. |
| P2-A6 | Met: full contrast/motion/tabular suites and unchanged floors; compact-header role change replaces display with title, not an assertion deletion. |
| P2-A7 | Met: equivalence/palette/privacy checks, no public private/unknown requests, no dependencies/CI/Make/constraint changes. |
| P2-A8 | Met: exact-code owning gates, fresh independent review and final-code CI passed. The historical f5dfcf1 failure remains unresolved. |

Exact-code recheck at `314e29308db508f90c9bea7a2a09b1f0ebbe1c82` independently passed validate, all 186 browser cases and integration (exit 0), with clean tracked/index state. The reviewer acknowledged the correction and audit accuracy. An additional failed-refresh probe kept observed 2026-08 scope, four spending tiles and its record when HTTP 503 was injected. No loaded scope was replaced by failure text.

Reviewer found no blocking source findings. Its initial Expenses error header said “Loading ledger period…” after failure; corrected to unavailable/not-loaded states, preserving observed scope whenever retained data exists, with a source regression. Other notes are retained rather than silently changing scope: narrow six-column Overview tiles can be tall and add named landmarks; generic public frame accessible names and verbatim ledger-period wording; existing cross-currency minor-unit sorting; public record-table horizontal clipping not baseline-compared; nonzero sleep/library and public geography/order probes lack equivalent retained source fixture depth. These are follow-up considerations, not broader rollout or blanket conformance.

CI37045770918 at exact `f5dfcf1` failed 16 public browser cases and reported one retry-only pass, while validate/build and integration passed. Public serving response step did not run. At that checkpoint the retained public chart/bootstrap failures were not diagnosed, fixed or classified as flaky; the later investigation below supersedes that diagnosis status for the matching exceptions. No CI rerun was authorized or performed. The initial-error header correction is unrelated to those public failures. Subsequent source-triggered CI37051262855 at `314e293` passed validate/build, integration, browser tests and public serving. This was not a rerun of the failed job and the private-header correction did not diagnose or fix the earlier public failures. The historical failure remains unresolved; the final current-source gate is satisfied. Documentation-only closeout and its CI are recorded in PR #108.

## Public startup CI correction

Latest documentation-only CI37053239465 at `ae95b4a` failed 14 public cases plus two retry-only passes despite unchanged runtime code from the preceding successful run. Its primary exception was in ECharts' `setOption`: `Cannot read properties of undefined (reading '__ec_inner_N')` during axis statistics. The subsequent `getVisual` exceptions came from models whose visual update never completed. The interrupted Svelte flush also left bootstrap feedback behind after page content existed.

Both Vite hosts resolved host chart imports to their own `node_modules/echarts`, but shared component imports to `packages/iroha-shared/node_modules/echarts`. Separate physical copies duplicate ECharts' cycle-cache/inner-state keys. A bounded mixed-copy Vite reproduction (charts from shared, core/components/renderers from public host) failed three of three browser startups with the same `__ec_inner_N` exception. This is evidence for this particular failure mechanism, not a claim that every historical public failure has the same cause.

The retained `chart-resolution.spec.ts` regression uses each consumer's actual Vite configuration/resolver and real shared/host importer paths. Before correction both consumers failed on unequal resolved core paths. The minimal fix adds `resolve.dedupe: ["echarts", "zrender"]` to both host configurations, following [Vite's documented linked-package deduplication](https://vite.dev/config/shared-options.html#resolve-dedupe). No package, dependency version, lockfile, chart implementation, assertion, retry or threshold changed.

`chart-initialization.spec.ts` retains a browser startup journey asserting no page exception, bootstrap removal, two initialized charts and resolved series colors. The resolution regression checks core/charts/components/renderers and zrender paths from both importer homes. Focused repeated checks passed 15/15 after the fix; `make validate`, `make e2e CI=1` (189/189) and `make test-integration` exited 0. No inspector catches, artificial sleeps or weakened visual assertions were added. Final independent review and source-triggered CI are promoted in PR #108 before readiness.

The [Phase 1](2026-10-01-grapher-pilot-baseline.md) and [renderer follow-up](2026-10-02-grapher-verification-gaps.md) limits remain: Chromium synthetic journeys, not screen-reader/Firefox/Safari/color-vision/production conformance; resolved canvas text foreground, not antialiasing, glyph clipping/arbitrary overpainting or parent-group opacity. Private detail overrides retain their prior collector-bypass limit. No original missing cockpit audit was reconstructed. Prior chart-initialization and proposal CI attempt-1 public failures remain unroot-caused; successful retries did not establish a fix.
