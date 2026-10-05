# Cockpit UX improvement plan

Date: 2026-10-03. Updated: 2026-10-04. Status: owner-approved implementation. Original review baseline: main `e06fcad6ee004993a76d2998f3a95d0bc6945872`. Consolidated working baseline: Reports commit `2b2a52e231d420cd5f417c1d31b5e6a2258dbe15`.

Evidence and finding IDs belong to the [cockpit UX review](../audits/2026-10-03-cockpit-ux-review.md). The owner approved the entire plan under [#121](https://github.com/azusachino/iroha/issues/121), then directed all remaining cockpit work and safe local [#116](https://github.com/azusachino/iroha/issues/116) acceptance into existing [PR #125](https://github.com/azusachino/iroha/pull/125), rather than new staged PRs. The thin implementation slices and verification checkpoints below still apply within that PR.

On 2026-10-05 the owner directed pushing PR #125 for personal final review and consolidating all unfinished work in existing [#116](https://github.com/azusachino/iroha/issues/116). Local independent gates passed; the incomplete rendered matrix/scores, clean final image identity and submitted-head CI are review followups, not silently accepted exceptions. No merge-readiness claim is made.

Live tasks remain in Asobi `iroha:cockpit-ux-implementation` and `iroha:remaining-acceptance`. Motion, Night and Library are merged; the Reports slice passed independent verification and exact-head CI before this scope expansion. Its evidence does not certify subsequent changes. GitHub #121 was closed by the owner before the plan finished; the remaining acceptance scope is recorded here and in PR #125, not inferred from that issue state. Historical audit ownership remains unchanged.

## Outcome and constraints

Make failures unambiguous, exact evidence accessible and common journeys recoverable before expanding the visual system. Then make the compact Overview easier to scan without sacrificing scope,
provenance, links, text size or hit areas. No shell replacement is needed for the first batches.

- Preserve Grapher, existing shared tokens, fonts, ECharts, public/private boundaries, date/query semantics, exact tables and exports.
- Application adapters own requests/navigation. Shared view contracts and presentations live in `packages/iroha-shared`; never add host-only theme assets or imports back into application API clients.
- Use `createAsyncResource` for route-level async state. Initial unavailable, confirmed empty, confirmed zero and retained-data refresh are distinct states. Do not infer readiness from `!loading`.
- Stable/LTS tool selection and frozen Bun application dependencies remain. No new dependency, numeric floor reduction, skipped assertion, changed retry policy or report-only substitute for a regression gate.
- No deployment, release, live recovery/database/migration or edge-policy work. Those need separate ownership/approval and verification.

## Order and slice ownership

The following batches are coherent implementation checkpoints within PR #125, not permission for one large patch. Each route sub-slice below has its own focused test and can be independently reviewed. Use one writer per checkout; do not
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
   Admin tabs are checked separately: root `scrollWidth` passing is not enough to establish control/content reachability. The replacement audit found selected tabs off-screen on initial load at 320/375px and an Imports file input extending to x=338 at 320px. On 2026-10-05 the owner explicitly chose to fix both in PR #125 rather than defer them; preserve native upload behavior and keyboard tab navigation, with focused bounds/long-file-name regression checks.
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

## Approved consolidation and design checkpoints

The owner selected these criteria on 2026-10-04:

- Finish every remaining #121 slice and safe local #116 verification in PR #125. Do not open additional Iroha implementation PRs for this work.
- At 320×900, the first meaningful Overview pattern panel starts at `y < 900` in equivalent sparse and dense synthetic fixtures. Preserve information, scope/provenance, links, text/control sizes and hit areas; do not solve this by hiding content or globally shrinking type.
- Add Metrics under Analyze, preserving command-palette access, existing URLs and scope behavior.
- Approve an expanded replacement cockpit audit as explicitly new evidence. It does not reconstruct the missing historical artifact or change its original owner's claim/status.
- Record explicit approved deferrals for live-only recovery, migration, release and edge-policy evidence. No live operation, version/tag publication, deployment, real-data access or public edge-policy change is authorized.

## Local #116 acceptance and approved deferrals

These requirements supplement the existing route slices; source exploration determines the smallest owning fixture/helper changes before implementation.

| Item | Falsifiable local acceptance | Boundary / owner decision |
| --- | --- | --- |
| Shared coverage | Final report inventories actual app/shared sources and passes unchanged floors; link delivered PR #118 rather than redo its workspace migration. | Already delivered; fresh final provenance is required, not a reduced threshold. |
| Missing full audit | Enumerate production routes/details and publish the expanded route/state/viewport matrix described above, with actual commands, bounded synthetic evidence, applicable omissions and six-dimension findings. | Owner explicitly approves this as replacement acceptance, not recovered history. Preserve historical ownership. |
| Recoverability | Document actual owning boundaries and local source/fixture evidence separately from operational proof; every unresolved alerting, backup-before-migration and restore-drill claim is individually listed as an approved deferral. | Live failure alert delivery, real backup sequencing and isolated operational restore drills are deferred. No cluster changes or live exercise in this PR. |
| Release hygiene | Establish reproducible local verification of clean immutable annotated-tag/source identity and OCI revision provenance using existing tools and throwaway synthetic Git/image fixtures; negative cases must fail. Record commands, reviewed source and what an actual image build establishes. | Real version bumps, published tags/releases/images and production provenance/rollout evidence are deferred. No weakening of release checks or thawed dependencies. |
| Public serving | Isolated fixture proves one encoded snapshot representation, ETag identity and RFC-grounded conditional GET/304 behavior across matching and changed revisions. Source exploration found no existing ETag/304 implementation, so this is a new HTTP semantic requiring an owning ADR and independent contract review, not certification of an existing feature. Public/private disclosure and edge caching policy stay unchanged. | Live edge caching and disclosure/removal-policy changes remain deferred; an HTTP behavior correction must preserve existing published semantics and receive independent contract review. |

Source exploration also found per-request snapshot encoding, no OCI revision labels or clean annotated-tag preflight, and a historical lightweight `v0.4.5` tag. These are recorded gaps, not delivered release acceptance. Do not mutate published historical tags; implement prospective local verification with synthetic negative fixtures and distinguish actual image-build evidence from mocked inspection.

Use the existing #116 acceptance issue and PR #125 as the owning delivery record. Build release/public-serving fixtures only after tracing the actual transport/encoding/build paths; existing checks may already establish parts of this contract and should be reused explicitly. Do not manufacture evidence for deployed behavior from source inspection.

## Consolidated delivery sequence

1. Preserve the completed Reports slice, then implement Expenses, Patterns and To-go read/recovery parity one route at a time.
2. Complete route/detail/pagination recovery, exact Library distributions, navigation state and context-safe Today shortcuts in independently testable slices.
3. Apply the approved Overview geometry, public/compact reachability and Analyze/Metrics choices after trust and exact-data checkpoints.
4. Complete safe local #116 fixtures and the explicit operational-deferral record; reuse verified shared coverage without changing floors.
5. Run the expanded replacement audit and full owning gates on the combined source, obtain fresh independent Sonnet-class/medium acceptance against this plan, promote actual evidence and require CI on the latest submitted PR head.

One lead writes the checkout. Herdr peers may perform bounded read-only exploration/review, never concurrent source/index mutations. All long gates run in the background with direct Herdr completion/blocker messages; use actual job completion rather than fixed long sleeps. Use scoped `make local-check` groups during iteration; cached or earlier-slice success is not final consolidated acceptance.

No task is DONE because another route or a previous commit passed. PR #125 remains a normal review PR while CI runs, as requested by the owner; readiness is not gate acceptance. Merge/deployment remain owner actions.
