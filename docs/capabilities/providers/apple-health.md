# Apple Health provider capabilities

Status: full-export ingestion implemented end to end. Automated daily intake is implemented end-to-end via [Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) (HAE)
Format v2 JSON at `POST /api/v1/intake/health`.

## Evidence

Apple Health is ingested via two tiers:

1. **Continuous Daily Intake**: Automated background sync via Health Auto Export sending Format v2 JSON to `POST /api/v1/intake/health`. Every request requires the dedicated intake credential.
   Payloads are ingested using `bounded_replacement` mode.
2. **Historical Bulk Backfill**: Full export ZIP containing `export.xml` and route files uploaded via `POST /api/v1/raw-files` and `POST /api/v1/imports`. The export is treated as a complete snapshot
   and reconciled by stable provider source identity plus content hash.

See [Setting up Health Auto Export HTTP Intake](../../health-auto-export-setup.md) for client configuration and export settings, and [ADR-0008](../../adr/0008-health-auto-export-http-intake.md) for
the architecture decision.

## Implemented capabilities

| Capability           | Status      | Notes                                                                                          |
| -------------------- | ----------- | ---------------------------------------------------------------------------------------------- |
| Activity sessions    | Implemented | Workout records, summary metrics, and GPS routes (from HAE and full export)                    |
| Routes               | Implemented | WorkoutRoute/FileReference GPX linking and HAE coordinate stream points                        |
| Sampling streams     | Implemented | Heart rate samplings and time series from HAE workouts and export records                      |
| Laps                 | Implemented | WorkoutEvent lap/segment boundaries                                                            |
| Sleep                | Implemented | SleepAnalysis sessionization, durations, and stage segments                                    |
| Daily summaries      | Implemented | ActivitySummary rings (full export); Move/Exercise/Stand (HAE when all three metrics are selected) |
| Daily metrics        | Implemented | Steps, distance, resting/walking HR, HRV, flights, VO2 max, body mass, SpO2, respiratory rate (HAE and full export) |
| Automated HAE intake | Implemented | Sparse Format v2 JSON parser, 10MB body limit, Tailnet perimeter security, bounded replacement |

## Acceptance boundary

- Full exports retain complete-snapshot reconciliation semantics.
- HAE payloads retain raw evidence and create source coverage assertions in the same transaction as canonical observations.
- Duplicate evidence is skipped at the import-job boundary; changed or late bounded windows create a new snapshot without deleting records outside the bounded evidence.
- Coverage completeness is recorded evidence, not a replacement instruction. `bounded_replacement` updates observations in the bounded window without deleting records outside it.
- `unknown` completeness is accepted end to end -- defensive default for background mobile sync. HAE remains `unknown` when no unsupported metric is seen and `partial` when the upload contains unmapped metric names; neither asserts complete Health-category coverage.
- A full Apple Health export asserts `covered` only over the calendar days represented by dated observations; an export with no dated records asserts no coverage.
- Media current-state lists assert `covered`/`full_snapshot` only after pagination completes; activity feeds assert `covered`/`incremental` over their fetched window. A failed sync with a stored page records `partial` coverage.
- Production setup uses Health Auto Export's REST API automation; see [the HAE reference](health-auto-export.md) and [the setup guide](../../health-auto-export-setup.md).
