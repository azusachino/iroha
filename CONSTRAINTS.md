# Constraints

Last reviewed: 2026-09-29

This is Iroha's quality contract. It covers the Go workspace, `scripts/`, the private web app and its shared UI package, and the local seeded runtime checks described below. Public-site coverage, production SLOs, screen-reader testing, and deployment-volume verification are not established by these baselines.

## Floor

- Add no checker suppressions, unfinished stubs, skipped tests, or unreviewed exceptions.
- Keep test assertions and quality thresholds intact. Tightening is welcome; weakening requires a tracked exception with an owner and expiry.
- Keep secrets out of tracked source and reports. Secret scanning is not currently an automated Iroha gate.
- `make quality-floor-check` enforces the diff-scoped rules above against `QUALITY_BASE` (default `origin/main`). It fails closed if Git cannot establish the base. CI fetches full history and supplies its target branch.

## Blocking checks

`make check` remains the blocking local and CI gate. It runs Go formatting, vet, lint and tests; API contract and Python script tests; theme, responsive and motion checks; web formatting, type checking and Vitest tests. Go, Python and web coverage are collected during those test runs, not by a second test pass.

| Dimension | Floor | Checked by |
| --- | --- | --- |
| Go statement coverage | >= 30.84% | `make test` (`go test -coverprofile` across `go.work`) |
| Python combined statement/branch coverage | >= 36.66% | `make scripts-test` (`coverage run --branch`, excluding test files) |
| Web statements | >= 10.34% | `make web-test` (`vitest run --coverage`, app and shared UI) |
| Web branches | >= 12.21% | `make web-test` (`vitest run --coverage`, app and shared UI) |
| Web functions | >= 8.60% | `make web-test` (`vitest run --coverage`, app and shared UI) |
| Web lines | >= 10.30% | `make web-test` (`vitest run --coverage`, app and shared UI) |

The floors use the measured baseline minus 0.5 percentage points to absorb small denominator changes. The initial baselines were Go 31.34% (3,211/10,246 statements), Python 37.16% combined coverage (971/2,553 statements and 289/838 branch outcomes), and web 10.84% statements, 12.71% branches, 9.10% functions, and 10.80% lines. The final 104-test Python run reports 37.30% after the new quality-tool tests were added; its floor remains anchored to the initial baseline. Python's combined percentage is the `coverage.py` total with branch measurement enabled. Web coverage includes all `iroha-web` and `iroha-shared` TypeScript/Svelte source; untested routes and components count as uncovered. Go excludes integration-tagged tests. These are starting floors, not aspirational coverage targets.

The quality-floor guard detects added suppressions, stubs and skips; deleted test files and removed assertions; removed or weakened `CONSTRAINTS.md` rules; and new exception rows. It reports rule and file only, not matched text. It is a diff guard, not a secret scanner or substitute for review.

## Measured, report-only baselines

These checks are outside blocking `make check`. Their initial baselines are for calibration through 2026-10-13; keep their reports visible and review regressions before promoting any to a blocking gate.

| Dimension | Baseline and direction | Checked by / limits |
| --- | --- | --- |
| Grapher route audit states | >= 250 states | Seeded `make web-mobile-check` over 25 routes, five widths (320, 375, 414, 768, 1280), light and dark. |
| Grapher browser-contract failures | <= 0 failures | Same seeded Chromium audit. This is a custom layout/accessibility contract, not an axe, WCAG-conformance, or screen-reader audit. |
| Focus contrast | >= 5.04:1 | Same audit's computed focus-token contrast measurement. |
| JavaScript raw assets | <= 2096109 bytes | `make web-bundle-report`; aggregate only, not route-level transfer. |
| JavaScript gzip assets | <= 643063 bytes | `make web-bundle-report`; sum of individually compressed emitted files. |
| CSS raw assets | <= 280729 bytes | `make web-bundle-report`. |
| CSS gzip assets | <= 54193 bytes | `make web-bundle-report`; sum of individually compressed emitted files. |
| API cold p95 | <= 13.33 ms | `make release-candidate` against its small seeded local fixture. Not a production SLO and does not authorize testing a deployed instance. |
| API cold p50 | <= 1.26 ms | Same local fixture; comparison point only. |
| API cache-hit p50 | <= 0.71 ms | Same local fixture; comparison point only. |

The browser route checker and release-candidate fixture require a local seeded stack; they are not run by CI's `make check`. `make web-bundle-report` reports aggregate sizes but does not compare them with the table or fail on growth. These gaps are explicit: report-only values are not CI warnings until a follow-up wires reproducible comparison into the relevant lifecycle.

## Exceptions

Record an exception here only with a named owner, reason, and expiry. The floor guard flags new `W<n>` or `E<n>` rows for review.

| ID | Rule | Scope | Reason | Owner | Expires |
| --- | --- | --- | --- | --- | --- |
