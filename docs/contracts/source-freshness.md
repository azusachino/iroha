# Source delivery freshness

`GET /api/v1/connections` is the authenticated rollup shared by Admin Sources and external homelab clients. It sends `Cache-Control: no-store`, evaluates freshness at request time, and propagates request cancellation into its database queries. No new public endpoint or bot credential is introduced.

For each registered source:

- `last_receipt.received_at` is the last delivery time, not the newest measurement time or a proof of successful interpretation.
- `expected_interval_s` and `next_expected_at` appear only when an active delivery policy exists. Interval seconds are a positive number, preserving fractional and subsecond worker intervals. The deadline is last receipt plus interval; before the first receipt, it is source creation plus interval.
- `freshness` becomes `overdue` at the deadline, even with no receipt. Before the first deadline it is `unknown`; a received source before its deadline is `within_cadence`.
- Active Health Auto Export credentials use a 24-hour delivery expectation. Revoked credentials stop that expectation unless another active credential with the same device name remains.
- AniList, AniList activity and Bangumi use their worker's enabled interval schedule, including custom intervals. Disabled/absent schedules and manual sources are `not_scheduled`; an uninterpretable enabled schedule is `unknown`.
- Collection coverage, import failure and delivery freshness remain separate. A newly received payload can still fail parsing or have unknown coverage. Failed/importing badges do not hide an overdue delivery note.

Admin shows the last receipt and next expected delivery, counts failed or overdue sources once each, and refreshes once per minute while the Sources panel is mounted. A refresh failure preserves the last successful rows and shows an error; it does not declare the source healthy. Manual refresh remains available.

Homelab clients can poll the same authenticated endpoint at their agreed cadence, select connections whose `freshness` is `overdue` or whose `operation` is `failed`, and use timestamps for their own notification deduplication. This change does not install a notification scheduler or authorize production alerting changes. Sources must already be registered in `tb_source_instances`; issuing a credential without any registered source is not itself monitored. The 24-hour HAE policy is an expectation, not a guarantee of iOS background delivery.

Verification: deadline-boundary unit tests; disposable PostGIS tests for fresh/overdue/never-delivered/revoked/custom/disabled/invalid cadence and cancellation; `make e2e` for periodic refresh, coverage separation, manual-source copy and error retry at 320px; `make validate` for both frontend consumers and the backend.
