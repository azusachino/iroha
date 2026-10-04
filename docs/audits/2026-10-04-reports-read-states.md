# Reports trustworthy read states and observed month

Issue: [#121](https://github.com/azusachino/iroha/issues/121), Asobi Reports task 4, slice 2a / F9 of the [staged plan](../plans/2026-10-03-cockpit-ux-improvements.md).

## Scope and contract

This slice covers the registered Grapher Reports host, shared report props/composition and synthetic browser regressions. No service DTO, dependency, shared async/retry primitive, coverage floor or browser policy changed.

Reports is legitimately month-only. Source review of `apps/iroha-server/pkg/httpapi/reports.go` and `pkg/reports/monthly.go` establishes the default twelve-month series: sparse observed `reports`, separate `empty_months` and a successful current report even when that month has no canonical records. Calendar closure, observed records and collection completeness remain separate. This is backend source evidence, not live SQL certification.

Selected month and URL now identify the request. Report headings, metric periods, receipt and comparison range identify the retained successful envelope. Loaded August → selected September while pending → September 503 keeps August values and identities; retry adopts September only after success. The shared composition derives its month from `report.period.month` and no longer accepts a competing month prop.

The host-created empty/loading report envelope is removed. Before a successful report there is no report composition or comparison claiming empty evidence. A successful empty report establishes genuine empty sections and remains usable across quiet refreshes. Existing async-resource guards suppress superseded successes and errors without clearing successful evidence or re-triggering initial inert loading.

Report and date-range reads have independent named retries. A failed range does not silently clamp selection or claim empty history. Without range evidence, selected-year calendar month choices stay usable and are explicitly not a record-availability inventory. Successful range recovery retains the original clamping behavior; a genuine clamp issues a new monthly report read. Incoming year or `scope=lifetime` URLs normalize to the monthly fallback, preserving unrelated query parameters.

Recovery targets actual observed-report and period-controls regions. Retry notices live inside their targets, allowing the existing shared retry primitive to restore the fresh named button after repeated failure. Successful recovery focuses the corresponding region only if the user has not moved focus elsewhere.

## Reproduction and resolved findings

Four valid baseline light/dark pending and failed September cases established August first, then reproduced August data labeled September. All four passed after the identity fix. Persistent assertions now pin selection, URL, observed month, heading, received dates, comparison window and exact 8 km evidence; successful September recovery establishes 9 km.

Early checks caught nullable date-bound accesses, unused error CSS and removal of a still-required `MonthlyReport` type import. These were corrected without suppressions. Fresh source review identified blank selectors during range failure and incomplete repeated-failure focus coverage. Usable calendar choices, truthful pre-read copy, retry containment and stronger stale-success/stale-503, keyboard, dependency and clamp journeys resolved those findings.

The first independent runtime pass passed validation but stopped with 28 passing / 2 failing focused tests. The tests incorrectly used `lifetime=true` as a browser scope and expected its removal. The actual shared browser contract is `scope=lifetime`; unrelated query fields are preserved. Correcting that input and assertion retained all thirty cases and did not change application, helper or HTTP behavior. The earlier failures remain recorded; no assertions or gate policies were dropped to obtain green.

## Independent verification

Fresh verifier: Sonnet 5.5, session `bcc917e7-0c09-42db-8789-ea460d6f3b2e`, Herdr peer `iroha-reports-verify`. Native launch used `--model sonnet --effort medium`; both visible medium effort and usable Sonnet access were recorded. Reviewed branch: `fix/reports-trustworthy-scope`; base: `4b198571a7bd619c9f31452e4fa68cd522668d0f`.

Five source/test paths remained frozen during the accepted recheck. SHA256 of sorted relative-path/NUL/content/NUL entries was unchanged before and after gates: `653d4619360814b4d9c1589a901e56387b86e5bfd69f32c4b779e9dbb0b57a84`. The audit and promoted images were added afterward and require their own prose/docs check.

All eight bounded criteria were independently MET: honest first/empty states; requested/observed identity and exact retained data; month-only query/calendar contract; supersession and sticky readiness; dependency-specific recovery; keyboard/focus; unchanged gates/floors; bounded provenance.

| Independent command | Result |
| --- | --- |
| `make validate` | Exit 0; 228 Vitest; zero Svelte typecheck errors/warnings; builds passed |
| `CI=1 make e2e ARGS='e2e/reports-states.spec.ts --workers=2'` | Exit 0; 30 passed |
| Same spec, pending/stale/superseded/empty/focus/repeated-failure cases repeated three times | Exit 0; 36 passed |
| `CI=1 make e2e ARGS='--workers=2'` | Exit 0; 319 passed, none skipped/flaky |
| `make fmt-docs-check` | Exit 0 before this audit was added |

Web coverage measured all app/shared sources and remained above unchanged floors: 13.96% statements, 15.70% branches, 12.04% functions and 14.04% lines. `CI=1` retained two retries, fail-on-flaky and forbid-only. Vitest transformation printed existing MonthNavigator accessibility/reactivity warnings; no attribution beyond the unchanged source or suppression is claimed. Exact submitted-head CI remains a separate PR acceptance gate.

A fixed `sleep 580` was unnecessary status polling while separate gates ran. The lead inspected its exact PID/parent and stopped only that sleeper, not any gate. The verifier then completed the pending gates and sent its completion directly through Herdr. Scoped local checks remain for iteration; cached success does not replace fresh acceptance or CI.

## Rendered evidence and limits

Five synthetic images are persisted in [the evidence directory](evidence/2026-10-04-reports-read-states/). Both lead and verifier viewed all five. Four focused captures show unavailable at 320 and populated at 1280 in both modes. Unavailable views show usable period controls, a visible report-retry focus ring and no fabricated empty report. Populated views show September identity, received calendar dates, comparison window and exact 9 km receipt evidence. The observed-region focus outline is visually prominent but nonblocking. Raw API-path failure copy remains a later recovery-copy concern.

The initial full-page populated images show a blank Movement trend. A separate bounded synthetic probe, without tracked-source changes, confirms a settled 8 km → 9 km line in `comparison-viewport.png`. Its all-null Sleep/Media charts correctly have no observed values. No persistent data loss was reproduced; the exact mechanism of the earlier capture artifact was not established by re-capturing those tests. A full-suite pass alone was not used to explain the image. The probe's task-owned listener on port 5191 was stopped by inspected PID after a process-pattern cleanup attempt was denied; no pattern kill executed.

This does not close #121 or the separate #116 acceptance scope. Source review, synthetic runtime checks and rendered inspection are distinct evidence. Real SQL/parser behavior, DST boundaries, every exact-data/null case, screen readers, other browsers, real mobile devices and full cockpit recoverability are not certified. Existing no-main-sleep reporting semantics and broader exact-data/recovery work remain outside this slice. No deployment, database migration, recovery exercise, release or edge-policy action was performed.
