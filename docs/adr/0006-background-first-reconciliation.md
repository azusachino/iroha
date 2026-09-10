# ADR-0006: Background-first reconciliation

- Status: Accepted
- Date: 2026-09-10
- Scope: provider-backed imports, canonical projections, and media matching

## Context

Iroha should minimize routine user interruption. A human-facing resolution inbox
turns normal provider disagreement into a personal task queue, and the current
resolution API records decisions without applying them to canonical data. That is
the wrong default for automatic intake and a poor handoff surface for agents.

Ambiguity still exists. The system must preserve evidence and avoid false merges,
but it must complete an import without waiting for a person.

## Decision

Reconciliation is background-first and agent-accessible:

- exact provider IDs and trusted bridge IDs attach automatically;
- one exact title/date candidate attaches automatically;
- multiple or weak candidates remain separate source-owned records with
  provenance intact; they do not create an open human task or fail the import;
- conflicting current values are retained as source observations/history, and a
  deterministic domain rule selects the current projection; arrival order is not
  the rule;
- raw evidence, source observations, and machine-readable match state remain
  queryable so an agent can inspect and reconcile a case when explicitly asked;
- the default UI and import path contain no resolution inbox, confirmation prompt,
  or required human decision.

Human correction remains an explicit operation, not a prerequisite for ordinary
sync. This decision never permits discarding evidence, silently merging ambiguous
records, or hiding a conflict.

## Alternatives considered

### Human resolution inbox

Rejected. It interrupts automatic intake, accumulates work the system cannot
apply, and makes the default behavior depend on manual triage.

### Fail the whole import on ambiguity

Rejected. One uncertain match should not prevent unrelated source facts from being
imported and preserved.

### Last-write-wins

Rejected. Arrival order is not source authority and can silently roll back a newer
or stronger observation.

## Consequences

- Media resolution task creation and its human-facing API/UI are not part of the
  v0.5 target path.
- The importer needs deterministic per-domain selection rules and tests for
  ambiguous, conflicting, delayed, and repeated observations.
- Agents need a read/reconcile command or equivalent machine-readable API; that
  surface is for explicit investigation, not a recurring user inbox.
- Separate records may temporarily exist for genuinely ambiguous identity until
  an agent or stronger source rule can reconcile them.
