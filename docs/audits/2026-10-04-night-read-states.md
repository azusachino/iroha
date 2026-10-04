# Night trustworthy read states

Issue: [#121](https://github.com/azusachino/iroha/issues/121), Asobi Night task 2, slice 1b of the [staged plan](../plans/2026-10-03-cockpit-ux-improvements.md).

## Scope and contracts

This slice covers the registered Grapher Night route, its host adapter, shared sleep view contract and composition. It follows the merged Motion reference without copying its different summary API contract. No service DTO, calendar aggregation, dependency, coverage floor or browser policy changes.

Sleep year, month and lifetime aggregates are separate all-history inventories; each successful inventory can project a selected period immediately. They are not newly scoped network reads on every period change. Session records have a separate observed date scope, response and cursor. Retained rows keep their observed label while the requested filter is pending or failed. A records failure does not discard canonical totals; an aggregate failure does not discard sessions or another aggregate inventory. A failed bounds read does not change the requested period.

Missing buckets after a successful aggregate read establish zero counts. Pending or failed aggregate reads do not. Main-sleep averages are undefined when `main_sleep_count` is zero, including nap-only history: display an em dash, not the API's coalesced zero. Actual nap duration and counts remain available. Chart labels consistently identify the main-sleep average.

Dependencies have individually named retries. Recovery uses the existing opt-in RetryNotice focus behavior and actual rendered targets. A newer focused control remains focused. Paging status belongs to its observed page; a pending append for an old month cannot block the new month's paging, and stale records/appends cannot replace the current snapshot.

## Evidence and verification

Synthetic fixtures mirror the current handler/service contracts: date/year/month/lifetime selection, timezone pass-through, limit and opaque cursor, aggregate granularity and buckets, main-versus-nap counts/means and empty bounds. Unknown API endpoints fail closed. Handler/service source review is separate from synthetic browser behavior; these tests are not live API integration.

The initial four red tests reproduced unavailable reads rendered as 0% and loss of independently available year totals, in both themes. Early probes also failed because test expectations used human-readable duration/month formatting instead of the route's canonical formatter, and because a nap-count assertion omitted DOM whitespace. These were corrected to actual representations, not by weakening application assertions.

Fresh source review identified undefined no-main averages, old-scope paging liveness, a main-average label mismatch and weak pagination assertions. Corrections retain API values internally, clarify display meaning, track the pending observed page and exercise a truly paginated retained snapshot and a new cursor request while the old append remains held.

Fresh independent verifier: Sonnet 5.5; lead-recorded native `--effort medium` and visible medium-effort status; session `c353b25c-ac77-4bef-abe4-7bd72736393b`. Reviewed base `16104de1034a795c3e8c14fda869137b17fdf572`, branch `fix/night-trustworthy-states`. Seven source/test paths were frozen throughout independent runtime checks; staged binary diff SHA256 `7d9c6589704d3f91969fd7996b42a593117c483ddab77e338d1a6316f3c0d245` remained unchanged. All seven bounded criteria were independently MET: honest initial/empty states, independent reads, observed scope and supersession, retries/focus, request semantics, scoped paging and unchanged boundaries/floors.

| Independent command | Result |
| --- | --- |
| `make validate` | Exit 0; 228 Vitest; zero Svelte errors/warnings |
| `make e2e ARGS='e2e/night-states.spec.ts --workers=1'` | Exit 0; 24 passed |
| Night stale append, superseded read and failed period changes, repeated three times | Exit 0; 18 passed |
| `make e2e ARGS='--workers=2'` | Exit 0; 261 passed, none failed/flaky/skipped |
| `make fmt-docs-check` | Exit 0; rumdl and Prettier passed |

Lead validation, 24 targeted cases, 18 repeated cases and the complete 261-case browser run also passed. The lead mistakenly requested nonexistent `make check-docs` (exit 2); the verifier preserved that failure and ran the actual owning `make fmt-docs-check`. The lead confirmed the correct target. During runtime the lead stopped only a completed-validation polling `sleep 540`, not a gate or server; the verifier subsequently used native completion. Exact submitted-head CI remains a separate PR acceptance gate.

Four synthetic captures from the verifier's targeted run are persisted in [the evidence directory](evidence/2026-10-04-night-read-states/): unavailable at 320 and populated year scope at 1280, in light and dark themes. The lead inspected all four. The verifier inspected unavailable-light and populated-dark; the other two were not independently image-reviewed. These rendered inspections show em dashes for unavailable totals and canonical populated counts/averages, not broader accessibility certification. The full browser run emitted two worker-load console messages; the verifier did not trace their attribution, so this record makes no claim that every console message was explained.

## Limits and remaining ownership

This is bounded read-state parity, not full #116 cockpit acceptance or completion of #121. Library is next. Later recovery work still owns pagination failure feedback and detail-route parity. The inactive unregistered legacy non-theme Night branch is not certified by this slice and retains historical fallback behavior.

Tests do not establish timezone-boundary/DST aggregation, future-scope 400 behavior, encoded cursor validation, screen-reader conformance, live-data recovery or comprehensive mobile reachability. They establish synthetic query pass-through and the tested rendered states. RetryNotice removes a retry button while its callback runs; focus can briefly return to body before recovery, unless the user chooses another target. No deployment, database migration, recovery exercise, release or edge-policy action was performed.
