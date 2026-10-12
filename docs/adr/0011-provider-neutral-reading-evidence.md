# ADR-0011: Provider-neutral reading evidence and controlled outbound synchronization

- Status: Proposed; awaiting owner's final review, not implementation approval
- Date: 2026-10-12
- Scope: book-source ingestion, identity, selected reading state and optional outbound synchronization
- Origin: [#142 reading-hub handoff](https://github.com/azusachino/iroha/issues/142#issuecomment-6116906753)
- Related: [ADR 0002](0002-provider-adapter-contracts.md), [ADR 0005](0005-media-provider-time-semantics.md), [ADR 0006](0006-background-first-reconciliation.md)
- Detailed proposal and evidence: [Reading Hub Phase A](../plans/reading-hub.md)

## Context

BookOrbit is a useful file/reader synchronization owner, but a book can exist in Iroha without a local file. WeRead and Kindle Unlimited histories must not depend on creating synthetic EPUBs. The synthetic BookOrbit 3.3.0 tests in [PR #148](https://github.com/azusachino/iroha/pull/148) show that ordinary writes do not preserve lossless private WeRead provenance/history. Hardcover provides catalogue and personal-list GraphQL surfaces, with beta compatibility and permission limits.

Iroha already has provider/connector registries, immutable intake bytes, source receipts, interpretation snapshots, media works/items/state history and Postgres-backed jobs. However, existing media persistence does not yet implement a complete account-scoped, field-authority reading ledger. Reusing table names is not proof that those semantics work: the detailed proposal identifies actual gaps.

## Decision

Propose a reading capability within existing Iroha components, not a separate service or universal synchronization framework:

1. **Independent inbound sources.** BookOrbit, WeRead and Hardcover contribute evidence directly through approved adapters. Kindle starts with explicit manual/lawful export evidence. None is a mandatory transit store for another source.
2. **Separate work, edition and file.** A work may have several translation/format items and no file. Keep provider identities and account-local personal records distinct from global catalogue identities. Verified cross-references may connect works; they do not make positions equivalent.
3. **Evidence before projection.** Preserve source-native captures and receipt/version lineage. Deterministic, DB-free parsers produce typed facts. Current status is a selected projection, not the historical authority. Resolve retention before promising replay of unmodeled native fields.
4. **Reuse actual boundaries.** Use existing `provider/v1` media import interfaces, `connector/v1` fetch contract, import transactions and job service. Add narrow typed fields or persistence only for demonstrated book requirements. No Redis, Kafka, extra database, renderer or speculative provider super-interface.
5. **Background-first matching.** Disable generic title/year auto-attachment for new book adapters. Ambiguous books remain separate source-owned records with inspectable evidence; they neither block unrelated import facts nor create a routine human inbox. Explicit correction uses existing agent-accessible matching decisions. Work-only linking across distinct editions needs a tested extension, not item collapse.
6. **Deterministic field authority.** Retain disagreements and select by approved authority/time rules, not arrival order or maximum progress. Exact locators stay reader-owned. Different editions keep different positions; no automatic translation-progress conversion.
7. **Read-only first.** Outbound routes default disabled. BookOrbit keeps its existing Hardcover writer for explicitly delegated target fields. A future Iroha writer must have a verified capability, mapping, field allowlist and exclusive target ownership; disabling a route is not an automatic undo of remote writes.
8. **Durable outbound intent.** When a concrete outbound route is approved, persist operation state/receipts transactionally and execute using existing jobs. Unknown remote outcomes require reconciliation before retry. No exactly-once delivery claim, blind read-attempt insertion or echo fanout.
9. **Private by default.** Iroha owns public opt-in independently of source privacy. Use the current validated `/public/v1` projection, never expose private API credentials, notes, hashes, paths or locators through it.

This is a proposed design. It does not promote the earlier proposed ADRs, approve provider accounts/polling, change #144–#146 acceptance, or authorize deployment/migration.

## Alternatives considered

### Funnel all sources through unmodified BookOrbit

Rejected as the canonical-history architecture. Tested ordinary writes lack lossless external provenance/history and metadata-only creation. A deliberately limited matched-book mirror remains optional, with its losses documented.

### Use Hardcover as the canonical ledger

Rejected. It is an optional external service with beta APIs and a shared work-level personal record; edition/attempt/source evidence is not interchangeable with reader sessions. Provider disappearance or permission changes must not erase Iroha's useful history.

### Create a new book schema/service and generic sync engine first

Rejected. Existing media/import/job boundaries already supply most infrastructure. Concrete gaps justify small SQLx/typed-contract extensions; illustrative handoff entities do not justify duplicating everything.

### Unconditional bidirectional last-write-wins

Rejected. It creates echoes, competing Hardcover writers and fabricated cross-edition equivalence, and mishandles rewinds/rereads. Direction, target ownership, field authority and uncertain outcomes must be explicit.

### Direct BookOrbit database ingestion

Not selected initially. Version-pinned authenticated snapshots keep application access checks and reduce schema coupling. A future read-only database exception would require separate permission/query/compatibility acceptance; Iroha migrations must never modify BookOrbit tables.

## Consequences

Positive:

- Kindle/Hardcover metadata-only books and continuing WeRead usage do not depend on file migration.
- History/provenance survives projection changes under the approved retention policy.
- Independent provider capability limits remain visible; partial capability is useful without fabricated sessions.
- Outbound ownership and receipts prevent blind fanout and unsafe retry behavior.

Costs and prerequisites:

- Book persistence needs tested work-only linking, account-scoped facts, conservative matching and deterministic reading-source selection.
- Existing source-observation rows are mutable latest-record indexes, not an append-only native-payload ledger; version lineage/retained bytes must carry history.
- Existing fetch-run completion is not proof that all enqueued imports succeeded.
- Native author roles, repeated read attempts, annotations, locators and field-authority controls have incomplete typed/persistence support.
- At least two fixture-backed imports, source drift/outage behavior and evidence replay must be accepted before scheduling; live accounts/device journeys remain separate gates.
- A real outbound route needs an additional tested durable-intent seam; the existing job queue alone is not a remote receipt ledger.

## Acceptance and next implementation sequence

The [Phase A plan](../plans/reading-hub.md) contains the capability matrix, minimum observation mapping, inbound/reconciliation/outbound state machines, test-first slices and risks. The owner reviews this complete package after documentation and repository gates pass. Acceptance of this ADR records the architecture, not a runtime provider PASS.

First implementation candidate: private BookOrbit and Hardcover **saved-snapshot** imports, with one explicitly mapped shared work, separate translation items and a metadata-only Kindle/Hardcover item. Prove deterministic normalization, identity, retention and replay before network scheduling or an outbound writer. WeRead's authenticated contract and controlled WeRead → Hardcover route follow as separately scoped slices. Public sharing remains a separate published-contract outcome.
