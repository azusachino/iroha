# Architecture decision records

Statuses below are the decisions recorded in each ADR, not inferred approval from the presence of code. Implementation and release evidence remain in the linked records and [release notes](../../CHANGELOG.md).

| ADR  | Decision                                                                                                  | Recorded status                                              |
| ---- | --------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| 0001 | [Provider observations and canonical domain records](0001-provider-observations-and-canonical-records.md) | Proposed                                                     |
| 0002 | [Provider adapter contracts](0002-provider-adapter-contracts.md)                                          | Proposed                                                     |
| 0003 | [Cache backends and invalidation](0003-cache-backends-and-invalidation.md)                                | Accepted; freshness extended by 0004                         |
| 0004 | [Cache correctness and report reads](0004-cache-correctness-and-report-reads.md)                          | Accepted                                                     |
| 0005 | [Media provider time semantics](0005-media-provider-time-semantics.md)                                    | Accepted; implemented in v0.4.1                              |
| 0006 | [Background-first reconciliation](0006-background-first-reconciliation.md)                                | Accepted                                                     |
| 0007 | [Bound v0.5 Health acceptance to available evidence](0007-v05-health-acceptance-boundary.md)              | Accepted                                                     |
| 0008 | [Health Auto Export for automated daily Health intake](0008-health-auto-export-http-intake.md)            | Accepted; authentication decision updated                    |
| 0009 | [Native Bun workspace and stable project tools](0009-native-frontend-workspace.md)                        | Accepted by owner; supersedes unmerged pnpm proposal in #117 |
| 0010 | [Encoded public snapshots and exact-byte validators](0010-public-snapshot-validators.md)                  | Proposed; PR #125 requires independent contract verification |

Keep numbering stable. Extend or supersede a decision explicitly; do not silently promote proposed ADRs while updating this index.
