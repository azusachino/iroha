# Apple Health provider capabilities

Status: full-export and bounded Shortcut ingestion implemented; physical-device acceptance remains a deployment check.

## Evidence

Apple Health is ingested as a full export ZIP containing `export.xml` and route files. The export is treated as a complete snapshot and reconciled by stable provider source identity plus content hash.

The automatic path accepts the versioned `iroha.health.shortcut.v1` JSON envelope at `POST /api/v1/intake/health`. It is authenticated with the deployment-scoped `IROHA_HEALTH_INTAKE_TOKEN`, stores the unchanged payload as raw evidence, and queues a normal import job. Shortcut coverage is bounded and uses `bounded_replacement`: partial windows never imply that older records should be deleted.

## Implemented capabilities

| Capability        | Status      | Notes                                       |
| ----------------- | ----------- | ------------------------------------------- |
| Activity sessions | Implemented | Workout records and summary metrics         |
| Routes            | Implemented | WorkoutRoute/FileReference GPX linking      |
| Sampling streams  | Implemented | Selected Record types and activity windows  |
| Laps              | Implemented | WorkoutEvent lap/segment boundaries         |
| Sleep             | Implemented | SleepAnalysis sessionization and stages     |
| Daily summaries   | Implemented | ActivitySummary rings                       |
| Daily metrics     | Implemented | Source-priority and interval-union reducers |
| Bounded Shortcut intake | Implemented | Strict versioned payload, coverage assertions, replay-safe import |

## Acceptance boundary

- Full exports retain complete-snapshot reconciliation semantics.
- Shortcut payloads retain raw evidence and create source coverage assertions in the same transaction as canonical observations.
- Duplicate evidence is skipped at the import-job boundary; changed or late bounded windows create a new snapshot without deleting records outside the bounded evidence.
- A seven-day physical-device trial (locked phone, missed day, offline delivery, and permission/empty-state behavior) is still required before calling automatic Health sync operationally accepted.
