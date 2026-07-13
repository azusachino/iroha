# Apple Health provider capabilities

Status: implemented, migration to observation-backed canonical storage
planned.

## Evidence

Apple Health is ingested as a full export ZIP containing `export.xml` and route
files. The export is treated as a complete snapshot and reconciled by stable
provider source identity plus content hash.

## Implemented capabilities

| Capability | Status | Notes |
| --- | --- | --- |
| Activity sessions | Implemented | Workout records and summary metrics |
| Routes | Implemented | WorkoutRoute/FileReference GPX linking |
| Sampling streams | Implemented | Selected Record types and activity windows |
| Laps | Implemented | WorkoutEvent lap/segment boundaries |
| Sleep | Implemented | SleepAnalysis sessionization and stages |
| Daily summaries | Implemented | ActivitySummary rings |
| Daily metrics | Implemented | Source-priority and interval-union reducers |

## Migration requirements

- replace `Parsed*` output names with observation contracts;
- replace `tb_apple_source_items` as the long-term identity boundary;
- backfill existing Apple rows into provider observations;
- preserve real-export counts and reprocess idempotence.
