# Library trustworthy read states

Issue: [#121](https://github.com/azusachino/iroha/issues/121), Asobi Library task 3, slice 1c of the [staged plan](../plans/2026-10-03-cockpit-ux-improvements.md).

## Scope and contracts

This slice covers the registered Grapher Library route, its host/state adapter and shared media contract/composition. It reuses the existing async resource and retry primitives. There is no service DTO, dependency, coverage floor or browser policy change.

Library records and aggregates now have independent scoped successful snapshots. A failed list does not discard available totals; failed aggregates do not discard records. Before successful evidence, title/completion/active counts show em dashes, and distributions say unavailable rather than successful-empty. A successful empty response establishes genuine zero counts and empty history. A genuine rated zero remains `0.0`; no rating evidence remains an em dash.

The actual Library API differs from Night's all-history inventories: aggregates are filtered by family, status and completed year on each request. Retained totals and records carry independently observed scopes while controls show requested filters. Cursor, rows and paging state belong to the observed list page; stale filter responses and appends cannot replace or append into a newer snapshot. An old pending append cannot block a newer eligible page.

Active count is another contract distinction: the list's count facet intentionally ignores status, retains family/completed-year filters, and excludes paused hidden-from-continue entries. Its caption now explicitly says counts across statuses with that observed facet. Historical completion uses the latest dated progress completion or finished/completed event, not current status. No date, month or timezone parameter was added to this API.

Each read has a named keyboard retry. The existing opt-in RetryNotice recovery focuses the corresponding region only if the user has not chosen another target. Completed-year options are not invented while aggregate evidence is unavailable.

## Reproduction and review

The first fixture attempt omitted the required authenticated session argument; it was invalid reproduction evidence and caused a type error. Early checks also found nullable legacy host accesses and redundant region roles. All were corrected without suppressions or weakened assertions.

Four valid baseline red cases, using the corrected auth fixture, reproduced fabricated zero titles and loss of independently available aggregate totals in both themes. The lead temporarily reversed only its own four-file implementation patch, ran the baseline cases, and restored that patch through an exit trap. No shared work was stashed or reset.

Fresh source review identified a misleading active-count caption that included the selected status despite status-insensitive backend counts. The fix separates observed count-facet scope from record scope. Fixture review also strengthened the complete family mapping, paused buckets/exclusion and historical maximum completion dates. Movie/manga, paused, genuine rated zero, completion-year/status/lifetime, independent failed refresh, keyboard focus and stale append/read journeys are persistent regressions. Unknown fixture endpoints fail closed.

## Independent verification

Fresh verifier: Sonnet 5.5, session `00530e35-87eb-4969-8c39-9703cc3149c3`; native launch `--model sonnet --effort medium`, recorded by the lead. Branch `fix/library-trustworthy-states`, reviewed base `2225925b2596724824f0e951a54dcf954d798859`. Six source/test paths stayed frozen during independent gates. Their staged binary diff SHA256 remained `d30a0d1f38869fe1c98d4d9b6071e68e62dbfc75a6fe44fc784902ac6458bf87` before and after runtime checks.

All seven bounded criteria were independently MET: honest first/empty states, independent reads/exact values, observed scope/supersession, retries/focus, actual request contracts, scoped paging and unchanged boundaries/floors.

| Independent command | Result |
| --- | --- |
| `make validate` | Exit 0; 228 Vitest; zero Svelte errors/warnings; builds passed |
| `make e2e CI=1 ARGS='e2e/library-states.spec.ts --workers=1'` | Exit 0; 28 passed |
| Same spec, stale filter/old append/failed family cases repeated three times | Exit 0; 18 passed |
| `make e2e CI=1 ARGS='--workers=2'` | Exit 0; 289 passed, none failed/flaky/skipped |
| `make fmt-docs-check` | Exit 0; owning docs checks passed |

`CI=1` retained the two-retry, fail-on-flaky and forbid-only policy. Lead validation, 28 focused cases and the complete 289-case browser suite also passed. Exact submitted-head CI remains a separate PR acceptance gate.

Six synthetic images are persisted in [the evidence directory](evidence/2026-10-04-library-read-states/). Four main captures come from the verifier's targeted run: unavailable at 320 and populated at 1280 in both modes. The lead inspected all four; the verifier inspected unavailable-light and populated-dark, not the other two. The populated fixture deliberately has a genuine zero rating. Two additional persisted images show totals focus at light 320 and records focus at dark 1280 after successful keyboard retries; both lead and verifier inspected these.

The verifier's supplementary two-case scratch browser probe confirmed actual region focus and visible two-pixel outlines. The records outline surrounds the whole list and is visually heavy, but remains useful and was accepted as a nonblocking cosmetic limit. This probe supplements the persistent keyboard assertions; it is not a new full-route accessibility gate. The verifier made no attribution claim about console or worker failure messages.

## Limits and remaining ownership

This does not close #121 or the wider #116 acceptance scopes. Reports selected/observed identity is next. Distribution labels/exact categorical tables remain the later Library slice 3b; visible pagination failure and detail-route recovery remain slice 3a.

The inactive unregistered non-theme Library branch is not certified. Raw API scope labels remain; current-completed versus historical-completed label refinement is deferred. Completed-year options require successful aggregate evidence and are not fabricated during failure. Retry buttons can disappear while their callback runs before useful focus is restored.

Synthetic fixtures are a fidelity aid, not live SQL or backend integration certification. Parser-limit behavior, real encoded cursor validation, timezone/DST year boundaries, screen readers, broad mobile reachability and live-data recovery are not established. No deployment, database migration, recovery exercise, release or edge-policy action was performed.
