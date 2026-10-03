# Dependency and loading cleanup

Owner-approved scope: remove unused/unnecessary dependencies, incorporate coverage PR #91, and investigate heavy sparsely-used dependencies with packaging/loading UX priority. No deployment, release, Node/TypeScript major upgrade, quality-floor changes or live-data probes.

Baseline: main `6612293c6f70f3b6d76c4506406fde328eb1ece7`. Branch: `chore/dependency-cleanup-table`. Workstation tier 1; A1–A7 independently accepted at runtime/test revision `567940ab3868a7dc43f0ef38bdd6b02e18600d52`. [PR #109](https://github.com/azusachino/iroha/pull/109) owns delivered review; CI status remains in the PR.

## Decisions and acceptance

- A1: Remove unused `three` from private/shared, `@types/three` from shared, and `uplot` from private. Regenerate locks without unrelated version updates. No source imports exist for those packages.
- A2: Owner explicitly chose removing TanStack rather than upgrading to stable 9.2.4. Preserve header labels, initial descending date order, first-row-type toggle defaults, Shift state, leading-key-only row ordering, year/sport filters, progressive rows and detail links. Correct stale header arrows and expose the actual leading sort with `aria-sort`.
- A3: Incorporate coverage 7.16.2. Owner explicitly approved this version before the seven-day age window elapsed; preserve general `P7D` lock metadata and leave no lasting package exemption. Coverage floors stay unchanged. `uv lock --check` must pass.
- A4: Defer the public map engine until a map mounts. No-route public landing must not request it. Delayed or failed library loading must leave charts, activity records and route summaries usable, with explicit loading/error feedback.
- A5: Package MapLibre's worker explicitly for both hosts. A canvas alone is insufficient: public and private overview maps remain busy until their GeoJSON source reports loaded or a terminal map error is displayed. Verify actual worker/source loading, explicit failure feedback with usable route summaries, deep links, back navigation and successful recovery after an explicit page reload. Private motion-detail playback is not a new runtime acceptance claim.
- A6: Preserve sanitized public-only requests, existing map styling/attribution/controls, shared chart runtime deduplication and existing test assertions/thresholds. No replacement map/chart dependency is introduced.
- A7: Run owning validation, full browser suite, integration, production-focused browser checks, and fresh independent verification. Report actual bundle changes and limitations, not speculative transfer/latency gains.

## Findings

Read-only advisor `iroha-dep-advisor`, Sonnet 5.5 / medium, session `d992c052-1b6e-4772-a4d7-485c3a145f23`, independently traced source imports. Lead compared emitted artifacts and browser behavior.

| Dependency | Decision | Evidence / boundary |
| --- | --- | --- |
| Three.js / its types | Remove | No runtime/type imports in private/public/shared source; leftover declarations. Removal cleans installs, not emitted JS. |
| uPlot | Remove | No source imports; README's line-chart claim was obsolete. |
| TanStack Table | Replace with typed Svelte state | Public page already sorted and paged arrays manually; library only supplied header/toggle state. The retained baseline test exposed stale arrows after actual keyboard sorting. |
| ECharts | Retain | Shared bar/line/pie/fused charts, tooltip/legend/theme/motion/data contracts; existing modular imports. A home-made replacement would recreate substantial verified behavior. |
| MapLibre | Retain, defer public loading, repair worker packaging | Raster basemap, GeoJSON routes, bounds, zoom/pan, markers/playback and attribution. Heavy engine cost warrants a bounded alternative prototype. |
| Lucide | Retain | Named imports across 87 source sites; upstream `sideEffects: false`. No measured reason to maintain copied SVGs. |
| Playwright / test package / coverage / formatters / Svelte / Tailwind / Vite | Retain | Actual scripts, configs, tests and CSS/build consumers. Build-only dependencies are not initial browser payload. |
| Node types | Retain for now | Tool/config type contracts; no runtime payload benefit from removal. Do not mistake absence of app runtime imports for unused tooling types. |
| Python direct dependencies | Retain | `requests` in CLI/auth/smoke scripts, `psycopg` in media bridge, coverage in gates. |
| Go direct dependencies | No demonstrated removal | Advisor found import evidence across owning modules, including test-only WebAuthn helpers. Not a full transitive/security audit or `go mod tidy` result. |

### Loading bugs found during implementation

The initial map tests exposed MapLibre's indirect default worker URL resolving to a missing worker in Vite's optimized dependency directory. Production output also lacked an emitted worker. Host-local build adapters now use Vite's explicit `?worker&url` asset and MapLibre's `setWorkerUrl` API; both emitted clients include the worker.

Chromium retained a failed dynamic import, so a same-document “Retry map” could not recover the network-failed module. The UI instead offers the documented, explicit “Reload page” action. That reload preserves the deep-link URL, but ordinary in-memory year/sport/city selections reset as with any page reload; it is not advertised as a quiet data refresh.

The first tile test fixture also contained an invalid PNG. It was replaced with a valid synthetic image; external OSM tiles are not used as acceptance evidence.

Sources: [Vite worker URL imports](https://vite.dev/guide/features.html#import-with-query-suffixes), [Vite load-error handling](https://vite.dev/guide/build.html#load-error-handling), installed MapLibre 6.11.2 declarations (`setWorkerUrl`), [TanStack v8 toggle implementation](https://github.com/TanStack/table/blob/v8.21.3/packages/table-core/src/features/RowSorting.ts), [ECharts modular imports](https://echarts.apache.org/handbook/en/basics/import/).

## Emitted packaging measurements

Same pinned tools, local production builds. The table records lead captures at the initial implementation; filenames/compression can drift slightly between builds and review fixes. Raw bytes and gzip of each emitted file with `mtime=0`; static route graph follows Vite manifest `imports`, excluding `dynamicImports` and non-JavaScript files. These are artifact costs, not measured network transfer, cache behavior, FCP/LCP or production latency. The independent verifier did not rebuild the baseline; the percentage comparison is lead evidence, not independently reproduced. Its graph walk included additional files and is not the same measurement definition.

| Measurement | Baseline raw / gzip bytes | Updated raw / gzip bytes |
| --- | --- | --- |
| Public landing static JS graph | 1,745,375 / 511,521 | 676,541 / 228,740 |
| Public aggregate emitted JS | 1,752,670 / 515,281 | 2,255,462 / 657,213 |
| Private aggregate emitted JS | 2,107,711 / 647,187 | 2,617,525 / 791,393 |
| Private aggregate emitted CSS | 292,510 / 56,198 | 292,510 / 56,198 |

The public static landing graph falls about 61% raw / 55% gzip. Aggregate output grows because the previously missing functional worker is now emitted (509,702 raw / 144,167 gzip per host), plus chunk separation overhead. Worker bytes are fetched when a map is initialized, not on no-route public landing. Private aggregate JS and CSS values exceed the existing report-only bundle baselines; this is visible, not a lowered threshold or a claim of an aggregate reduction. CSS was already over its 280,729 / 54,193 baseline in the captured pre-change artifacts and did not grow in this comparison. At this baseline, private overview still statically imported its map engine; PR #115's follow-up below addresses that loading cost.

Before map changes, native table replacement alone saved 46,071 raw / 11,767 gzip bytes in public aggregate JS. Unused Three.js/uPlot removal did not reduce private emitted JS; minor hash/compression variation is not a meaningful saving.

## Replacement priorities

1. Prototype Leaflet against the actual raster/GeoJSON/bounds/marker/reduced-motion/keyboard/attribution contracts before considering a map-engine swap. Official stable registry version observed: 1.9.4. Integrity-verified upstream `dist/leaflet.js` is 147,552 raw / 42,434 gzip bytes; CSS is 14,806 / 3,522. Those are upstream artifact sizes, not a measured Iroha replacement bundle or a proven drop-in. [Official GeoJSON API](https://leafletjs.com/reference.html#geojson), [bounds API](https://leafletjs.com/reference.html#map-fitbounds).
2. Defer private overview map imports and public activity-detail-specific code where emitted graphs show benefit, retaining truthful loading/error and exact-record access.
3. Retain ECharts until a bounded simple-chart prototype proves equivalent unknown-value handling, provenance, theme, reduced motion, resolved paint and controls. Do not replace it with uPlot just because uPlot was declared.
4. Do not craft a tile engine, chart framework or copied icon library without evidence that the reduced payload outweighs maintenance/accessibility/interaction costs.

## Verification record

- Initial cleanup: `make check` exit 0; native sorting browser checks 2/2.
- Deferred-map and sorting checks: 6/6 against dev servers and 6/6 against both production preview servers. Both production preview process groups were task-owned, terminated, and ports verified unbound.
- Public type check/build passes after correcting the lazy module's TypeScript namespace to its host adapter.
- Fresh independent verifier `iroha-dep-verifier`, Sonnet 5.5 / medium, session `dd591960-5ed0-482a-bb4d-668b717d3d00`, reviewed clean `1e1e5651139406868fadbf8c5b64f9a88c988907`. Validation, integration, full browser 195/195 and production-focused 6/6 passed. The terminal restart interrupted report retrieval; the same verifier session resumed without a model or protocol fallback.
- Initial review found missing private-worker runtime evidence, CSS overage disclosure, measurement limits, and busy-without-error feedback on worker failure. Private overview now has a retained worker/source-ready regression; both overview and public maps display terminal map errors and clear busy while keeping route summaries accessible. CSS and measurement limits are disclosed above.
- Same independent verifier's exact-head recheck at `567940ab3868a7dc43f0ef38bdd6b02e18600d52` accepted A1–A7: `make validate` exit 0 (Vitest 221/221), full `CI=1 make e2e` exit 0 (198/198), both-host production-focused browser exit 0 (9/9), and disposable-DB integration exit 0 (35 `ok`, no failures). Source/index remained clean and task-owned previews were stopped with ports unbound.
- [Promoted report](https://github.com/azusachino/iroha/pull/109#issuecomment-5964933027). Worker/runtime, error feedback, CSS disclosure and measurement-definition findings are resolved within acceptance. Independent final private JS capture was 2,618,039 raw / 791,636 gzip; exact capture drift does not change the report-only overage or static-graph caveat.
- Non-blocking source-derived concern: a single tile failure may trigger the generic terminal “map unavailable” message. This isolated case was not reproduced during review; [issue #110](https://github.com/azusachino/iroha/issues/110) owns its reproduction and precise feedback acceptance. Private motion-detail playback, baseline rebuild and Go transitive audit remain outside this verification claim.

Retained specs: `public-table-sorting.spec.ts`, `public-map-loading.spec.ts`, `private-map-worker.spec.ts`. Baseline, failed probes, build logs and advisor transcript remain in workstation scratch `.tmp/iroha-dependency-cleanup/`; the accepted report is promoted to the owning PR above.

Replacement PR will supersede #91, #92, #94 and #97 only once merged. Bot PRs, TS7 PRs and Node-types PRs remain untouched in this slice.

## PR #115: bounded loading follow-up

The owner requested this work in the same PR as CI optimization. Private overview
now imports the existing map adapter only when a route map mounts, following the
public host's established lifecycle. Its loading/library/worker failure paths keep
route summaries accessible; source-ready and partial-tile feedback remain distinct.
No engine or worker packaging change is included.

Public landing defers its approved-rich/summary activity-detail adapter behind a
native Svelte await/import block. The selected title, distance/duration and Back
link remain available while its module is pending or failed; failure offers an
explicit keyboard-operable page reload. Closing the detail cancels the rendering
branch, so late imports cannot bring it back. The API projection and existing
per-record fetch/fallback behavior are unchanged. This does not claim new API
failure UX or lazy chart loading across every route.

Lead artifact captures use the same pinned tools: the unchanged `11fd490` outputs
from its prior validation versus the follow-up build. The graph starts at private
overview node 19 or public landing node 2 and recursively follows manifest static
`imports`, excluding dynamic imports and non-JavaScript assets. Gzip is per file
with `mtime=0`; layout/bootstrap, CSS, dynamic worker and network timing are outside
this definition. Compression uses gzip level 9; rebuild filenames/compressor
variation can shift captures by a few gzip bytes. No aggregate-output or FCP/LCP
saving is inferred.

| Static route JS graph | Before raw / gzip bytes | After raw / gzip bytes |
| --- | --- | --- |
| Private overview | 1,822,531 / 543,029 | 788,299 / 268,569 |
| Public landing | 676,768 / 228,820 | 660,891 / 225,428 |

Private static gzip cost falls about 51%; public detail deferral saves 3,392 gzip
bytes (about 1.5%). These lead captures do not prove baseline reproduction by an
independent agent; the final owning PR records acceptance and measurement limits.
Retained browser specs cover no-engine/detail requests on no-route landing,
deferred and failed imports, reload recovery, detail deep links/record clicks,
back navigation during a pending import, approved detail and summary fallback,
and unchanged worker/GeoJSON-ready/partial-tile contracts.

Old #79/#85 task reconciliation is recorded in [issue #116](https://github.com/azusachino/iroha/issues/116).
Closed parent issues do not prove every child accepted. The absent full cockpit
audit, shared-coverage proof, backup/restore sequencing, release provenance and
public snapshot/ETag evidence remain explicit follow-ups. Existing task owners
are preserved; no historical failure is asserted to remain a current defect.
No live operation, re-baseline, edge policy or release is authorized by this PR.
