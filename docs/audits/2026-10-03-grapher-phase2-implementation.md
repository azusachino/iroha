# Grapher Phase 2 implementation acceptance

Status: implementation present; fresh independent acceptance pending. Owner explicitly requested implementation in the existing [PR #108](https://github.com/azusachino/iroha/pull/108), approving the [four-pilot decisions and criteria](../plans/2026-10-02-grapher-phase2-chrome.md). [Issue #107](https://github.com/azusachino/iroha/issues/107) carries acceptance, not a deployment/release request.

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
| `make e2e CI=1` | Exit 0; final 185/185, including summary geometry/order cases. |
| `make test-integration` | Exit 0; owning disposable test-database lifecycle. |
| `make e2e ARGS='e2e/chrome-summary.spec.ts'` | Exit 0; 5/5. |
| `make e2e-pilot-audit OUT=…/pilot.json` | Exit 0; 35/35 report-only collection cases. |

An earlier combined gate invocation hit its 20-minute harness deadline after 151 passing cases, without a completed browser verdict. Its evidence is retained as `chrome-full-e2e.interrupted.log`; only its orphaned owned preview PIDs were stopped. An unchanged-source later full browser run passed 182/182. The interruption is not diagnosed or called flaky.

Several development failures were concrete feedback, not conformance: initial header exceeded 8rem; unshrunk header actions overflowed narrow public layout; implicit frame grid tracks pushed range controls outside native zoom bounds; old locator/type probes needed migration to the new actual roles. Floors and assertions were retained. Fixture expected values/methods were corrected against their source, not against production guesses.

## Acceptance boundary

P2-A1–A7 have lead source/runtime evidence in the regressions above; P2-A8 remains pending fresh independent review and its exact-head gates. Do not mark PR ready based on this lead record alone. Promote the independent criterion verdicts and exact reviewed revision into this document and PR before completion.

The [Phase 1](2026-10-01-grapher-pilot-baseline.md) and [renderer follow-up](2026-10-02-grapher-verification-gaps.md) limits remain: Chromium synthetic journeys, not screen-reader/Firefox/Safari/color-vision/production conformance; resolved canvas text foreground, not antialiasing, glyph clipping/arbitrary overpainting or parent-group opacity. Private detail overrides retain their prior collector-bypass limit. No original missing cockpit audit was reconstructed. Prior chart-initialization and proposal CI attempt-1 public failures remain unroot-caused; successful retries did not establish a fix.
