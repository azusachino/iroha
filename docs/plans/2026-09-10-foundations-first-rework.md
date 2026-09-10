# Foundations-first review and rework

Status: proposed; prerequisite to automation adoption and surface work.

Source baseline: `main` at `8f798b0e9d3876897293204b23d63ed855c37421`, reviewed 2026-09-10 JST. Findings below come from source and SQL inspection, not a deployed-data audit or executed concurrency
reproduction. Unverified areas are explicit work items, not claims that those areas are safe or defective.

## Priority and relationship to the earlier plan

The user wants the underlying design reviewed and repaired before surface improvements. This plan takes precedence over the execution order in
[Automatic sync, data coverage, and late-arriving history](2026-09-10-automatic-sync-coverage-and-late-data.md). That document remains the detailed downstream product proposal; it is not an
architectural approval.

Begin with ownership, identity, SQL invariants, evidence lifecycle, canonical resolution, transaction boundaries, and worker correctness. Then implement late-data/coverage contracts and automatic
ingestion. Only after the relevant correctness gates pass should Today, Overview, and other user-facing surfaces adopt the new behavior.

This is not authorization to discard existing data, reset databases, rewrite all modules, or deploy changes. Preserve the working system while repairing one verifiable boundary at a time. Safety fixes
in the current implementation should land before any broad model migration.

## Ground-level product and data contract

Iroha is a personal record system assembled from incomplete, delayed, overlapping sources. Its first obligation is to retain and explain trustworthy history. Presentation depends on that obligation
being met.

The design must answer these questions before choosing tables or jobs:

1. What is a source instance: a provider, an account, a device, or a particular export stream?
2. Which identifier survives retries, source corrections, and parser upgrades?
3. What does a provider report, and what does Iroha select as the canonical value?
4. Can two sources disagree without one silently destroying the other's evidence?
5. Which time describes the event, the source capture, the receipt, and the processing attempt?
6. What establishes that an absence is a deletion, an empty period, or merely unknown?
7. What is atomic, and what recovery obligation remains after a crash?
8. How does a reader know which committed revision it is seeing?

Start single-user. Source-instance scoping is required for multiple accounts/devices and overlapping ingestion methods; it does not require speculative multi-tenant SaaS infrastructure.

## Confirmed structural concerns

Paths and line numbers refer to the source baseline.

### F1. Canonical identity is entangled with the file that first created it

`apps/iroha-imports/reprocess.go:40-46,60-85` deletes Apple source items, observations, and canonical objects based on the raw file or its first/last snapshot membership. Observation deletion includes
first-seen membership even if the observation has since been updated by a newer import. Missing identities then cause new UUID allocation, for example `daily_import.go:71-79`.

Source-derived failure scenario: import A, import corrected B with the same source identity, then reprocess A. The purge can remove state now supported by B and recreate canonical IDs from A. There is
no executed reproduction yet; the implementation gate must establish the exact behavior and protect stable IDs, references, and newer evidence.

**Required design:** canonical identity is independent of one raw file. Reprocessing updates that evidence's interpretation, then re-resolves affected objects using all eligible observations and
explicit precedence. Removing one derivation must not imply removal of an object supported elsewhere. Decide whether original versus corrected interpretations are retained as revisions or reproducible
from evidence plus parser version; do not require full event sourcing by default.

### F2. Snapshot authority is based on execution order rather than a verified source sequence

`apps/iroha-imports/activity_import.go:17-25,71-88,116-135` reconciles every complete Apple snapshot against all tracked Apple source items. The reviewed path has no source-scoped publication lock or
comparison with the active snapshot's source order. `service.go:304-309` supplies snapshot time, but this is not an ordering guard.

An older export arriving later can remove records present in a newer export. Overlapping jobs introduce an additional race. Daily partial input adds another destructive ambiguity if it is treated as a
full export.

**Required design:** validate ingestion mode (`full snapshot`, `bounded replacement`, or `incremental events`), source scope, covered interval, and ordering basis. Serialize publication for that
scope. Receipt time alone cannot prove a snapshot is newer. When capture order is unknowable, accept evidence but require an explicit authority/reconciliation policy before absence-based deletion.
Source deletions must only affect records within the validated source scope and mode.

### F3. Two identity/provenance systems do not advance together

SQL already defines both `tb_apple_source_items` and `tb_source_observations` (`00001_current_schema.sql:180-193,257-268`). Unchanged Apple workouts, sleep, and daily values return after updating only
the Apple tracker (`activity_import.go:162-167`, `sleep_import.go:64-69`, `daily_import.go:64-68`). The generic observation tracker can therefore retain an older last-seen snapshot.

**Required design:** declare one authoritative identity and confirmation contract. Distinguish last observed from last changed. Skipping an unchanged value write must still record any promised
confirmation/coverage. Migrate the legacy tracker only after parity evidence; do not add a third identity table to work around the first two.

[ADR 0001](../adr/0001-provider-observations-and-canonical-records.md) is still marked Proposed. Observation tables exist, but their presence does not prove the proposed observation-first selection
architecture is complete. Reconcile that ADR against actual behavior before adopting or revising it.

### F4. SQL enforces existence more strongly than semantic ownership

`00001_current_schema.sql:296,360,415-416` makes selected-observation IDs foreign keys. Those FKs do not enforce that the selected observation belongs to the same canonical object.
`tb_sleep_session_observations` introduces an additional relationship alongside the direct observation-to-session FK (`:337-380`). The intended ownership/multiplicity must be explicit.

Other scope decisions are embedded in uniqueness: raw SHA is globally unique (`:7`), Apple source keys are globally unique (`:193`), generic observations use `(provider, source_kind, source_key)`
(`:268`), and canonical daily metrics use `(day, metric)` (`:176`). Some are valid for a single selected projection; they must not be reused as independent source-observation keys without account,
unit, or source semantics being settled.

**Required design:** publish an invariant matrix and enforce critical ownership constraints with suitable composite FKs/uniqueness, or simplify redundant relationships. Define canonical metric units
rather than blindly adding `unit` to every key. Audit existing rows before tightening constraints. A state label such as `unresolved` also needs a precise meaning when an observation's canonical FK is
non-null.

### F5. Blob deduplication and receipt provenance share one row

`apps/iroha-runtime/rawfiles/service.go:80-86,131-139` returns an existing raw row solely by content hash. That row also owns source kind and, for connector snapshots, observation time. Identical
bytes arriving again do not create a new receipt at this layer. `Create` renames a file before SQL insertion and does not remove the renamed file if insertion fails (`:94-111`); `StoreSnapshot`
removes its file on insertion failure (`:148-165`).

**Required design:** retain content deduplication while representing each relevant source receipt independently, using existing import/snapshot records where sufficient. The same bytes may be received
at different times or under different source contexts. Define the filesystem/SQL failure states and a bounded reconciliation procedure; a database transaction alone cannot atomically commit filesystem
bytes. Concurrent identical uploads should converge on one blob and valid receipts, not an unexplained uniqueness error or orphan accumulation.

### F6. Worker status and lease ownership do not establish safe execution

The earlier plan identifies import errors returning nil after successfully writing failed status (`apps/iroha-imports/service.go:423-429`). In addition:

- `apps/iroha-runtime/jobs/service.go:226-237,262-264` completes/fails by job ID and running status without checking the current owner/attempt. A stale worker can affect a job reclaimed by another
  worker.
- Lease refresh at `:299-315` can stop without canceling the executing handler. Owner fencing must protect publication, not merely the final status update.
- `ClaimNext` performs expired-job recovery inside a transaction, then returns `ErrNoJobAvailable` if it claims nothing (`:188-216`). That error rolls back recovery. When only an expired final-attempt
  job remains, its terminal recovery at `:476-488` can repeatedly roll back.

**Required design:** preserve handler errors, commit recovery even when there is nothing new to claim, and require an owner plus claim-generation/attempt token for heartbeats, final transitions, and
relevant publication. Reject zero-row ownership transitions as lost claims. Cancel on lost lease, while still making database side effects safe against a worker that has not stopped yet. Keep delivery
at-least-once and make handlers idempotent; do not promise exactly-once execution.

### F7. Canonical commit and cache visibility have an unclosed process boundary

`apps/iroha-imports/service.go:314-330` persists canonical results, then separately completes import status and flushes cache. A crash or status-write failure can leave committed data without the
normal invalidation. `:378-385` logs an invalidation error. Cache degraded state is client-local (`apps/iroha-runtime/cache/cache.go:130-131,342-373,405-413,439-449`). The worker and HTTP server are
separate processes, so worker-local degraded state does not make the server bypass stale shared entries.

**Required design:** choose one durable post-commit visibility mechanism. Candidate: record a namespace invalidation intent in the canonical transaction, retry it durably, and make readers detect a
pending/new canonical revision rather than serve an old cached revision while intent delivery is delayed. Alternatively, transactional Postgres revision counters can participate directly in read-cache
identity, including when response bytes are in Valkey. Select the simplest mechanism that passes the separate-process/crash test; an outbox by itself only gives eventual delivery, not immediate read
freshness.

Do not replace the existing cache system wholesale. Preserve generation-safe population, namespace dependencies, and canonical-query fallback. Revisit
[ADR 0004](../adr/0004-cache-correctness-and-report-reads.md)'s process assumptions explicitly; one server plus a separate writer is already relevant.

## Target model: ownership before more tables

| Layer                            | Owns                                                                             | Must not decide                                              |
| -------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Evidence blob                    | Original bytes and integrity                                                     | Event identity or source completeness                        |
| Source receipt / import snapshot | Source instance, capture/receipt time, input mode, evidence reference            | Which conflicting value wins merely by arrival order         |
| Provider observation             | Stable source identity, typed facts, source precision, interpretation version    | Global canonical identity through an archive hash            |
| Canonical record and resolution  | Stable domain identity, chosen observations, explicit correction/selection rules | Deleting independent evidence when one source disappears     |
| Coverage assertion               | Supported source/category/period actually collected as of evidence               | Fabricated activity, measurements, or universal completeness |
| Durable job attempt              | Claim, execution, retry, failure and recovery                                    | Treating a fetch as completed downstream materialization     |
| Read projection/cache            | Reproducible view of a committed canonical revision                              | Owning irreplaceable history or hiding stale data            |

These are conceptual boundaries. Reuse existing tables when they can express them correctly. Do not create one table per box without a demonstrated lifecycle or integrity need.

## SQL and domain invariant review

Before any foundational migration, produce a table-by-table inventory from all forward migrations, runtime models, and actual writer/read paths. `docs/data-model.md` contains MVP-era descriptions and
is not a sufficient current schema inventory.

For each table record its authority, primary/business keys, incoming/outgoing references, nullable meanings, checks, deletion behavior, write owner, read consumers, and rebuildability. The initial
pass must cover raw/import/snapshot tables, both provenance trackers, activity and observation children, sleep links/segments, daily projections, media state/events, expenses, jobs/schedules, cache,
and tasks/public-export relationships where they reference canonical IDs.

| Invariant family    | Concrete checks                                                                                                                                                                            |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Identity            | Same source instance/key survives retry and parser change; distinct accounts cannot collide; blob identity is separate from receipt and event identity.                                    |
| Selection           | Selected observation belongs to canonical object; preferred flags and selected IDs cannot disagree; supported multi-source observations survive source removal.                            |
| Value validity      | Domain-approved units, finite values, valid intervals, nonnegative durations, permitted media states, exact versus date-only time precision. Do not impose universal positive-value rules. |
| Derived consistency | Canonical totals and child streams agree with the selected interpretation; reducers are versioned and deterministic; daily-average rollups use the documented weighting.                   |
| Deletion            | Full source removal, partial absence, user deletion, correction, and parser reprocess have distinct behavior; links and user overrides survive according to explicit policy.               |
| Jobs                | Claim ownership, legal transitions, retry bounds, terminal recovery, and schedule uniqueness are enforced and tested under concurrency.                                                    |
| Expenses            | Stable source/account transaction identity, correction versus replay, currencies, credits/refunds, statement scope, purchase/posted date policy.                                           |
| Query/index fit     | Trace actual joins/filters/orderings; inspect plans on representative fixtures before adding indexes. Verify stable pagination under equal timestamps.                                     |

Do not label every missing SQL CHECK a production bug. Classify each invariant as database-enforced, application-enforced with evidence, or unverified; add constraints where they protect meaningful
boundaries.

## Foundation-first execution order

### Gate 0: Establish a reproducible baseline and recovery path

Deliver a current schema/writer inventory and a small deterministic fixture set. Include full export A, changed B, older A replay, partial C, duplicate inputs, competing source instances, conflicting
observations, and late/corrected expenses. Use synthetic fixtures by default; private evidence stays local.

Restore the pinned toolchain trust configuration so repository gates can execute without disabling TLS verification. Rehearse database-plus-blob backup/restore in a disposable environment. Record
baseline canonical IDs, totals, references, and hashes. Existing correctness bugs may have failing reproduction tests; label them explicitly.

Exit: reproductions are runnable, expected semantics are written down, and restore is verified. No production migration or data reset is part of this planning task.

### Gate 1: Fix false success and job ownership/recovery

Implement F6 and the nil-error fix first in imports/runtime jobs. Ensure recovery commits with an otherwise empty queue. Fence stale attempts and verify that lost workers cannot publish or transition
another claim. Preserve bounded retry and Retry-After support.

Exit: parser failure is not success; final-attempt abandoned jobs become terminal; worker A loses its lease to B and cannot overwrite B's state or publish stale results. Repeat delivery is safe.

### Gate 2: Freeze identity, evidence, time, and resolution contracts

Decide the authoritative provenance tracker, source-instance scope, immutable receipt semantics, supported ingestion modes, snapshot ordering, canonical ID stability, and user correction precedence.
Define what “latest” means when source clocks are absent or disagree. Define last observed, last changed, occurrence date, and source capture independently.

Reconcile proposed ADR 0001 and provider contracts with those decisions. Explicitly choose whether unresolved observations may exist without a canonical object, and whether sleep observations have one
owning session or a many-to-many relationship. Use actual supported producer requirements, not hypothetical universal matching.

Exit: every baseline fixture has an unambiguous expected result. A parser upgrade and an older arriving snapshot cannot implicitly erase newer authoritative history.

### Gate 3: Enforce SQL ownership and repair evidence publication

Implement F4/F5 using forward migrations and the smallest compatible model changes. Audit existing violations before adding constraints. Introduce source-instance/receipt references only where the
frozen contract requires them. Resolve raw storage insertion races and orphan handling. Stage backfills separately from destructive cleanup.

Exit: invalid cross-object selections are rejected; identical blob deliveries retain correct receipt context; concurrent ingestion converges; filesystem/SQL failure injection leaves recoverable,
inspectable state. Backfill counts and relationship checks pass.

### Gate 4: Make reconciliation and reprocessing preserve history

Implement F1-F3 against the now-authoritative identity/receipt contract. Serialize snapshot publication, reject or quarantine stale authority, preserve independent observations and stable canonical
IDs, and advance confirmations consistently even when values are unchanged. Re-resolve changed objects instead of globally purging by first raw file.

Exit: A → B → reprocess A keeps stable identities and the selected state required by the authority policy; removing A does not remove B-supported facts; partial C never activates global deletion;
concurrent full snapshots converge deterministically. Existing complete-export deletion semantics still work within their validated scope.

### Gate 5: Close transaction-to-read visibility and late-data semantics

Implement F7 with a durable revision/invalidation design and test the separate worker/server boundary. Preserve occurrence-date queries and existing namespace invalidation. Integrate statement
coverage and correction semantics only after expense identity is defined. Add collection coverage independently from observation density and calendar closure, as detailed in the earlier plan.

Exit: after canonical commit, a fresh read cannot return an older cached revision as current under the promised consistency policy; crash and invalidation failure recover. Late August expenses update
primed August report/series caches without becoming September spending. Missing data stays unknown; supported covered-empty periods need no fake domain records.

### Gate 6: Migrate safely and prove the foundations end to end

Run fresh-install and upgrade migrations on disposable copies, verify checks/FKs/selected ownership, compare IDs and aggregates against baseline, and rehearse interruption/restart at migration
boundaries. Prefer additive changes plus backfill and read switch; retain old structures until parity is established. Do not edit the squashed baseline migration to upgrade existing deployments.

Run `make test`, `make contract-check`, relevant integration tests, and `make check`. Use `make validate` and release-candidate verification at release scope. Integration targets can start
dependencies/apply migrations and must use the intended test environment. Test both supported cache backends and no-cache fallback; inspect representative query plans and resource use before declaring
performance sufficient.

Review private-data access and public-export boundaries before exposing the new ingestion endpoint: credential scope, payload limits, safe diagnostics, raw-file access, and lack of private-field
leakage. These security/performance/restore areas remain unverified until this gate; the present review is not certification.

Exit: no unresolved history-loss, false-success, stale-owner publication, identity-collision, or acknowledged-stale-read issue remains in the exercised supported paths. Record residual limits with
evidence rather than declaring the whole system perfect.

### Gate 7: Adopt automation, then improve surfaces

After foundations pass, execute the earlier plan's Health Shortcut trial, media scheduling, and end-to-end readiness work using these contracts. Validate actual device coverage and recovery. Then
update Today, Overview, reports, and shared theme components to expose truthful status and refresh behavior.

Exit: automatic inputs are safe before the UI depends on them. Two weeks of ordinary use measures manual interventions and unexplained gaps. Surface changes consume server-owned facts and coverage;
they do not conceal unresolved foundation defects.

## Required adversarial scenarios

| Scenario                                                     | Invariant proved                                                                        |
| ------------------------------------------------------------ | --------------------------------------------------------------------------------------- |
| Same bytes, different receipt time/source instance           | Blob deduplication does not erase provenance.                                           |
| Two concurrent identical uploads                             | One valid blob identity, usable receipts, bounded recoverable filesystem residue.       |
| Full A, newer B, parser reprocess A                          | Stable canonical identity and explicit source authority survive reprocessing.           |
| Unchanged A then B                                           | Values remain stable while promised last-seen confirmation advances consistently.       |
| Older complete snapshot after newer snapshot                 | No silent rollback or global deletion based on arrival order.                           |
| Selected observation from another canonical object           | SQL or equivalent hard boundary rejects the relation.                                   |
| One source disappears while another supports the fact        | Evidence and canonical selection are reconciled without deleting independent support.   |
| Stale worker resumes after lease reassignment                | Old claim cannot heartbeat, finalize, or publish over the new claim.                    |
| Only expired final-attempt job remains                       | Recovery reaches a durable terminal state even with no next claim.                      |
| Crash after canonical commit, before completion/invalidation | Job replay is idempotent; readers follow the declared committed revision.               |
| Worker-only invalidation failure with HTTP server alive      | Process-local flags cannot hide the stale-read risk.                                    |
| Cross-month expense correction and late import               | Both periods and all affected cached projections update correctly.                      |
| Restore DB without matching blob state, then matched restore | Missing evidence is detected; complete recovery is demonstrated with aligned artifacts. |

## Scope limits and recommendation

Keep Go, Postgres/PostGIS, explicit migrations, domain tables, provider adapters, and the durable queue unless measurements or invariants show a specific reason to change them. Do not treat preserving
these technologies as approval of their current schema and transaction design.

The first implementation should be the small job/error correctness slice plus executable regressions for snapshot ordering and reprocessing. The next decision is the canonical identity/provenance
contract. New dashboards, theme work, generic rollup tables, additional providers, and a native mobile app wait until these foundations can safely support them.

## Planning verification

This document was checked against the source baseline and an independent jobs/cache and import/provenance review. No database, device, concurrency, migration, or restore test was executed for this
planning task. The previous turn's required `make check` attempt was blocked before checks by pinned tool-install certificate errors; no toolchain or TLS bypass is proposed. Documentation formatting
can be verified with the installed formatter, but that does not replace the implementation gates above.
