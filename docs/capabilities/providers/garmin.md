# Garmin provider capabilities

Status: planned.

Garmin must add provider observations, not a second canonical activity or sleep model. Candidate evidence includes FIT, TCX, CSV/JSON exports, and future connector snapshots; the exact intake formats
are implementation decisions.

## Planned capabilities

| Capability        | Status  | Initial expectation                          |
| ----------------- | ------- | -------------------------------------------- |
| Activity sessions | Planned | Garmin activity identity and summary metrics |
| Routes            | Planned | FIT/TCX track points                         |
| Sampling streams  | Planned | Heart rate, cadence, power where present     |
| Laps              | Planned | Provider lap records                         |
| Sleep             | Planned | Device-reported episode and stages           |
| Daily summaries   | Planned | Source-specific daily totals                 |
| Daily metrics     | Planned | Explicit reducer/source-selection rules      |

## Required before implementation

- stable Garmin source-key strategy;
- timezone and unit normalization;
- snapshot versus incremental sync semantics;
- device identity handling;
- overlap/matching policy with Apple observations;
- fixtures and real import verification.
