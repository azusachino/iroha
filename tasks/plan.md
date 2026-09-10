# Iroha v0.5: reliable collection, trustworthy history, complete cockpit

Status: executable release proposal; implementation not started.

Target: `v0.5.0`, evolving the existing `vendor/iroha` project from `0.4.5`. Source review baseline: `8f798b0e9d3876897293204b23d63ed855c37421`, 2026-09-10 JST. The working tree also contains a
user-owned `.mise.toml` change; it is not part of this plan's documentation change.

This is the single execution plan for v0.5. It selects an in-place rebuild of unsafe internals, retains the mature domain experience, and supersedes the execution alternatives in the three September
10 design papers. Those papers remain supporting evidence:

- [Sync, coverage and late data](../docs/plans/2026-09-10-automatic-sync-coverage-and-late-data.md).
- [Foundational defects and invariants](../docs/plans/2026-09-10-foundations-first-rework.md).
- [Mature-product capability inventory](../docs/plans/2026-09-10-mature-product-clean-rebuild.md).

Tasks are tracked in the shared Asobi graph as `iroha:v0.5`, children `iroha:v0.5:task-N` matching the numbered work items below. This file owns scope, decisions, acceptance and dependencies; Asobi
owns dispatch and execution status. Existing unrelated epics are not closed or resumed by this plan.

## Release decision

Build v0.5 inside Iroha. Reuse provider parsing knowledge, tested reducers, existing domain queries where correct, and shared visual assets. Replace the Apple-specific identity tracker, destructive
raw-file purge, unsafe job transitions, and process-local read-freshness mechanism. Rebuild derived data where that is simpler than preserving flawed storage arrangements.

Do not spend the release on a new repository, a language/ORM migration, splitting or merging every Go module, a workflow engine, or a universal fact database. These changes do not resolve the user's
daily collection and trust problems.

Breaking internal components and coordinated private API changes are allowed. They must be intentional: update the server, worker, shared contracts, web, CLI and other identified clients together.
Keep `/api/v1` as the actively developed contract; use explicit schema/representation changes and a documented client cutover, not a second API surface solely to protect pre-release mistakes.

## What v0.5 delivers

1. Automatic Health intake for the agreed daily metrics, sleep and workout summaries, with historical full exports and rich activity data retained. Routes/samples/laps must remain browsable; any
   difference between daily-transport fidelity and archived fidelity is explicit.
2. Automatic AniList/Bangumi collection and mapping maintenance, with correct library state, dated updates, exact events where supplied, and actionable resolution of identity conflicts.
3. Expenses that work with monthly imports: accounts, a documented normalized statement CSV, refunds, corrections, per-currency totals and historical reports that update correctly.
4. Per-source coverage and delivery status throughout the cockpit. No fake daily measurements, no “complete month” claim based only on the date, and no failed ingestion reported as success.
5. Complete existing exploration: activity details/routes/streams/laps, sleep stages and naps, daily patterns, media detail/history, expenses, metrics, monthly reports and supported period
   comparisons, tasks, supported themes and public archive.
6. Tested replay, migration, backup/restore, worker recovery, cross-process cache correctness, private/public boundaries and everyday operational behavior.

The release is not complete when only the foundation or a small Today screen works. Existing capabilities stay in the acceptance inventory unless an explicit product decision removes one. Conversely,
support for every provider listed in an old roadmap is not a v0.5 commitment. The product must reliably support its selected sources before accumulating new integrations.

## Non-negotiable user outcomes

| Situation                                 | v0.5 behavior                                                                                           |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| User does nothing today                   | Supported collection runs without routine export/upload/sync clicks; delay is handled by source policy. |
| Phone misses several days                 | A later run recovers eligible missing periods; irrecoverable provider limits are explained.             |
| Source sends the same or older data       | No duplicate totals, new canonical identities, or silent rollback of newer selected facts.              |
| One domain is late                        | Other domains remain useful; that domain shows its actual date/coverage.                                |
| August expenses arrive in September       | August spending and reports update; receipt time does not become spending time.                         |
| User resolves a match or corrects a value | The choice is retained and can be inspected/undone; the next sync does not silently discard it.         |
| Worker or cache fails                     | Work remains recoverable and reads do not silently present an old revision as current.                  |
| User opens an old detail link             | Preserve canonical IDs where possible; explicitly map changed IDs during the one-time rebuild.          |

## Chosen architecture and data rules

### 1. Keep the current application structure

Keep `iroha-core`, `iroha-providers`, `iroha-imports`, `iroha-runtime`, server/job/web/public-site apps and `packages/iroha-shared`. Improve boundaries inside them. Use the existing
Go/Postgres/PostGIS/Svelte toolchain and `make` entry points. New abstractions require two actual consumers or an essential correctness boundary.

Provider adapters parse typed observations. Imports own receipt interpretation and canonical publication. Domain services own domain queries and explicit mutation contracts. Runtime owns durable
claims, database revisions, evidence storage and shared IDs. The frontend renders typed facts and owns interaction/loading, not identity or aggregation rules.

### 2. One source identity authority; separate bytes from receipts

Retain `tb_raw_files` as the deduplicated blob store, including compatibility metadata during transition. Add explicit source instances and receipts. A source instance identifies a provider plus
account/device/export stream; a receipt identifies one acquisition, its blob, capture/receipt clocks, declared scope and ingestion mode.

Extend the existing generic observation model as the sole source identity authority. Its business key becomes source-instance + domain/source-kind + source key. Retain interpretation/receipt
membership separately from current selected values, so unchanged observations can be confirmed by a new receipt without losing history. Do not add another parallel current-identity registry.

Remove `tb_apple_source_items` and `purgeDerivedForRawFile` from active write paths after validated conversion. Canonical identity must never depend on `first_raw_file_id`. Keep that field only as
historical provenance if useful; it cannot authorize deleting a canonical object.

Use explicit migrations to add receipt/interpretation links and revision/provenance fields to existing tables. Do not promise a final table count before the ownership constraints are written, but do
not redesign unrelated tables merely for naming consistency.

### 3. Source interpretation precedes canonical resolution

Resolve in this order:

1. Deduplicate by stable source identity within the source instance.
2. Recognize a newer eligible interpretation according to source authority, not worker completion time.
3. Apply explicit cross-source identity links; use conservative candidate matching for ambiguous cases.
4. Select canonical values under metric/domain rules and retained user choices.

Keep activity routes/samples/laps owned by observations and expose the selected view without duplicate canonical child copies once compatibility reads have moved. Each sleep observation belongs to one
canonical sleep session; remove the redundant many-to-many ownership mechanism after migrating preferences. Selected-observation FKs must enforce ownership through suitable composite keys/constraints.

Canonical daily projections use a fixed documented unit per metric. Source observations retain original unit/normalization metadata. Repeated day totals replace the relevant source/day value; they are
not accumulated as extra days. Health source priority and interval-union rules remain domain-specific. Media list state is never an exact consumption session.

### 4. Explicit input modes and replay authority

Support three validated modes: full snapshot, bounded replacement, and incremental changes. Absence permits deletion only within a proven authoritative full/bounded scope. Partial input never invokes
global absence reconciliation. Source-capture/version metadata determines order when trustworthy; an undated/manual old export is importable evidence but cannot automatically supersede current state
by upload time alone.

Serialize publication per source instance. Reprocessing produces a new interpretation and re-resolves affected identities without purging objects supported by other evidence. User selection/correction
precedence survives replay. Stable IDs and tombstones survive retries and parser upgrades.

### 5. Repair and keep the durable queue

Fix error propagation first. Use the existing attempts value, together with job ID and worker ID, as the claim-generation fence if it is incremented on every claim and never reset. Heartbeat,
complete, fail and protected publication must check that claim. A stale handler must not commit simply because cancellation was delayed.

For publication, lock/check the current job claim and source scope inside the canonical transaction, following one documented lock order. Keep critical publication bounded; stage large parsing work
outside it. Recover exhausted leases even when no other job is claimable. Keep existing retry/backoff/Retry-After behavior and test it through real handlers.

Link fetch runs/receipts to interpretation jobs. A fetch cursor advances after durable evidence and downstream work exist. The end-to-end run is usable only after required interpretation commits;
failed interpretation stays replayable independently of that cursor.

### 6. Transactional Postgres revisions govern all cached reads

Choose database revisions, not a new invalidation outbox. Maintain a revision per affected read namespace in Postgres and increment it in the same transaction as canonical/coverage mutations. Use a
consistent sorted lock order for multi-namespace writes. All writers, including imports, expense replacement/deletion and media resolution, use this transaction-aware mutation boundary.

For a cached read, load its dependency revision vector from primary Postgres before cache lookup. Include the vector in both the response-cache key and the singleflight key. Keep Postgres or Valkey
response storage disposable; its invalidation API is no longer the correctness boundary. Obsolete entries expire or are cleaned up normally.

Freshness means a consistent database snapshot acquired during this request, after any write acknowledged before the request began. Concurrent later commits may yield the earlier snapshot; this is not
“latest at response send.” On a cache miss, multi-query report/domain loads share the repeatable-read snapshot used to obtain the revision vector. Never label old data with a revision fetched after
loading. If revision lookup fails, do not serve cached data using remembered revisions.

Refactor the relevant read loaders to accept the transaction/query context; buffer responses until successful assembly. Avoid long read transactions around public build/export work: those operations
create a distinct explicit snapshot artifact. Verify snapshot behavior and costs before expanding this mechanism to every unrelated endpoint.

### 7. Coverage is evidence, not a measurement

Add source/category/interval coverage assertions linked to successfully committed interpretations. Store covered intervals and holes; a latest timestamp does not establish continuity. Derive absent
calendar days on reads instead of inserting empty activities or zero-valued metrics.

Separate capability, collection completeness, processing status and freshness. Preserve the last successful coverage when a later attempt fails. Empty responses with ambiguous permission/query status
remain unknown. Monthly expense policy does not generate daily missed-sync alarms. Body mass and other sporadic metrics do not require a sample every day.

Keep observation counts in metrics. Add collection metadata separately. Rename the UI's calendar-only report completeness label and extend the API without silently redefining the old field. Compute
time-relative stale labels from timestamps/policy, not a day-old cached boolean.

### 8. Choose and prove the daily Health transport early

The first candidate is a built-in Shortcut sending bounded structured data directly to Iroha. A device fixture trial must establish supported fields, stable identities or valid aggregate replacement,
source fidelity, sleep boundaries, payload size and automatic execution behavior before the server adapter is finalized.

Initial release-required daily categories: steps, active energy, exercise/stand summaries where available, resting heart rate/HRV where measured, sleep summary/stages, and workout summaries. Preserve
all currently supported archival Health metrics and detailed activities. Record transport-specific unsupported fields explicitly; adding a generic “Health connected” badge is insufficient.

Trial the Shortcut for seven ordinary days early in the release, with a locked-phone run, one network failure, one deliberately missed day and recovery. If it requires a daily manual ritual or cannot
supply the agreed categories reliably, test Health Auto Export against the same fixtures immediately. If neither meets the target, the Health-client work must be re-estimated, including a focused
native client; do not silently ship v0.5 as manual-only or broaden native-app scope without evidence. Server foundation tasks can continue during device trials.

Routes/samples may have a different demonstrated delivery cadence from daily summaries. This distinction must be accepted and visible; the release cannot claim automated full-fidelity Health history
unless the trial proves it. A manual historical archive path is not evidence of daily automation success.

### 9. Automatic media with honest history

Create one enabled schedule per configured AniList/Bangumi source. Initial policy: hourly sync with rate-limit-aware backoff and request coalescing; bridge refresh weekly. These are defaults to tune
against actual provider limits, not exact-time guarantees. Disabled/unconfigured sources get no jobs.

Current state, provider date facts, observed changes and exact events remain separate. Preserve conflict/matching decisions through sync; ambiguous matches enter the existing attention workflow with
provenance and undo. Do not invent sessions for providers that only expose snapshots.

### 10. Monthly expense import is a first-class workflow

Ship a normalized CSV format rather than an unspecified “all banks” importer. A statement manifest supplies account/source identity, period `[from, to)`, completeness declaration and statement
revision. CSV columns are stable transaction ID, occurrence date, currency, positive minor-unit magnitude, kind (`expense` or `refund`), category, merchant and optional note/original-transaction
reference. A refund may be unlinked; transfers are rejected in this contract rather than counted as spending.

Preserve existing positive expenses as kind `expense`. Expose gross spend, refunds, net spend, purchase count and refund count explicitly; keep currencies separate. Each row uses account + source
transaction ID. Exact replay is idempotent. Corrected statements update source-owned fields unless a retained user override exists; conflict handling must show the difference rather than silently
overwriting the override. Explicit deletion/tombstone policy applies to complete statement revisions, never arbitrary partial CSV absence.

Default date is occurrence date; formats with only posting date identify that basis in account/import metadata. Statement coverage is not inferred from its last row. Manual entries remain supported
and are not deleted when importing a provider statement. Preview validation reports rejected rows before publication; a failed/partial statement cannot be labeled complete.

## Components to remove or replace

| Existing component                                               | v0.5 disposition                                                                                      |
| ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Apple-only source identity tracker                               | Replace with the extended generic source identity/receipt contract, then remove obsolete writes/table |
| Raw-file-based global purge                                      | Delete from reprocessing; use interpretation and canonical re-resolution                              |
| Duplicate canonical and observation measurement storage          | Move reads to selected observation children, then remove redundant copies where verified              |
| Redundant sleep ownership relation                               | Use one observation-to-session owner plus canonical selection; migrate preferences                    |
| Nil-return import failure handling                               | Replace with error-preserving status recording                                                        |
| Unfenced completion/failure and rolled-back empty-queue recovery | Replace with claim-checked transitions and committed recovery                                         |
| Process-local cache degradation as correctness mechanism         | Replace with transactional Postgres revision identity; retain useful cache storage/cleanup            |
| UI-inferred readiness from row counts alone                      | Replace with explicit content + source coverage contracts                                             |
| Today hand-written async counters                                | Replace with the established async resource while preserving dated retained content                   |
| Theme-specific duplicated data calculation                       | Move domain calculation into server/shared typed adapters; preserve theme compositions                |

Internal deletion is permitted. These are planned replacements, not instructions to drop live tables before their successor is verified.

## One maintenance cutover, not a second product

Use disposable database copies to build and test the new schema and replay path. Existing Iroha remains available until the planned maintenance window. There is no permanent dual-write system or
separate replacement application.

External origin does not make every current record recoverable from a future API call. Preserve existing raw archives/provider snapshots. Inventory expenses, manual media events, tasks, user edits,
match decisions, settings and publication choices; export any irreplaceable state into a versioned local migration bundle. If none exists, record that result and keep the rebuild simple.

Rehearse: stop intake and workers, take and verify an aligned DB/blob snapshot, apply forward migrations, replay retained evidence under source authority, restore retained user state/ID mappings, run
reconciliation/privacy/consumer checks, then resume writers. Do not rewrite the original migration to upgrade an existing installation. A fresh database can use the completed migration chain.

Use namespace/table cleanup only after the new projections pass reconciliation. Rollback before reopening writes restores the aligned snapshot. After reopening writes, rollback also requires
exporting/replaying post-cutover user inputs and receipts; an old backup alone is insufficient. Measure replay/cutover duration during rehearsal and publish it before deployment.

## Delivery and task ledger

The numbered rows are bounded work items with a specific verification result. Paths are likely boundaries, not authorization to refactor every file in a directory. Split an item at dispatch if it
exceeds one focused change; retain its acceptance and dependency links. Do not dispatch a dependent task until its prerequisites are verified.

### Phase A: Establish the release contract and executable failures

| ID  | Work item / dependency                                               | Likely files                                                                                             | Acceptance and verification                                                                                                                                                                                                      |
| --- | -------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Capture v0.5 capability, consumer and retained-data inventory / none | `docs/`, existing route inventory, migration tooling                                                     | Every current route/CLI/public workflow has a keep/replace decision; identify external re-fetch limits and user-authored state. Verify inventory against source and a read-only deployment export when available.                |
| 2   | Establish runnable fixtures and baseline / 1                         | `scripts/`, current integration tests, existing Make targets                                             | Reproduce false-success, final-lease recovery, old-snapshot/reprocess and stale-cache boundaries. Run `make test` and focused integration cases in disposable storage; record existing failures rather than approving them.      |
| 3   | Prove daily Health capture / 1                                       | Device Shortcut fixture, `docs/capabilities/providers/apple-health.md`, bounded intake contract fixtures | Seven-day trial plus forced missed/locked/offline scenarios measures fidelity and intervention. Choose transport using the policy above; preserve raw fixture bytes locally. No live partial input through old full-export path. |

Checkpoint A: task 2 can expose failing cases; task 3 is an external dependency, not a reason to pause independent foundation fixes. Reconcile the user-owned toolchain edits and run actual targets
before repeating the previous TLS-blocker claim.

### Phase B: Reliable work and SQL publication contracts

| ID  | Work item / dependency                                      | Likely files                                                            | Acceptance and verification                                                                                                                                                                |
| --- | ----------------------------------------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 4   | Preserve import processing errors / 2                       | `apps/iroha-imports/service.go`, import handler tests                   | Failed parse produces failed/retrying work, never completed queue status. A retry resets attempt-specific status and can succeed. `make test` plus handler integration reproduction.       |
| 5   | Commit exhausted-lease recovery on empty queue / 2          | `apps/iroha-runtime/jobs/service.go`, jobs tests                        | One expired final-attempt job becomes durably failed even when no new job is claimed. `make test` plus Postgres regression.                                                                |
| 6   | Fence claims and protected effects / 4,5                    | Runtime jobs, worker handler publication boundary, focused tests        | Pause A, reclaim with B, resume A: A cannot heartbeat, finalize or commit protected data. Test ownership loss and lock ordering with deterministic barriers.                               |
| 7   | Add source instances and receipt identity / 1,2             | New forward SQL migration, runtime models/rawfiles, core input contract | Same bytes can have distinct source receipts; duplicate concurrent upload converges; incomplete filesystem/SQL writes are recoverable. Verify constraints and raw-store fault tests.       |
| 8   | Add interpretation membership and ownership constraints / 7 | Forward SQL migration, runtime models, import helpers                   | One generic source identity authority; selected observations belong to canonical objects; units and source scope are explicit. Test invalid direct inserts and legacy-row backfill checks. |

Checkpoint B: durable work failures are truthful; schema invariants and claims are proven before increasing automatic traffic. Task 6's protected publication adapter is reused by later imports, not
copied.

### Phase C: Replace reconciliation and read-freshness internals

| ID  | Work item / dependency                                     | Likely files                                                         | Acceptance and verification                                                                                                                                                                                |
| --- | ---------------------------------------------------------- | -------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 9   | Publish activity observations without Apple tracker / 6,8  | `apps/iroha-imports/activity_import.go`, activity queries/tests      | Stable canonical IDs, observation-owned detail data, per-source publication and stale-snapshot policy. Test A/B/older A and simultaneous snapshots through list/detail reads.                              |
| 10  | Publish sleep and daily projections under source rules / 9 | `sleep_import.go`, `daily_import.go`, domain query adapters/tests    | Cross-midnight sleep identity and source-priority daily values survive overlap and unchanged confirmation. Test missing versus measured zero and selected ownership.                                       |
| 11  | Replace purge-based reprocessing / 9,10                    | `reprocess.go`, interpretation helpers, integration tests            | A → B → reprocess A preserves the required selected state, IDs and other-source support. Remove obsolete tracker/purge writes only after the regression passes.                                            |
| 12  | Write Postgres revisions transactionally / 8               | Runtime revision helper/migration; transaction-aware writer adapters | Import, media, expense and coverage mutations commit their revisions atomically; rollback leaves neither. Verify each writer dependency and sorted lock order.                                             |
| 13  | Use revision vectors for cache and coalescing / 12         | Runtime cache, HTTP read-cache wrapper/tests                         | Cache and singleflight keys include captured primary-DB revisions. Separate-process writer crash after commit cannot leave the next reader on old data. `make test` plus cache-backend integration matrix. |
| 14  | Assemble reports from a consistent read snapshot / 13      | `pkg/reports`, domain read transaction adapters, HTTP tests          | Barrier-controlled concurrent write yields wholly before or after report, never mixed. No response bytes emitted before successful assembly; verify with both cached and cold reads.                       |

Checkpoint C: provenance/replay and committed reads are correct. This is the foundation required by the visible product; it is not itself v0.5 completion.

### Phase D: Complete automatic intake, coverage and late expenses

| ID  | Work item / dependency                              | Likely files                                                       | Acceptance and verification                                                                                                                                                                           |
| --- | --------------------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 15  | Persist and query scoped coverage / 8,12            | Coverage migration/service, core contracts, OpenAPI                | Unknown/partial/covered-empty/unsupported are distinguishable; holes remain visible; unchanged values can still advance confirmation. Test successful empty versus ambiguous empty and account scope. |
| 16  | Deliver chosen automatic Health intake / 3,10,11,15 | Chosen provider adapter, receiver, device artifact, contract tests | Versioned authenticated bounded payload flows into new publication contract. Duplicate/late/partial batches preserve history; required categories meet device fixture expectations.                   |
| 17  | Schedule media and link fetch outcomes / 6,7,15     | `sync.go`, worker initialization, connector state/tests            | Configured sources run without clicks; manual/scheduled overlap coalesces; receipt-to-import outcomes and replay are visible. Test restart, 429, failed child parse and disabled sources.             |
| 18  | Preserve media matching decisions / 8,12,17         | Media resolution/import service, decision model/tests              | Exact IDs auto-link; ambiguous matches need a recorded choice; user decisions and undo survive sync and bridge refresh. Snapshot dates never become consumption time.                                 |
| 19  | Add expense accounts and refund semantics / 8,12    | Expense migration/service/API, shared expense/report types         | Existing amounts remain valid; gross/refund/net and separate counts/currencies are explicit. Test refunds, unlinked refunds, manual entries and cross-month edits.                                    |
| 20  | Import normalized expense statements / 15,19        | Statement parser/service, import endpoints/contracts/tests         | Preview/validation, stable row identity, replay/correction/tombstones and scoped completeness work. Prime August report/series caches then import in September; verify August changes only.           |

Checkpoint D: the three input domains work end to end, including delayed input and recovery. Complete new-field report/metric wire support before frontend integration; keep data-model changes from
becoming UI-local calculations.

### Phase E: Deliver the mature cockpit on the new contracts

| ID  | Work item / dependency                                              | Likely files                                                                     | Acceptance and verification                                                                                                                                                                                        |
| --- | ------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 21  | Expose consistent cockpit status and period contracts / 14,15,18,20 | Briefing/overview/report/metric endpoints, OpenAPI and shared types              | Facts retain domain time scopes; collection and observation coverage differ; calendar-closed labels do not claim complete data. `make contract-check` and period integration matrix.                               |
| 22  | Build Connections and actionable attention / 17,18,21               | Source API client, attention/administration adapters, shared components          | See last receipt/ingestion, covered periods and the actual next action; retry/reconnect/resolution works without DB access. Existing personal tasks remain distinct. Browser and API flow tests.                   |
| 23  | Update Today and Overview / 21,22                                   | Today state, overview route, shared compositions                                 | Independent domains, explicit dates, retained loading content, focus/pending refresh and honest supplemental failures. `make web-test`, theme-boundary and responsive checks.                                      |
| 24  | Complete domain, expense and report surfaces / 19,20,21             | Activity/night/library/expense/metric/report view adapters and shared components | Detailed history, comparisons, statement reconciliation and refund/net labels work under every supported theme. Break into per-route dispatches; verify the capability inventory, not only happy-path screenshots. |
| 25  | Restore API/CLI and public-publishing parity / 21,24                | CLI, public exporter/site adapters, contract/privacy tests                       | Identified clients work after coordinated contract changes; preview/export is consistent and contains only permitted fields. Verify private/public negative cases and existing links/ID mappings.                  |

Checkpoint E: full agreed feature capability is present. Accessibility, themes, error/empty/partial states and detailed activity data are release requirements, not optional follow-up work.

### Phase F: Rebuild, rehearse and release

| ID  | Work item / dependency                                    | Likely files                                                         | Acceptance and verification                                                                                                                                                                     |
| --- | --------------------------------------------------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 26  | Rehearse data rebuild and rollback / 11,14,20             | Focused migration/replay scripts and tests                           | Restore DB+blobs, convert provenance, replay, restore manual state, compare IDs/totals/details; rollback preserves data. Measure duration and retain discrepancy ledger.                        |
| 27  | Run full product and operational acceptance / 16-18,22-26 | Existing release-candidate tooling, scenario fixtures, release audit | All gates below pass on representative data; two-week daily trial records interventions/delay/failure recovery. No unverified coverage or foundation claim is marked done.                      |
| 28  | Cut over and release v0.5.0 / 27                          | Release/version/docs and project-owned deployment targets            | Execute rehearsed window, verify client/public behavior and fresh ingestion, then observe. Tag/deploy/publication follow explicit release authorization; do not change VERSION during planning. |

Task 24 is a bounded feature group whose per-route subitems must be dispatched separately, not one giant frontend patch. Other table entries should likewise be split when actual changes exceed a
focused session. This plan does not force a misleading file-count estimate before implementation tracing.

## Dependency and parallel execution rules

Critical foundation path: 1 → 2 → 4/5 → 6 → 7/8 → 9/10 → 11. Revisions 12-14 can proceed after the SQL/transaction contract stabilizes. Device trial 3 runs early alongside foundation work; do not
discover transport failure at release time.

After shared contracts, Health 16, media 17-18 and expenses 19-20 can proceed independently. One integrator owns migration numbering, shared transaction helpers and contract merges. Frontend design
exploration can run earlier, but stateful integration waits for 21. No parallel worker should change shared ownership semantics independently.

First dispatch: task 1, then task 2 and the device-trial preparation for task 3. First production-code fix: task 4. Do not start by rewriting Today or registering recurring live syncs.

## Release gates and test evidence

| Gate                 | Required proof                                                                                                                                                                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Job correctness      | False-success, expired-final-attempt recovery, claim fencing and stale-handler publication tests pass                                                                                                                                             |
| Data integrity       | Full/partial, duplicate, out-of-order, source conflict, corrected observation and A/B/reprocess tests preserve intended identities and values                                                                                                     |
| SQL ownership        | Invalid selected links and duplicate scoped identities are rejected; migration backfill checks pass                                                                                                                                               |
| Read consistency     | Two-process crash-after-commit test and barrier-controlled report test pass for Postgres/Valkey response storage and no-cache mode                                                                                                                |
| Health usability     | Required daily categories arrive automatically during ordinary use; failed/missed device opportunities recover or have an explicit source limitation; full-fidelity claims match actual transport evidence                                        |
| Media usability      | No routine sync clicks; exact events/date facts/snapshot state remain distinct and matching decisions persist                                                                                                                                     |
| Expense correctness  | Normalized statements, refunds, repeated/corrected imports and account-specific completeness are verified with historical cached reports                                                                                                          |
| Surface completeness | All agreed routes, detailed domain data, scopes, themes, task flows, CLI clients and public archive pass their acceptance inventory                                                                                                               |
| Privacy and security | Scoped ingestion credentials, input limits, redacted diagnostics, private authorization and public-field/route-policy negative tests pass                                                                                                         |
| Recovery             | Aligned backup restore, maintenance cutover and rollback are exercised, including treatment of post-cutover writes                                                                                                                                |
| Performance          | On recorded reference hardware and representative ten-year history, target p95 under 500 ms for ordinary non-export reads; expensive routes/exports are bounded. Record cold and warm separately; revise only with evidence and explicit tradeoff |
| Daily operation      | Two-week trial has no routine export/sync intervention for supported sources; record actual lag, exception interventions and unresolved gaps rather than declaring success from one demo                                                          |

Run `make check` before commits, appropriate disposable DB integration tests for persistence work, `make validate` and `make release-candidate` at release scope. Existing `make test-integration`
starts dependencies/applies migrations; it must use the intended test environment. `make validate` includes public-site data generation, so use the seeded rehearsal database, not an arbitrary personal
connection.

The earlier planning run's tool downloads failed certificate validation. The user's `.mise.toml` has since changed. Revalidate the current setup in task 2 rather than treating the old failure as
permanent or changing their config automatically. Never disable TLS verification to make a gate green.

## Feasibility boundaries

This is a substantive release with structural and product work. The plan controls effort by retaining the stack, modules, source knowledge and visual assets; making one schema/maintenance transition;
supporting one explicit expense format; and choosing one demonstrated Health transport. It does not reduce the final cockpit to a minimal dashboard.

Do not add a generalized plugin framework, arbitrary report builder, automatic foreign-exchange conversion, every bank/provider adapter, or a new mobile app without a demonstrated release need. Native
Health capture remains the explicit contingency for a failed device-transport gate, with a revised effort estimate before committing to that path.

Uncertainty is concentrated in the Health transport, actual retained data/consumer inventory, and replay duration. The first phase resolves these while foundational fixes progress. Estimates should be
based on those results; no calendar delivery date is asserted by this document.

## Migration tooling and implementation handoff

Use SQLx CLI instead of Goose for v0.5 database migrations. This is a migration-runner change; the Go application and its database access layer remain in place. SQL migrations remain authoritative.
The [SQLx CLI documentation](https://github.com/launchbadge/sqlx/blob/main/sqlx-cli/README.md) describes custom migration directories and separate reversible `.up.sql` / `.down.sql` files.

Include the tooling transition in task 2, before tasks 7 and 8 introduce new schema migrations:

- Convert the existing 12 Goose migrations into paired SQLx files, preserving SQL, ordering and rollback behavior. Do not pass the combined Goose files directly to SQLx: their Down sections must not
  execute during apply.
- Update `scripts/db.py` to use SQLx run/revert/info while retaining the existing Make entry points. Update wrapper tests, `scripts/release_candidate_reset.sql`, operational instructions and CI/tool
  provisioning together, including the Goose requirements in `AGENTS.md` and `CONTRIBUTING.md`.
- Replace the Goose build stage in `ops/images/Containerfile.server` and adapt `ops/images/db-migrate-entrypoint.sh`. Test SQLx in the final runtime image and preserve the deployed invocation
  contract, or coordinate its change with deployment consumers before release.
- Define and rehearse adoption for an existing Goose-managed database. Verify its applied versions and actual schema before recording equivalent SQLx history/checksums; refuse unknown or partially
  applied states. Never rerun already-applied DDL or blindly mark migrations applied. Exclude concurrent migration runners during adoption. Retain the original history for audit; rehearse restoring
  schema and both tools' migration metadata together, including rollback after later SQLx migrations.
- Verify a fresh database, an upgraded existing database, a second no-op apply, a failed migration, and reversible rollback/reapply on disposable databases. Compare resulting schemas and retained
  records. Keep this transition separate from v0.5 domain/schema changes so failures are attributable.
- Use the user's Nix-managed SQLx and Prettier where available, with explicit executable/version checks. The currently observed SQLx resolves through `~/.cargo/bin`, so verify the intended executable
  rather than assuming Nix ownership. Document and provision compatible versions in CI and the implementation environment. Keep Go, uv, Bun and golangci-lint project-managed; do not restore Goose as
  the long-term dependency.

This PR contains the planning handoff and the user-selected mise configuration/lockfile changes. Implementation will happen in another environment. Read this file as the canonical plan; the three
earlier review/rebuild documents retain supporting evidence and alternatives. The local Asobi graph is optional handoff context, not a prerequisite: all task scope, acceptance and dependencies are in
this file. Revalidate the target checkout, toolchain, deployment data and task status before execution.

## Planning verification

This plan chooses concrete decisions from source-backed reviews and independent ingestion/cache feasibility checks. It does not claim that v0.5 changes, live device trials or migrations have been
executed. The current change includes documentation, local task-state registration and the user-selected mise configuration/lockfile changes. No components or stored data were deleted.

Planning validation on 2026-09-10: after the user restored golangci-lint, `make check` and `make validate` passed Go formatting, vet, lint, unit/contract tests, 91 Python script tests and
theme/responsive/motion checks, then stopped at frontend formatting because `prettier-plugin-svelte` was unavailable. `make web-install public-site-install` could not install dependencies because
downloads failed with `SELF_SIGNED_CERT_IN_CHAIN`, including retries using the machine's CA bundles; TLS verification remained enabled. Scoped formatting of all four planning documents and staged
whitespace checks passed. Full repository validation, builds, device trials and database rehearsals remain unverified. The PR also includes the user-selected `.mise.toml` and `mise.lock` changes.
Goose has been removed from tool provisioning; migration commands still require it until the planned SQLx transition is implemented.
