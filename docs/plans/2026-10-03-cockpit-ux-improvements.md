# Cockpit UX improvement plan

Date: 2026-10-03. Status: proposed; review/planning only is authorized. Baseline: main `e06fcad6ee004993a76d2998f3a95d0bc6945872`.

Evidence and finding IDs belong to the [cockpit UX review](../audits/2026-10-03-cockpit-ux-review.md). [Issue #116](https://github.com/azusachino/iroha/issues/116) retains the wider acceptance gaps. Live delivery
of this review is tracked in Asobi `iroha:cockpit-ux-review`; implementation tasks are not claimed or started. Record an owner's selected slices in their own issue/Asobi epic before implementation.

## Outcome and constraints

Make failures unambiguous, exact evidence accessible and common journeys recoverable before expanding the visual system. Then make the compact Overview easier to scan without sacrificing scope,
provenance, links, text size or hit areas. No shell replacement is needed for the first batches.

- Preserve Grapher, existing shared tokens, fonts, ECharts, public/private boundaries, date/query semantics, exact tables and exports.
- Application adapters own requests/navigation. Shared view contracts and presentations live in `packages/iroha-shared`; never add host-only theme assets or imports back into application API clients.
- Use `createAsyncResource` for route-level async state. Initial unavailable, confirmed empty, confirmed zero and retained-data refresh are distinct states. Do not infer readiness from `!loading`.
- Stable/LTS tool selection and frozen Bun application dependencies remain. No new dependency, numeric floor reduction, skipped assertion, changed retry policy or report-only substitute for a regression gate.
- No deployment, release, live recovery/database/migration or edge-policy work. Those need separate ownership/approval and verification.

## Order and slice ownership

Suggested PR batches are coherent outcomes, not permission for one large patch. Each route sub-slice below has its own focused test and can be independently reviewed. Use one writer per checkout; do not
split host and shared-contract changes between concurrent writers. Do not mark a whole batch complete because its first route passed.

| Batch / slice | Finding | Outcome and likely paths | Dependencies / scope |
| --- | --- | --- | --- |
| 1a Motion state reference | F1 | Distinguish unavailable summary/list/series from confirmed zero; host `motion/+page.svelte`, shared activity view contract and Grapher Activities, focused E2E | First; M |
| 1b Night state parity | F1 | Apply the verified pattern to Night summaries/history without losing partial available totals; Night state/host, shared sleep contract/composition, E2E | 1a pattern accepted; M |
| 1c Library state parity | F1 | Unavailable aggregates/list are not zero titles or no completion history; Library state/host, shared media contract/composition, E2E | 1a pattern accepted; M |
| 2a Reports identity | F1, F9 | No invented empty report on initial failure; selected and observed month remain distinct on pending/failed refresh; Reports host, shared Reports props/composition, E2E | Reproduce F9 before fixing; M |
| 2b Expenses and Patterns state parity | F1 | Separate route-sized sub-slices: preserve explicit unavailable versus successful-empty ledger/pattern data; each host, existing shared composition, E2E | 1a pattern accepted; each S/M |
| 2c To-go recovery state | F1, F3 | Separate initial task/job reads and mutation/poll errors using existing resource abstraction; To-go host and tests, shared presentation only if reusable | 1a pattern accepted; M |
| 3a Recovery and pagination | F3, F4 | One route at a time: named retry, announced failure, retained rows and visible second-page error; Today/Patterns/detail adapters, then Motion/Night/Library state and E2E | Successful populated/second-page fixtures needed; each S/M |
| 3b Library exact evidence | F2 | Three existing distributions have meaningful labels and exact categorical tables; shared MediaBarChart or existing BarChart integration, Grapher Media and E2E | Populated fixture first; M |
| 3c Navigation/control semantics | F5, F6 | Separate small slices for route/filter states and Today modifier/widget-safe shortcuts; existing navigation/shared control source, Today state and tests | Reproduce shortcut contexts; each S |
| 4a Compact Overview | F7 | Reduce redundant scope and tall summary composition locally, retaining truthful mixed windows and sleep/library links; shared Dashboard/StatTile opt-in behavior, layout tests | Batches 1–3 trust/accessibility checkpoint; M |
| 4b Record/compact reachability | F8 and audit layout limits | Public table is one S/M slice; separately triage nonpilot body extents and Admin tabs with real focus/scroll checks before proposing fixes | Per-route fixtures; each S/M |
| 4c Metrics discovery decision | F10 | Owner chooses an Analyze entry or documented command-first role; navigation/Manual and navigation tests | Owner decision; S |

S means roughly 1–2 implementation files plus tests; M means roughly 3–5 files. If a route slice expands beyond that, stop and split it before continuing. Do not extract a new generic state/view framework
just to make the table look uniform. Reuse the existing resource, loading boundary, retry and chart primitives first.

## Acceptance and verification per batch

### Batches 1–2: trustworthy reads

Each route-sized slice must meet all three conditions:

1. Deferred first load and HTTP 503 never announce a genuine zero or successful-empty history; no fabricated DTO reaches a visible report/summary.
2. A successful-empty response still shows the truthful empty state. A successful populated response retains existing exact values, units, null semantics and links.
3. During deferred/failed refresh, prior data remains accessible with its observed scope; requested scope is not presented as loaded scope. Recovery clears error feedback without stale-response resurrection.

Add persistent Playwright fixtures for the route's real DTOs. Do not use the pilot fixture's unknown-endpoint 404 as a successful-empty response. Test independent dependencies failing separately, not just
an all-503 scenario. Start with Motion; its accepted host-to-shared state contract can inform, but must not automatically certify, the other routes.

For Reports, demonstrate F9 with a loaded August report, delayed September request and September HTTP 503 before changing code. Assert both selected and observed labels explicitly. For To-go, keep task/job
reads and add/finish/action/poll errors separate; do not clear a real mutation failure merely because an unrelated read succeeded.

### Batch 3a: usable recovery

For each route migrated:

1. Initial failure exposes a named keyboard-operable retry and useful context; activation recovers without reload. Existing back/navigation links remain available.
2. Second-page failure preserves exact rows and cursor, announces the failed page near Load more and permits retry without duplication.
3. Retry/poll success clears only the relevant error; routine refetch never reactivates first-load inert content.

Use `private-states.spec.ts` as the proven deferred/retry pattern, not as evidence for new routes. Extend real fixture handlers and assert request changes and exact retained rows.

### Batches 3b–3c: accessible evidence and controls

1. Each populated Library distribution has a meaningful accessible name and exact category/count table; no existing category, ordering, reduced-motion or chart identity behavior regresses.
2. Active route and filter state is announced; normal Tab/Enter and existing disclosure focus-return behavior remain usable.
3. Today day shortcuts work only in their intended context; modifiers, browser navigation and focused widgets/dialogs are not intercepted.

Verify shared/host placement and import direction; test both chart consumers if a shared primitive changes. Use unit target/modifier cases plus actual Chromium keyboard journeys. A DOM label assertion is
not a screen-reader acceptance claim.

### Batch 4: compact presentation

1. Overview's first meaningful pattern panel is visible sooner than the measured y=1179 at 320×900, with no lost information or globally reduced font/control sizes. Establish the exact geometry target with
   the owner before implementation; compare equivalent sparse and dense fixtures.
2. Public record columns/sort controls are discoverable and keyboard/touch reachable at compact widths. Internal overflow is intentional and named; there is no exact-data loss. Nonpilot body extents and
   Admin tabs are checked separately: root `scrollWidth` passing is not enough to establish control/content reachability.
3. Metrics navigation follows the explicit owner choice and preserves command-palette discovery, URLs and scope behavior.

Do not promise a global 8rem header or all-route shared-frame migration from these local slices. The earlier Phase 2 pilot criteria are context, not universal acceptance.

## Checkpoints and owning gates

At each route-sized slice: focused unit/browser tests, source placement/import checks and screenshots in both modes. Before commit/PR: `make check`, `make fmt-docs-check` for changed docs and `make validate`.
Before a runtime batch is accepted: complete `make e2e CI=1`, owning integration/public-serving gates where applicable, exact-head CI and fresh independent Sonnet-class/medium verification. Synthetic local
passes never replace submitted-revision CI. Run DB-backed gates only against the owning disposable test lifecycle; no live data access is authorized by this plan.

Before broader cockpit quality acceptance, extend coverage to every production route/detail at 320/375/414/768 in both themes, plus a desktop reference; include populated, confirmed-empty, first-load,
first failure, partial dependency, deferred refetch, failed refetch and successful retry where applicable. Retain keyboard/focus, native zoom, reduced-motion, chart/table/export parity and public privacy
assertions. Score the design contract's six dimensions only once that route evidence exists. Publish omissions honestly; do not replace the missing historical audit with a four-pilot pass.

## Decisions and risks

| Decision / risk | Handling |
| --- | --- |
| Review shows genuine failure-state defects, but not all source candidates are reproduced | Confirm F4/F6/F9 journeys before choosing implementation; retain current evidence and separate reproduced bugs from improvements. |
| Shared primitive changes can affect both hosts or nonpilot callers | Prefer composition-specific opt-in behavior; verify each runtime consumer and unchanged defaults. |
| Compacting summaries could remove meaningful scope/coverage | Keep mixed-window disclosure, unknown/missing states and links as explicit assertions; reduce repetition before reducing type. |
| Dense real histories differ from sparse synthetic data | Add bounded high-volume/long-label fixtures before judging density, chart legibility or pagination. Do not inspect live personal data without separate approval. |
| Public overflow and Metrics discovery have multiple valid designs | Owner chooses compact-table treatment and Metrics role at the Batch 4 checkpoint. |

Next owner checkpoint: accept or revise this prioritization, then select the first trustworthy-state slice. Approval of the audit/plan is not approval to redesign, deploy or close the remaining #116
acceptance scopes.
