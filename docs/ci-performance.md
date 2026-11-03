# CI performance round

Baseline: latest 50 runs sampled via `gh`; 112 successful jobs.
Successful-job medians: browser shard 1 619.5s, shard 2 505s,
validate/build/integration 220s. Go caches already existed; the
browser shards reinstalled pinned Chromium with system dependencies on
every run.

## Change ([PR #168](https://github.com/azusachino/iroha/pull/168))

Added a `bun.lock`-keyed `~/.cache/ms-playwright` cache to both
browser shards and the public-site browser job. Replaced the
shared-ref concurrency group (whose `cancel-in-progress` could cancel
a running main check; recent main history shows cancelled runs) with
PR-number/run-id groups that cancel only superseded PR runs and
preserve every main commit. The required `make check` aggregate, the
paths filters, Go caching and the shard split are unchanged. Local
`make check` passed in an isolated worktree before push; one
pre-existing shellcheck style note (SC2129) remains untouched.

## Measurements

Revision `6457c5b`, [PR run 41620822457](https://github.com/azusachino/iroha/actions/runs/41620822457):

| Lane | Baseline median | Cold (attempt 1) | Warm (attempt 2) |
| --- | --- | --- | --- |
| Browser shard 1 | 619.5s | 729s | 609s |
| Browser shard 2 | 505s | 628s | **418s** |
| Validate, build and integration | 220s | 374s | **166s** |

Warm verdict: mixed. Shard 2 improved **17.2%** and the validate lane
**24.5%** below their medians, but shard 1 landed within noise (−1.7%)
— the browser download the cache removes is evidently a small share of
that shard's total, which the two shards' differing medians already
suggested. Cold attempts ran above median (first-run population plus
hosted variance). One cold/warm pair; the cache is kept because it is
input-keyed, cost is a few seconds when it hits, and two of three
lanes improved, but this is a modest, lane-dependent saving rather
than a uniform speedup.
