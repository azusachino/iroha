# Consolidated cockpit and local acceptance — verification record

Date: 2026-10-04. Working branch: `fix/reports-trustworthy-scope`. PR: [#125](https://github.com/azusachino/iroha/pull/125). Acceptance: [cockpit plan](../plans/2026-10-03-cockpit-ux-improvements.md), [#116](https://github.com/azusachino/iroha/issues/116).

## Status and historical boundary

Handed to the owner for final review on 2026-10-05. Independently verified local gates passed; this record does not establish merge-readiness. At the owner's direction, unfinished work is consolidated in [#116](https://github.com/azusachino/iroha/issues/116), rather than continuing the rendered audit before pushing PR #125. The Reports slice at `2b2a52e231d420cd5f417c1d31b5e6a2258dbe15` passed earlier independent acceptance and CI, but that evidence does not certify the expanded working diff. The owner closed #121 before the full plan finished. This replacement audit is new evidence, not recovery of the missing `2026-09-30-v0.6-cockpit-quality.md`; its historical ownership remains unchanged.

## Implemented behavior and bounded iteration evidence

- Expenses: separate ledger/spending/range reads, nullable amounts and counts, observed scope and independent named retries. Genuine zero-amount categories now remain in the exact totals/chart instead of being discarded by the former positive-only filter; this is an intentional zero-versus-unavailable correction, not a category/order change. Four genuine baseline failures; 22 focused synthetic Chromium cases passed.
- Patterns: scoped series/latest/day/range resources and truthful retained drill identity. Four genuine baseline failures; 24 focused cases passed.
- To-go: independent inventories, separate mutation/action/poll errors, receipt versus inventory distinction, Tokyo completion date and existing polling backoff. Four genuine baseline failures; 26 focused cases passed.
- Motion/Night/Library cursors: retained exact rows/cursor, named retries, request supersession and no automatic failure loop. Twelve new cases repeated three times passed (36). Original state checks also passed at route checkpoints. Response holds eliminate observer/sentinel races without sleeps or application retry-policy changes.
- Detail reads: scoped record IDs; Library retry/backlink; independent Night session/stages; independent Motion record/route/heart-rate/laps. Ten baseline failures. Ten cases repeated three times passed (30). Request-count checks distinguish completed browser responses from canceled bootstrap transport, preserving exact completed-read assertions.
- Today: resource-based briefing/tasks/calendar reads, separate write failures and unavailable sections, observed task date, independent keyboard recovery and modifier/widget-safe day shortcuts. Eight baseline failures; eight focused cases passed.
- Library distributions: unchanged categories/order, three named canvases with exact categorical count tables, genuine zero score and reduced-motion chart behavior.
- Navigation: active-route `aria-current`, active family `aria-pressed`, Metrics under Analyze with existing URL/command discovery.
- Overview: activity calendar precedes the full six-tile summary; compact tile spacing is opt-in, with no font reduction. All mixed-window disclosure, values and sleep/library links remain. Four equivalent sparse/dense, light/dark 320×900 checks passed the `y < 900` criterion.
- Public records: named focusable horizontal region, visible reachability instructions and 44px sort controls. Keyboard checks reach every column and retain records across both fixture years without accessing private APIs.

These are iteration checkpoints, not substitute gates. Expanded production route/state/viewport coverage, fresh combined independent verification and final submitted-head CI must be recorded before changing this status.

## Local #116 evidence

### Public snapshot representation

[ADR-0010](../adr/0010-public-snapshot-validators.md) records exact-byte JSON encoding and SHA-256 validators, standard-library precondition precedence, whole-document Range behavior and bodyless 304. The original validator test failed against the previous handler. Go tests passed after implementation. Independent contract research identified cacheable 412 responses as a counterexample; the handler now marks them `no-store`, removes representation metadata and retains a regression assertion. Final gates/review must include that correction.

Origin unit fixtures establish exact cached bytes, no re-encoding on a cache hit, weak/list/wildcard/repeated validators, malformed/stale validators, If-Match precedence, unchanged caller headers, and metadata-generation changes. This is origin evidence, not live edge/cache conformance. The existing isolated Caddy shell/security-header gate remains required.

### Prospective release identity

`make release-source-check TAG=<annotated-tag> ARGS='--expected-revision <sha> --expected-tag-object <object> --image <local-image>'` is read-only. It rejects dirty/untracked source, lightweight tags, tags not at HEAD, changed expected revisions/annotation objects, and missing/mismatched OCI revision labels. No existing tag is changed. Eleven synthetic Git/image-inspection unit tests passed (lead-run gates, including the later CLI failure cases); an actual throwaway annotated Git candidate and two locally built scratch images confirmed matching source/image identity and rejected a wrong revision.

Actual Linux arm64 server/private-web/public-web candidate images built successfully using the existing Containerfiles and frozen application locks. They were neither published nor imported into k3s. These early builds carried the explicit `2b2a52e231d420cd5f417c1d31b5e6a2258dbe15-dirty` label; they are build evidence, not immutable release provenance or final-commit certification. Local development compose defaults explicitly to `unverified-development`; release verification rejects that marker against a checked revision. A final clean source revision must receive its own image evidence.

### Shared coverage

The workspace/coverage migration was delivered in [PR #118](https://github.com/azusachino/iroha/pull/118), not redone here. Final validation must inventory all app/shared sources and retain the exact existing numeric floors. Earlier scoped web checks passed 228 Vitest tests with no Svelte errors/warnings; final provenance remains pending.

## Individually approved operational deferrals

The owner approved these boundaries for this PR. Each remains unverified; local tests do not close it.

| Deferred claim | Evidence still needed / ownership boundary |
| --- | --- |
| Live failed-job/import alert delivery | Owning service/operations owner must observe a real alert path; local poll/error rendering is not delivery proof. |
| Backup before a production migration | Operations owner must identify the actual backup, source revision and migration sequence before a separately approved live change. |
| Isolated operational restore drill | Operations owner must restore a real retained backup into an explicitly isolated environment and verify evidence/data invariants. Synthetic Git/image fixtures do not prove recoverability. |
| Published release/tag/image provenance and rollout | Release owner must approve publication, immutable annotated tag, clean checked source/image identity and actual rollout verification. Historical lightweight `v0.4.5` is not rewritten. |
| Live compressed-edge caching and disclosure/removal policy | Edge/privacy owner must verify the deployed origin/proxy combination and policy. No edge policy or sanitized projection expansion is authorized here. |

## 2026-10-05 checkpoint (not final acceptance)

Fresh source reviews verified the complete dirty-file manifests: round 1 had 66 paths; round 2 had 73 paths, each per-file SHA verified. No source-blocking defect remained after the first review's fixes. Source checks do not certify the rendered matrix or runtime gates. The requested medium effort was recorded in the native Sonnet launch arguments; the verifier could not independently see that setting inside its session.

The first review found a Today completion race that invalidated a pending next-day inventory without reissuing it. Four new synthetic browser cases failed before correction. The corrected inventory reload, single-shot completion control, partial-section notices (both Grapher and classic), observed task date, write failures and Escape/dialog contexts passed a 54-case repeated checkpoint with public navigation checks. The separate exact-table/Library/Overview/navigation checkpoint passed 48 repeated cases. Actual ECharts options, not just ARIA labels, proved reduced-motion animation false/duration zero and ordinary animation true/duration 500. Dense Overview fixtures include five long-labelled records, 31 active days and 5,000-record totals; assertions preserve all six tiles, values/context, fonts and existing 24px link targets.

Rendered investigation identified unnamed internal Motion/Night scrollers and cramped Library table headers. Native focus buttons and named regions now preserve all exact columns; the focus buttons are 44px. Their new tests exposed a real shared PeriodSelector defect: ArrowRight while a scroll region was focused changed August to September. The guard now preserves native widget/region keys and already-prevented events; tests assert both real horizontal scrolling and unchanged date. Library count-table padding keeps the two headers visibly separated.

The lead reports post-review `make validate` and `make fmt-docs-check` passed with private/public Svelte checks at zero errors and zero warnings; 228 Vitest and 147 Python tests passed. Web coverage was statements 13.47%, branches 14.79%, functions 11.43%, lines 13.52%; the existing floors were unchanged. Go/router tests cover positive If-Match variants, unchanged HEAD/POST routing, CORS/304, rate-limit precedence and real service build failure before validators. Revision-changing import/ETag assertions were added to the existing disposable DB integration test; execution remains required.

Two subsequent full browser runs were not accepted: the first had 430 passes and one flaky execution-context failure while build/generated-file checks were concurrent; the second had 436 passes and seven flaky timeouts during repeated macOS sleep/wake cycles. Fail-on-flaky remained enforced. Lead-captured native `pmset` records establish the host sleep intervals; a new run uses process-local `caffeinate -i -w`, without changing host settings, retry policy or timeouts. A passing retry is not a clean full-suite result. The lead's subsequent awake `make e2e CI=1` run passed all 443 cases in 10.2 minutes without flaky results, including the retained real Chrome 200% zoom fixtures. That run precedes the owner-approved Admin CSS slice; the final 447-case inventory still needs independent verification.

The broad audit's first run covered only Motion/Night/Library before a targeted interrupt of its verified owning PID; known 90-second harness locator stalls were retained as failures, not inferred passes. A later static preflight produced 270 route/viewport/state cells, with 262 clean measurements and eight harness failures (including three closed-page errors). A 21-cell rerun produced 19 clean measurements and two different harness failures (including one closed-page error). Independent review found at least one clean measurement for each original cell, but the harness is report-only, screenshots following the Admin arrow walk show System rather than their initial tab, and the public detail fixture omitted a public endpoint. These artifacts are not a complete six-dimension acceptance report. A claimed preflight Markdown report was absent; actual JSON/log evidence, not that claim, was reviewed.

The preflight found selected Admin tabs off-screen on initial load at 320/375px and an Imports file input extending to x=338 at 320px. The owner explicitly chose on 2026-10-05 to fix both rather than defer them. That narrow slice adds initial/focused/keyboard-selected tab bounds, native long-file-name selection and containing-card/input bounds without sending writes. The lead ran two genuine red baselines: four selected-tab failures, then two file-input failures at 320px after fixing only the tabs (375px passed). The two CSS fixes passed all four new cases repeated three times (12), including input/form bounds after long-name selection, zero writes, and unchanged native upload controls. The selected tabs wrap rather than becoming an off-screen scroller; existing heights and roving keyboard navigation are unchanged. Final combined gates, manifest and independent runtime verification remain pending.

The lead reports three actual post-review arm64 images built locally, with IDs server `c9c3ba73327bc9ced9c562d95a95c1ffaf7c01bdef298dc7377572e2d2a67ed8`, private web `3dcb12c588bc1112d5ba5ab090a2fe73cf0e87115cfaca744339db437883dbd5`, public web `c2534baa3236bb39ba395719e47a932f163a8158c6074b2ea5d50ab830dcfdb5`. Each still carries the explicit dirty-source label. No image was published/imported; these precede the Admin slice and are not final clean-commit provenance.

Asobi coordination writes failed twice with backend-unavailable errors on 2026-10-05. No TLS checks were bypassed and no task completion was claimed. This owning record preserves the checkpoint; live graph observations must be reconciled when the selected remote graph is available.

## Owner review handoff — 2026-10-05

The owner explicitly requested pushing PR #125, taking final review personally and consolidating all unfinished work in one issue. Existing [#116](https://github.com/azusachino/iroha/issues/116) is that handoff. No merge, release, publication, deployment or live-data operation was requested or performed.

A fresh independent Sonnet 5.5 verifier, launched with native medium effort, checked the complete 76-file working-diff freeze before and after every gate. The freeze aggregate was `6d009ae8d4cc32658c7f9b83a1d7b013b25a709ec3e56b91b66f8319cfd23b78` at base HEAD `2b2a52e231d420cd5f417c1d31b5e6a2258dbe15`; subsequent changes are this handoff documentation, not production code.

- Independent `make validate` and `make fmt-docs-check`: exit 0; 147 Python and 228 Vitest tests, existing floors, private/public Svelte zero errors/warnings.
- Independent `make e2e CI=1`: exit 0, 447 passed, no flaky tests (10.4 minutes).
- Exact-CI PostGIS in an owned disposable `iroha_test*` database: migrations and `make test-integration-ci` exit 0. A separate fresh-container verbose run over all six workspace modules recorded 527 PASS, zero FAIL, two SKIP (only existing opt-in live AniList smokes). `TestIntegrationPublicProjectionIsAnonymousCachedAndFresh` passed with the new revision/old-ETag/new-304 assertions. Owned containers and volumes were removed.
- The first integration attempt's exit 2 remains failure evidence: a Unix-socket health check marked the temporary initialization postmaster ready before final TCP service. Explicit TCP readiness (`pg_isready -h 127.0.0.1` and an actual healthy wait) corrected the fixture. No product assertion, migration or data guard was bypassed.

The fresh rendered auditor (Sonnet 5.5, `CLAUDE_EFFORT=medium`) was halted on the owner's instruction. Its 147-scenario compact preflight reached 294 cells with zero harness/page errors; a marker rerun left 12 unclassified marker misses. The full 1,491-cell matrix was interrupted after 195 passing cells, exit 130. No final score or audit PASS was given. Servers and the auditor's own worker were stopped; the freeze was unchanged and the index unstaged. Partial report-only measurements and screenshots are not completed semantic acceptance.

Remaining candidates include Intake failed-read named recovery, System/Patterns empty or partial-day markers, unknown public-ID 404 notice and Motion/Night focused stops in exact-table overflow (not confirmed defects). A Library detail subresource failure scenario has no matching subresource and needs an explicit N/A decision. Legacy redirect checks, remaining viewports/states, six-dimension grading, clean final-commit consumer image identity, submitted-head CI and graph reconciliation are carried by #116. The operational deferrals above remain individually approved and unverified. The missing historical audit retains its original ownership.

## Final acceptance requirements

Record exact branch/commit and working-tree identity, actual tool versions, expanded rendered matrix and omissions, six-dimension findings, fresh eligible verifier identity/model/effort, commands/exit codes, independent verdicts, actual image identities, owning disposable integration/public-serving results, and CI for the final PR head. Do not infer acceptance from the existence of this document or the previous Reports check run.
