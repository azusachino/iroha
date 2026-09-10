# Automatic sync, data coverage, and late-arriving history

Status: proposed; source review complete, implementation and device validation pending.

Reviewed on 2026-09-10 JST against `main` at `8f798b0e9d3876897293204b23d63ed855c37421`. This document records a source review and an ordered implementation proposal. It does not establish the state
of deployed schedules, imported personal data, or an iPhone Shortcut. No existing ADR is superseded by this proposal.

## User story and outcome

The MVP is complete. The next goal is to reduce recurring work: manual Apple Health export/upload, remembering media syncs, and investigating empty or mismatched views. Expenses may legitimately
arrive in monthly imports.

> I keep using my normal apps. Iroha collects and reconciles their data automatically, lets me understand my recent history, and explains missing information without requiring me to operate its jobs.

Daily collection is a freshness target, not a precondition for retaining a day. A missed run must be recoverable. A transaction received in September but incurred in August belongs in August. A day
without evidence must not become a fabricated zero.

Success means fewer user interventions with trustworthy historical results. A two-week ordinary-use trial should measure manual interventions, time spent, source delays, unexplained gaps, and
duplicate or misdated records. Daily Health capture remains conditional on the selected device mechanism and its demonstrated capabilities.

## What the current system actually does

Source paths below are relative to the repository root; line references are for the reviewed revision.

| Surface or subsystem | Verified implementation                                                                                                                                                                       | Consequence                                                                                                                            |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Today                | `apps/iroha-server/pkg/httpapi/briefing_contributors.go:17-28` registers daily health, sleep, activities, and media. Sections use record counts for ready/empty at lines 113-126 and 149-161. | Today is not a universal daily aggregate. Expenses are not a contributor. Empty says nothing about collection success.                 |
| Today calendar       | `apps/iroha-web/src/routes/today-state.svelte.ts:159-185,250-256` derives data presence from sections but obtains available dates from `getDailyDates()`.                                     | The calendar/latest-day helper is health-oriented, not a cross-domain coverage index.                                                  |
| Overview             | `apps/iroha-web/src/routes/overview/+page.svelte:38-112` independently loads activity overview/routes, recent sleep, and media aggregates.                                                    | These are separate projections and time scopes, not a single synchronized daily row. Preserve independent rendering.                   |
| Media day coverage   | `apps/iroha-server/pkg/httpapi/briefing_contributors.go:138-147` contains only date and timezone.                                                                                             | Existing `coverage` is a scope descriptor, not ingestion completeness.                                                                 |
| Report completeness  | `apps/iroha-server/pkg/reports/monthly.go:175-181` marks a month complete when its calendar period ends.                                                                                      | A closed month can still be missing an expense statement or Health data. Calendar completion and data completion need distinct labels. |
| Aggregation design   | [ADR 0004](../adr/0004-cache-correctness-and-report-reads.md) retains canonical queries plus disposable, invalidated response caches.                                                         | Keep this design. A job that aggregates only yesterday would miss historical corrections.                                              |
| Media time semantics | [ADR 0005](../adr/0005-media-provider-time-semantics.md) separates library state, state history, and dated consumption.                                                                       | Automatic library refresh must not invent watching/reading on the sync date.                                                           |

## Review findings and priority

### P0: Import failure can appear as successful durable work

`apps/iroha-imports/service.go:423-429` writes failed import status but returns only the status-update error. When that update succeeds, the function returns nil. Processing error branches call this
helper, including lines 251-261 and 294-320. `apps/iroha-job/main.go:229-238` forwards the result directly; `apps/iroha-runtime/jobs/service.go:286-294` completes the durable job when the handler
returns nil.

This is a source-confirmed error-propagation defect: a failed import can produce a completed queue job and bypass its retry path. It is not a runtime reproduction of the user's missing data.

Suggested fix: preserve the processing error after recording failure, retain any status-write failure, and ensure retries reset attempt-specific import timestamps/state. Use the existing bounded retry
policy first; add terminal-error classification only for demonstrated cases. Test malformed evidence and a transient persistence failure through the real job handler, not only the status helper.

### P0 adoption gate: Partial Health input must not enter full-snapshot reconciliation

`apps/iroha-imports/activity_import.go:71-88,116-135` selects complete Apple reconciliation by source kind and removes prior Apple items absent from the snapshot, including derived activity, sleep,
and daily records.

This is appropriate for the existing complete-export contract. An automatic daily/window payload mislabeled as that source could remove older history. A distinct incremental adapter or explicitly
validated ingestion mode is mandatory before connecting a Shortcut or another exporter.

### P1: Fetch success and usable data are different checkpoints

`apps/iroha-imports/sync.go:72-90` stores evidence, creates imports, advances the fetch cursor, and completes fetching before child imports execute. Import creation and queue insertion are
transactional in `apps/iroha-imports/service.go:130-137`.

Advancing the fetch cursor after durable capture is valid; moving it backwards until every parser finishes would conflate two stages. The missing contract is the relationship from a sync run to its
imports and their materialization outcome. Failed imports must remain visible and replayable from stored evidence independently of the provider cursor. Never synchronously block a sync worker waiting
for child jobs in the same queue.

### P1: Media automation needs defaults and connector overlap protection

`apps/iroha-job/main.go:130-145` registers media handlers but ensures only a public-export schedule. `apps/iroha-server/pkg/httpapi/media_sync.go:48-51` explicitly describes bridge refresh as manual.
No default media schedule was found in the reviewed initialization paths; deployed database schedules were not inspected.

The scheduler already transactionally enqueues due jobs (`apps/iroha-runtime/jobs/service.go:388-429`) and supports interval/manual schedules (`:435-447`). Retries, Retry-After, and expired-lease
recovery already exist (`:240-264,451-488`). Reuse them. Add per-connector exclusion/coalescing: `apps/iroha-imports/sync.go:46-55,119-130` accesses shared cursor state without a connector-level
claim, so manual and scheduled jobs must not race on one connector/account.

### P1: Existing coverage measures observations, not collection completeness

`apps/iroha-server/pkg/metrics/series.go` exposes expected/observed periods; metric definitions in `pkg/metrics/registry.go` describe observed-day/period coverage. Those are useful measures of
recorded data, but no record on a date does not prove ingestion failed or succeeded. Conversely, one sample cannot prove a whole day is covered.

Keep observation coverage and add collection coverage separately. Do not rename the existing denominator into a stronger guarantee. An expected metric also needs an applicability policy: body mass or
VO2 max should not trigger a missing-data alert every day simply because no new measurement exists.

### P1: Late expenses already reach historical queries; statement completeness is missing

`apps/iroha-server/pkg/expenses/service.go` filters expenses by `occurred_on`; HTTP mutation wiring and the cache invalidation coordinator invalidate expense, metric, and report reads. Preserve that
path for future imports rather than writing directly to tables without invalidation.

The current expense schema (`apps/iroha-server/db/migrations/00007_expenses.sql`) has source identity but no statement/account coverage declaration. An identical source replay is idempotent, while
changed content conflicts (`pkg/expenses/service.go:415-423`). A monthly importer therefore needs an explicit correction policy and statement scope. The positive-amount constraint also means arbitrary
bank statements with refunds/credits are not automatically supported.

### P2: UI composition needs source status and a refresh lifecycle

Today uses local request counters (`apps/iroha-web/src/routes/today-state.svelte.ts:37-45,234-246`) despite the current AGENTS requirement to use `createAsyncResource`. Change that state management as
part of the scoped Today work, retaining the previous response's date while a new date loads.

Overview's activity load does not establish sleep/media readiness; its archive record count sums available values with zero fallbacks (`apps/iroha-web/src/routes/overview/+page.svelte:74-79`). Audit
the rendered theme contracts so a supplemental failure or unknown source is not presented as a complete archive total. Today loads on selected-day changes; Overview loads on mount. New server data
alone does not update an already open page.

## Proposed contracts

### Separate cadence, resolution, and coverage

| Axis                 | Examples                                                                     | Rule                                                                                   |
| -------------------- | ---------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Cadence              | Daily Health opportunity; periodic media poll; monthly statement             | Controls expected arrival and recovery, not event dates.                               |
| Resolution           | Timestamped sample; workout; sleep interval; day summary; month-only fact    | Preserve source precision. Never synthesize individual events from aggregates.         |
| Temporal membership  | Workout start; sleep wake date; daily calendar date; expense occurrence date | Use existing domain rules and explicit timezone. Received time is separate.            |
| Collection coverage  | Source/account/category interval successfully ingested                       | Describes retrieval evidence as of a run, not certainty about all real-world behavior. |
| Observation coverage | Days or buckets with actual measurements/events                              | Remains available even if collection completeness is unknown.                          |

Keep canonical reducers authoritative. Repeated daily totals replace the same source/day value; they are not added together. Preserve `(metric, unit)` grouping, source precedence, metric-specific
weighting, and currency separation. A partial Shortcut export must not override a richer trusted source merely because it arrived later.

### Minimal persistence proposal

Reuse raw evidence, import jobs, durable jobs, and connector state. Introduce only the missing durable associations and coverage assertions:

1. A sync-run/import association, with a uniqueness constraint for each pair. Reuse an existing run ID if the runtime model can supply it; do not create a second job lifecycle.
2. A coverage assertion for a source instance/account, category or metric set, half-open interval, timezone, evidence/import reference, ingestion mode, completeness assertion, and recorded time. A
   statement is one interval assertion, not 31 fabricated expense rows.
3. An explicit policy for enabled sources: supported categories, expected delivery cadence, and the scope in which completeness can be asserted. Prefer existing connector configuration where possible.

Derive unknown calendar days at read time. Persist evidence-backed assertions, not a placeholder row for every source/type/day. Resolve overlapping assertions by source scope and deterministic
replacement order; an older arriving run must not regress newer coverage. A scalar `covered_through` is only valid for a contiguous covered interval; retain holes internally and expose them for
queried periods.

Write canonical changes and their coverage assertion atomically. A fetch receipt may precede them, but may not claim successful ingestion. A fully exhausted, explicitly scoped empty result can assert
an empty collection interval only when the adapter can distinguish it from missing access or an incomplete query. Otherwise keep completeness unknown.

Historical rows establish observed data, not historical completeness. Do not backfill a complete status merely from min/max dates or the existence of records. Revalidate historical coverage from
trustworthy export metadata or leave it unknown.

### Independent status dimensions

Avoid one enum that mixes transport, applicability, and data presence:

- Availability: supported, unsupported, or disabled.
- Collection: unknown, partial, or covered as of specified evidence; covered-empty is covered with an explicit empty assertion.
- Operation: fetching, importing, idle, or failed; retain stage timestamps and a safe error reason.
- Freshness: within expected cadence, overdue, or not scheduled; compute relative to source policy and the current time.

Keep successful coverage when a subsequent attempt fails, while separately showing that failure and any overdue state. No “complete life” indicator should be inferred by summing source states. Use a
compact derived display state in the UI while retaining the underlying distinctions in the API.

### API and display suggestions

Add a typed source-status read contract, with period-scoped coverage queries. Proposed fields include source instance, category, requested range/timezone, fetched-at, ingested-at, covered
intervals/gaps, evidence basis, cadence, and next action. Final route/schema names should be settled in the contract slice and registered in OpenAPI before client wiring.

Expose source metadata alongside existing section values in briefing, metric series, and report responses. Preserve current ready/empty/unavailable content states during transition. The monthly-series
`completeness` field must be explicitly identified as calendar completeness; add collection completeness independently and update labels/fixtures rather than silently changing the old field's meaning.

Do not cache a time-relative “fresh” label for the full existing response TTL. Prefer stable timestamps/policy in the response and derive staleness at display time, or bound status-cache lifetime by
the next freshness transition. Coverage-only writes must invalidate relevant briefing/metric/report/status reads even when no canonical measurement changed.

Today should retain its selected day and offer an explicit jump to latest recorded data. A calendar date remains selectable with no observations. A new cross-domain date index, if needed, must
distinguish observed dates from collection coverage. Adding an expense Today section is optional product scope, not necessary for fixing historical expense correctness.

Overview should label each tile's actual scope, show supplemental failures independently, and avoid implying all domains share a single data cutoff. Shared source-status visuals belong in
`packages/iroha-shared`; app hosts own API loading and navigation. Verify all six theme compositions and their fallback rendering.

Refresh on window focus and successful relevant local mutations; use modest, visibility-aware polling only while ingestion is pending or the page needs freshness. Preserve mounted content during
refresh. Live subscriptions are not required for this first implementation.

## Domain-specific behavior

### Daily Health with missed runs and corrections

- Begin with a configurable seven-day overlap as a trial default, not a completeness guarantee. Recover longer outages from the last durable ingestion coverage and explicit missing intervals.
- A Shortcut should send a versioned bounded envelope, selected categories, interval/timezone, and structured values/source identifiers when available. Its receiver must authenticate a scoped
  ingestion credential and preserve raw evidence.
- If the client cannot expose stable sample identities, do not claim exact incremental sample synchronization. Evaluate bounded aggregate replacement for supported metrics, with explicit reduction
  semantics and source precedence.
- No record in a partial batch implies deletion. Support explicit tombstones or an adapter-validated complete replacement window; retain the legacy full-export contract separately.
- Sleep crossing midnight needs enough overlap/context to reconstruct the same session and wake-date attribution. A day boundary must not split one night into duplicate sessions.
- Health permissions and empty query results can be ambiguous. If the mechanism cannot prove a complete read, show received data with unknown/partial coverage.

The [iBeats pattern](https://github.com/yihong0618/iBeats) is a useful Shortcut collection prototype, not evidence of full Health coverage. Its shared Shortcut was not retrieved in the earlier
research. [Apple's automation guide](https://support.apple.com/guide/shortcuts/apd602971e63/ios) and
[HealthKit protected-data error](https://developer.apple.com/documentation/healthkit/hkerror/errordatabaseinaccessible) make device validation necessary. Compare
[Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) only against gaps demonstrated by the Shortcut trial. Transport selection does not change the server's
coverage/reconciliation contract.

### Media

Enable one default interval schedule per configured provider/account; preserve explicit opt-out and prevent duplicate schedules after restart. Coalesce overlapping requests and keep a slower
bridge-refresh policy where supported. Missed ticks should produce catch-up work, not an unbounded replay of every clock tick.

Keep library state and dated history separate. A successful Bangumi library snapshot can be covered for current library state while exact daily consumption remains unsupported. Refreshing a provider
snapshot today does not move its content into today's consumption. AniList dated updates retain their actual provider time basis and bounded history limitations.

### Monthly expenses

Example: on September 10, an August statement is imported. Its transactions affect August `occurred_on` queries; receipt metadata records September 10. Invalidate historical expense/metric/report
reads after canonical commit. A replacement moving an expense between months must refresh both periods; current namespace invalidation already covers both.

Record statement/account identity and explicit covered period when known. A complete card statement covers that account, not cash or every other account. Do not infer completeness from its last
transaction date. An empty statement can establish no transactions for its scope; no uploaded statement cannot.

Monthly expected cadence means September can be “awaiting statement” without a daily failure alert. Decide the statement due policy during source setup. Corrections need stable source-scoped
transaction identity, using provider IDs where available and a documented fallback/collision policy otherwise. The update/conflict policy must preserve deliberate user edits. Refunds, credits, and
posted-versus-purchase date semantics require an explicit contract before claiming general bank CSV support.

## Ordered implementation slices

| Order | Deliverable and concrete boundary                                                                                                                                                                                                       | Acceptance                                                                                                                                                                                          |
| ----- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1     | Repair import failure propagation in `apps/iroha-imports/service.go`; add handler/queue regression coverage using `apps/iroha-job` and `apps/iroha-runtime/jobs`.                                                                       | Failed parsing never produces queue success; transient failure reaches retry; replay can succeed without duplicates or stale failure metadata.                                                      |
| 2     | Specify coverage and input-mode contracts; add a forward migration under `apps/iroha-server/db/migrations`, matching runtime models, and focused coverage service/tests. Register typed API additions in `docs/contracts/openapi.yaml`. | Assertions are evidence-backed, scoped, atomic with ingestion, and distinguish unsupported/unknown/empty. Existing records are not automatically labeled complete.                                  |
| 3     | Implement the bounded incremental Health adapter in `apps/iroha-providers` and persistence wiring in `apps/iroha-imports`; preserve complete-export behavior. Build the minimal Shortcut receiver/client contract.                      | Partial input cannot remove older history; overlapping windows/corrections are idempotent; actual supported metrics are verified against device fixtures.                                           |
| 4     | Link fetch runs to child imports in `apps/iroha-imports/sync.go`; enable configured media schedules and connector exclusion in job/runtime wiring.                                                                                      | Fetch and ingestion status remain separate; stalled/failed imports are replayable; concurrent manual/scheduled requests cannot corrupt cursors; restart does not duplicate schedules.               |
| 5     | Add statement scope and correction contract to expenses; route imports through the canonical mutation/invalidation boundary. Add coverage metadata to reports/metric series and invalidate coverage-only changes.                       | August imports arriving in September update August; multi-account scope, replay, correction, deletion, and currency separation are tested. Closed month is not labeled data-complete by date alone. |
| 6     | Update `pkg/httpapi/briefing_contributors.go`, `pkg/reports`, API types, Today state, Overview, and shared view contracts/components. Use `createAsyncResource` and explicit refresh behavior.                                          | Each domain displays value, period, and source status independently; empty/failure/overdue states remain distinct across themes; no fake zeros or invented event dates.                             |
| 7     | Run isolated end-to-end and two-week device acceptance; document supported source capabilities and remaining limits.                                                                                                                    | No routine export/sync clicks for supported inputs; outages recover; discrepancies have an evidence trail; intervention count and latency are measured.                                             |

Slices 3 and 4 can proceed independently after their shared contracts exist. Statement work does not block validating Health capture. Each implementation slice should preserve existing readers until
its contract and client changes are ready; no large storage rewrite is required.

## Verification matrix

| Scenario                                                       | Required result                                                                                                                     |
| -------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Phone misses one day or exceeds the overlap window             | Explicit missing range is recovered; no irreversible gap and no invented zero.                                                      |
| Phone is locked or source permissions are insufficient         | No complete/empty assertion without evidence; last successful data remains visible.                                                 |
| Duplicate or out-of-order batch                                | Stable canonical identity, no doubled totals, no regression to older values/coverage.                                               |
| Partial Health batch after full historical export              | Old activity, sleep, and daily history survives.                                                                                    |
| Late sample, sample correction, or supported deletion          | Correct source/date projection updates; richer source precedence is respected.                                                      |
| Cross-midnight sleep, JST month boundary, DST-capable timezone | Existing domain date rules and half-open windows hold; no duplicate night.                                                          |
| Fetch completes while parsing fails                            | Fetch status can complete, ingestion cannot; raw evidence remains replayable.                                                       |
| Worker crash, HTTP 429, or competing scheduled/manual run      | Existing lease/retry behavior works, cursor writes are serialized, no duplicates.                                                   |
| Zero events with successful full scoped collection             | Covered-empty is possible without creating fake domain records.                                                                     |
| Only one metric arrives for a day                              | That metric's observations are visible; other categories are not marked covered.                                                    |
| August expense statement arrives in September                  | Prime August report/list/series caches first; after import all refresh, and September spending does not increase from receipt time. |
| Identical versus changed statement replay                      | Identical content does not duplicate totals; changed content follows the explicit correction/conflict policy.                       |
| Partial or failed expense statement                            | Coverage does not advance to complete; previous valid history and coverage remain available.                                        |
| One account imported, another missing                          | Account-specific coverage; no global complete badge.                                                                                |
| Expense corrected into another month or deleted                | Both old and new historical views refresh; remaining totals are correct.                                                            |
| Coverage changes but canonical values do not                   | Status/report/briefing metadata refreshes through an explicit invalidation dependency.                                              |
| Closed month with absent statements                            | Calendar period closed, collection incomplete/unknown.                                                                              |
| Long-lived open browser tab                                    | Focus/pending refresh obtains new data without blanking the current view or misdating retained values.                              |

Use `make test`, `make contract-check`, and `make web-test` for relevant implementation changes. Use `make test-integration` only with the intended disposable local database: the target starts
dependencies and applies migrations. Run `make check` before commits and `make validate` before a PR; real-device and live-deployment claims require separate evidence. Documentation-only validation is
formatting, source/link review, and the required repository gate, not proof these proposed behaviors exist.

## Decisions to settle through the first fixtures

- Which Health categories and source fields the actual Shortcut can export reliably, and whether aggregate replacement is sufficient for them.
- The user's actual media providers and acceptable refresh delay; initial polling intervals remain policy choices, not measured guarantees.
- Which expense account/statement format is first, including refund and transaction-date semantics.
- Whether a recent overview is preferable to a strict Today default; keep this independent of ingestion correctness.

Preserve raw evidence, canonical domain tables, provider boundaries, durable jobs, reducers, cache invalidation, and media timestamp safeguards. Defer a new workflow engine, daily aggregate table,
native iPhone app, and broad UI redesign until a demonstrated requirement justifies them.
