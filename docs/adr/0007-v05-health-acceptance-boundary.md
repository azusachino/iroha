# ADR-0007: Bound v0.5 Health acceptance to available evidence

## Status

Accepted

## Date

2026-09-11

## Context

The v0.5 scope includes a fresh SQLx schema, replay of the complete original Apple Health export, and an automatic bounded Health intake. The complete original export is available locally and provides
the authoritative historical data set. The repository also contains a versioned `iroha.health.shortcut.v1` fixture and an authenticated container-level intake path.

An earlier planning version made a seven-day physical iPhone trial a release prerequisite. That trial would prove phone transport behavior under lock state, offline delivery, missed days, and
permission ambiguity, but it is evidence about the device and deployment environment rather than evidence about the server's data model or replay behavior.

## Decision

For v0.5 acceptance:

- Use the complete original export replay, exact replay, and rollback rehearsal to prove data fidelity and cut-over safety.
- Use the authenticated `iroha.health.shortcut.v1` fixture through the real container endpoint to prove bounded raw-evidence storage, receipt/import/coverage publication, unknown or partial
  completeness, and exact replay.
- Treat the physical iPhone transport trial as post-v0.5 operational hardening. Do not claim that physical lock/offline/missed-day behavior has been verified until that trial is actually run.

This decision changes the release gate, not the honesty boundary: the v0.5 audit may accept the server and data pipeline without claiming device-level reliability.

## Consequences

- v0.5 can be accepted from reproducible repository and local-runtime evidence already available.
- The physical trial template remains useful for a later operational check but cannot block the fresh-schema release.
- Any later claim about automatic Health behavior on a real iPhone still requires device evidence.
- The SQLx fresh-schema cut-over boundary is unchanged; no legacy-schema migration or Goose adoption is introduced.
