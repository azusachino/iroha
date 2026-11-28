# Constraints

Last reviewed: 2026-11-12

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
| E5 | test-deleted | apps/iroha-public-site/** | Consolidate public site into iroha-web; test preserved in apps/iroha-web/src/lib/public/ | haru | 2026-12-08 |
| E6 | assertion-removed | scripts/dev_watch_test.py | Subtract retired apps/iroha-public-site fixture | haru | 2026-12-08 |
| E7 | assertion-removed | scripts/local_checks_test.py | Subtract retired apps/iroha-public-site fixture | haru | 2026-12-08 |
| E8 | test-deleted | scripts/public_site_response_check_test.py | Retire public-site Caddy fixture checker with Caddyfile.public-site | haru | 2026-12-08 |
| E9 | assertion-removed | apps/iroha-imports/service_test.go | Retire titleLanguageRank: ADR-0012 replaces script ranking with the i18n work-title projection (covered by apps/iroha-core/i18n/media_titles_test.go) | haru | 2026-12-08 |
| E10 | assertion-removed | apps/iroha-web/e2e/public-reading.spec.ts | Owner requested completed-only book archive; replace old in-progress and grouped-list assertions with finished-only, year, chart and undated-record checks | haru | 2026-12-12 |
| E11 | test-deleted | apps/iroha-web/src/lib/themes/registry.test.ts | Remove apps/iroha-web/src/lib/themes/ migration-debt layer; test preserved in apps/iroha-web/src/lib/theme-registry.test.ts | haru | 2026-12-25 |
| E12 | assertion-removed | scripts/check_theme_boundary_test.py | Retire apps/iroha-web/src/lib/themes/ host-adapter allowlist; replace allowlist assertion with rejection assertion | haru | 2026-12-25 |
| E13 | assertion-removed | apps/iroha-web/e2e/chrome-frame.spec.ts | Metric card controls moved to Expense spending panel and ledger renamed to Expense records (issue #236); assertions target the new regions | haru | 2026-12-31 |
| E14 | assertion-removed | apps/iroha-web/e2e/chrome-header.spec.ts | Overview subtitle and Expenses error/updating regions moved to Expense records read and numeric month select (issue #236) | haru | 2026-12-31 |
| E15 | assertion-removed | apps/iroha-web/e2e/chrome-summary.spec.ts | Six-tile summary and Route footprint replaced by focal band, Across the archive, and Monthly distance chart (issue #236); assertions target new regions and assert Route footprint absence | haru | 2026-12-31 |
| E16 | assertion-removed | apps/iroha-web/e2e/exact-table-scroll.spec.ts | Separate scroll-focus buttons replaced by directly focusable tabindex=0 MetricTable scroll regions (issue #236) | haru | 2026-12-31 |
| E17 | assertion-removed | apps/iroha-web/e2e/expenses-states.spec.ts | Spending tiles and verbose period banners replaced by Spending summary focal band, panel period labels, and combobox filter state (issue #236) | haru | 2026-12-31 |
| E18 | assertion-removed | apps/iroha-web/e2e/format.spec.ts | Overview stat-grid and Library items tile replaced by focal-band and inline empty-state summary (issue #236) | haru | 2026-12-31 |
| E19 | assertion-removed | apps/iroha-web/e2e/motion-states.spec.ts | Filtered activity summary and prefixed period labels replaced by Motion summary and Activity records regions with colon duration format (issue #236) | haru | 2026-12-31 |
| E20 | assertion-removed | apps/iroha-web/e2e/night-pagers.spec.ts | Separate scroll-focus button replaced by directly focusable tabindex=0 scroll region with 40px ArrowRight step (issue #236) | haru | 2026-12-31 |
| E21 | assertion-removed | apps/iroha-web/e2e/overview-compact.spec.ts | Six-tile compact grid replaced by Last 12 months focal band and updated action links (issue #236) | haru | 2026-12-31 |
| E22 | assertion-removed | apps/iroha-web/e2e/pilot-equivalence.spec.ts | Standalone duration text moved into Last 12 months focal-primary-detail (issue #236) | haru | 2026-12-31 |
| E23 | assertion-removed | apps/iroha-web/e2e/today-recovery.spec.ts | Today recovery error banner includes retry guidance alongside unavailable message (issue #236); assertion matches substring | haru | 2026-12-31 |
| E25 | assertion-removed | apps/iroha-web/e2e/private-map-worker.spec.ts | Prettier 4.2 collapses a multi-line assertion onto one line (issue #237 formatting repair); the assertion is unchanged | haru | 2026-12-31 |
