# Motion initial and filtered read states

Date: 2026-10-03. First implementation slice of [#121](https://github.com/azusachino/iroha/issues/121), following the cockpit plan in [PR #120](https://github.com/azusachino/iroha/pull/120). This does not complete the plan or the historical acceptance gaps in [#116](https://github.com/azusachino/iroha/issues/116).

## Scope

The private Motion adapter and its Grapher presentation now distinguish pending, failed, confirmed-empty, and retained reads. No deployment, live-data operation, dependency update, coverage-floor change, public-data contract change, or global typography redesign is included.

- Records, summary, canonical series, and available date range have separate failure/retry feedback.
- Failed summary reads show unavailable values rather than invented zero totals. Existing unknown-distance counts also produce unavailable distance, with an explicit missing-distance note.
- Retained records, summary values, and exact series retain their observed period/sport scope during a deferred or failed selection change.
- Pending responses cannot overwrite a newer selection. Explicit resource invalidation also covers lifetime queries whose series window cannot yet be established.
- Keyboard retries target the recovered summary/records/chart region or the year control for date-range recovery, without taking focus from another control selected while waiting. The existing RetryNotice opts into this behavior through a focus target; other consumers retain their default behavior.
- Paging cannot append an old query to the current list. Stale-scope paging controls are hidden. Pagination-failure messaging remains a later plan slice.

## Contract checks that changed the implementation

The summary HTTP handler resolves a month date to its **year**, then calls `SummaryInTimezone`. `by_month` is filtered by year/sport, `by_year` ignores year, and `by_sport` ignores sport. Using root totals for a month would therefore report year-wide values under a month label. Motion derives the month bucket from the stored observed request. A missing bucket in a successful year response means an empty month; a failed response remains unavailable. The fixtures mirror this server behavior instead of pretending it returns month-only totals.

The existing 600px infinite-scroll sentinel can start paging before a manual click. The stale-append regression holds the response before navigation and exercises that sentinel. Earlier manual-click probes failed when automatic paging consumed or renamed the control; no retries, assertions, or fail-on-flaky gates were weakened to hide this.

## Verification

Lead gates on the corrected source (2026-10-04): targeted Motion **30/30**, complete Chromium **237/237**, `make validate` **exit 0** with **228 Vitest tests** and zero Svelte errors/warnings, and `make fmt-docs-check` **exit 0**. The stale-append regression also passed six explicit repeat executions on the earlier sentinel correction. The independent verification below passed; submitted-head CI remains required before acceptance.

Earlier attempts are not hidden: two date-range focus assertions and typecheck failed when a function was incorrectly passed to an HTMLElement-only focus-target prop. Passing the actual year select resolved both. Frozen-install attempts also failed during intermittent certificate verification failures; no TLS check, assertion, coverage floor or flake policy was relaxed.

Synthetic Chromium journeys cover both light and dark modes: initial pending/failure, separate dependency failures/recovery, unavailable date range, successful empty lifetime and month, preserved request parameters, unknown distance, retained scope, superseded responses, lifetime-window failure, stale append, keyboard focus, and focus not stolen from another user target. The exact gate results are recorded above and below; the owning PR carries the reviewed revision and submitted-head CI.

The browser preset uses `PUBLIC_IROHA_TIMEZONE=Asia/Tokyo`; timezone assertions check request pass-through for that preset, not every timezone or DST transition. The fixtures use synthetic records only. They do not replace DB-backed integration or prove dense-history, screen-reader, Safari/Firefox, or blanket mobile conformance.

### Independent verification

Fresh owner-approved replacement `iroha-motion-final`, session `a36b631f-9eac-4082-931e-e1447bbfb024` (task-created Herdr pane `wV:p59`), Sonnet 5.5. Medium effort is established by the native startup argv (`--model sonnet --effort medium`) and visible `thinking with medium effort` status. The reviewer initially inferred “low” from an unscaled internal field, then explicitly withdrew that inference; it is not the basis of model/effort verification.

Reviewed base: `e06fcad6ee004993a76d2998f3a95d0bc6945872`. The 15-path staged binary diff fingerprint before and after independent gates was `222b2ab00723f4624a893f00fadb56d32b36e892ad70c99fcd90d60c8cec98fb`. This paragraph is a later evidence-only addition, not part of that fingerprint.

Between 2026-10-04 03:25:23Z and 03:30:36Z, the verifier independently ran the five commands under Reproduce, serially: `make validate` (228 Vitest, zero Svelte errors/warnings), targeted Motion (30), stale-append repeats (6), complete Chromium (237, zero failed/flaky), and docs gate. **Every command exited 0**; source/index were unchanged. All nine criteria were MET: truthful read states, observed scope, independent dependency recovery, API/query/null semantics, stale paging, bounded keyboard recovery, invalidation regressions, unchanged gates/boundaries, and truthful evidence. No blocking findings remained.

Rendered review was bounded: the verifier viewed two original samples (unavailable-light and populated-dark) at source review, then only unavailable-dark after refresh, confirming its corrected header and unavailable values. The lead inspected all four samples. Unchanged populated byte sizes were compared, not a new visual review. Two worker-load console messages in the full suite remained untraced by the verifier; Motion and the no-console-error smoke passed. The lead separately confirmed that existing private/public worker-failure injection tests are retained. No claim that every printed console message was individually attributed is made.

## Rendered evidence provenance

These are synthetic, one-state screenshots from `e2e/motion-states.spec.ts`, manually copied from Playwright's `info.outputPath` after a successful run. They are presentation samples, not an acceptance matrix. The unavailable case uses 320×900; the populated case uses the default 1280×720 viewport. Full-page capture extends image height. Populated capture follows successful summary recovery; focus/scroll are neutralized after assertions to avoid capturing a sticky sidebar at the previous scroll offset.

| Committed image | Originating test, for its light/dark mode |
| --- | --- |
| [Unavailable light, 320](evidence/2026-10-03-motion-read-states/unavailable-light-320.png) | `failed reads are unavailable, not zero history` |
| [Unavailable dark, 320](evidence/2026-10-03-motion-read-states/unavailable-dark-320.png) | `failed reads are unavailable, not zero history` |
| [Populated light, 1280](evidence/2026-10-03-motion-read-states/populated-light-1280.png) | `summary failure preserves independent records and chart` |
| [Populated dark, 1280](evidence/2026-10-03-motion-read-states/populated-dark-1280.png) | `summary failure preserves independent records and chart` |

Distance and duration series form **one paired dependency** (`Promise.all`), not independently retryable reads. Duration-only failure is not covered by these fixtures. Fixtures model UTC midday records and timezone query pass-through, not timezone-boundary aggregation; sport examples are run/walk, not compound sport normalization.

## Reproduce

From the owning repository, using the managed tool environment:

```sh
PUBLIC_IROHA_TIMEZONE=Asia/Tokyo make e2e ARGS='e2e/motion-states.spec.ts --workers=1'
PUBLIC_IROHA_TIMEZONE=Asia/Tokyo make e2e ARGS='e2e/motion-states.spec.ts -g "stale append" --repeat-each=3 --workers=1'
make validate
PUBLIC_IROHA_TIMEZONE=Asia/Tokyo make e2e ARGS='--workers=2'
make fmt-docs-check
```

`make validate` includes the complete `make check` gates and private/public builds. Submitted-head CI additionally retains DB integration, isolated public-serving checks, both complete browser shards, and the fail-closed aggregate check.

## Limits and next slices

Night and Library initial/filter state parity follow this reference. Existing pagination failures, nonpilot detail retries, Library distribution exact-data access, navigation semantics, Today shortcuts, Reports retained-month behavior, compact layouts, and Metrics discovery remain separate tasks. Overview composition and Metrics role retain the plan's owner checkpoints. This record does not reconstruct or accept the missing historical full cockpit audit.

Existing limits are not repaired here: lifetime series windows over the backend's 3660-day read-scope cap cannot succeed on retry; compound sport normalization is duplicated in the existing client helper; the inactive non-Grapher fallback lacks these recovery controls. Grapher is the only registered theme in this slice.

Framework references: [Svelte untrack and tick](https://svelte.dev/docs/svelte/svelte#untrack), [Playwright request interception](https://playwright.dev/docs/network#handle-requests). Contract evidence: `apps/iroha-server/pkg/httpapi/activities.go` and `apps/iroha-server/pkg/activities/summary.go`.
